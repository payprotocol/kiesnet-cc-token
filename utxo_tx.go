// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/contract"
	"github.com/key-inside/kiesnet-ccpkg/kid"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// params[0] : sender's address
// params[1] : positive amount: receiver's address who gets paid. negative amount: Original chunk key.
// params[2] : amount. can be either positive(pay) or negative(refund) amount.
// params[3] : optional. memo (max 128 charactors)
// params[4] : pending time (time represented by int64 seconds)
// params[5] : expiry (duration represented by int64 seconds, multi-sig only)
func pay(stub shim.ChaincodeStubInterface, params []string) peer.Response {

	if len(params) < 3 {
		return shim.Error("incorrect number of parameters. expecting at least 3")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if nil != err {
		return shim.Error(err.Error())
	}

	//sender address validation
	var sAddr *Address
	sAddr, err = ParseAddress(params[0])
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to parse the sender's account address")
	}

	// amount
	amount, err := NewAmount(params[2])
	if nil != err {
		return shim.Error(err.Error())
	}
	if amount.Sign() == 0 {
		return shim.Error("invalid amount. amount shouldn't be 0")
	}

	ub := NewUtxoStub(stub)
	rid := params[1] //receiver's id
	pkey := ""       //parent key(original chunk key). this field is used for tracking the original/parent chunk from the refund chunk

	// refund validation
	if amount.Sign() < 0 {
		ck, err := ub.GetChunk(params[1]) //if negative amount, param[1] must be the original chunk key
		if err != nil {
			return shim.Error("invalid chunk key was provided.")
		}

		rid = ck.RID     //getting the user's id from the original chunk
		pkey = params[1] //this parent key is written on the refund chunk created later.

		amountCmp := amount.Copy()
		amountCmp.Neg()
		if ck.Amount.Cmp(amountCmp) < 0 {
			return shim.Error("refund amount can't be greater than the pay amount")
		}

		totalRefundAmount, err := ub.GetTotalRefundAmount(params[0], pkey)

		if err != nil {
			return shim.Error("failed to get the total refund amount")
		}

		if ck.Amount.Cmp(totalRefundAmount.Add(amountCmp)) < 0 {
			return shim.Error("can't exceed the sum of past refund amounts")
		}
	}

	// receiver address validation
	rAddr, err := ParseAddress(rid)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to parse the receiver's account address")
	}

	if rAddr.Code != sAddr.Code { // not same token
		return shim.Error("different token accounts")
	}

	// IMPORTANT: assert(sender != receiver)
	if sAddr.Equal(rAddr) {
		return shim.Error("can't pay to self")
	}

	ab := NewAccountStub(stub, rAddr.Code)

	// sender account validation
	sender, err := ab.GetAccount(sAddr)
	if nil != err {
		logger.Debug(err.Error())
		return shim.Error("failed to get the sender account")
	}
	if !sender.HasHolder(kid) {
		return shim.Error("invoker is not holder")
	}
	if sender.IsSuspended() {
		return shim.Error("the sender account is suspended")
	}

	// receiver account validation
	receiver, err := ab.GetAccount(rAddr)
	if nil != err {
		logger.Debug(err.Error())
		return shim.Error("failed to get the receiver account")
	}
	if receiver.IsSuspended() {
		return shim.Error("the receiver account is suspended")
	}

	// sender balance
	bb := NewBalanceStub(stub)
	sBal, err := bb.GetBalance(sender.GetID())
	if nil != err {
		logger.Debug(err.Error())
		return shim.Error("failed to get the sender's balance")
	}
	if sBal.Amount.Cmp(amount) < 0 {
		return shim.Error("not enough balance")
	}

	// receiver balance
	rBal, err := bb.GetBalance(receiver.GetID())
	if nil != err {
		logger.Debug(err.Error())
		return shim.Error("failed to get the receiver's balance")
	}

	// options
	memo := ""
	var pendingTime *txtime.Time
	var expiry int64
	signers := stringset.New(kid)
	if a, ok := sender.(*JointAccount); ok {
		signers.AppendSet(a.Holders)
	}
	// memo
	if len(params) > 3 {
		if len(params[3]) > 128 { // 128 charactors limit
			memo = params[3][:128]
		} else {
			memo = params[3]
		}
		// pending time
		if len(params) > 4 {
			seconds, err := strconv.ParseInt(params[4], 10, 64)
			if err != nil {
				return shim.Error("invalid pending time: need seconds since 1970")
			}
			ts, err := stub.GetTxTimestamp()
			if err != nil {
				return shim.Error("failed to get the timestamp")
			}
			if ts.GetSeconds() < seconds { // meaning pending time
				pendingTime = txtime.Unix(seconds, 0)
			}
			// expiry
			if len(params) > 5 && len(params[5]) > 0 {
				expiry, err = strconv.ParseInt(params[5], 10, 64)
				if err != nil {
					return shim.Error("invalid expiry: need seconds")
				}
			}
		}
	}

	var log *BalanceLog // log for response

	if signers.Size() > 1 {
		if signers.Size() > 128 {
			return shim.Error("too many signers")
		}
		// pending balance id
		pbID := stub.GetTxID()
		// contract
		ptStr := "0"
		if pendingTime != nil {
			ptStr = params[4]
		}
		doc := []string{"utxo/pay", pbID, sender.GetID(), receiver.GetID(), amount.String(), memo, ptStr}
		docb, err := json.Marshal(doc)
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to create a contract")
		}
		con, err := contract.CreateContract(stub, docb, expiry, signers)
		if err != nil {
			return shim.Error(err.Error())
		}
		// pending balance
		log, err = bb.Deposit(pbID, sBal, con, *amount, memo)
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to create a pending balance")
		}

	} else {
		log, err = ub.Pay(sBal, rBal, *amount, memo, pkey)
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to pay")
		}

	}

	// log is not nil
	data, err := json.Marshal(log)
	if nil != err {
		logger.Debug(err.Error())
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}

// params[0] : Token Code or Address to prune.
// params[1] : Prune end time. It is not guaranteed that the prune merge all chunks into one in this given period time. If it reaches the threshhold of 500, then it finishes the current action expecting the next call from the client.
func prune(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 2 {
		return shim.Error("incorrect number of parameters. expecting 2 parameters")
	}
	// authentication. get KID of current user.
	kid, err := kid.GetID(stub, true)
	if nil != err {
		return shim.Error(err.Error())
	}

	var addr *Address
	code, err := ValidateTokenCode(params[0])
	if nil == err { // by token code
		addr = NewAddress(code, AccountTypePersonal, kid)
	} else { // by address
		addr, err = ParseAddress(params[0])
		if nil != err {
			return responseError(err, "failed to get the account")
		}
	}

	ab := NewAccountStub(stub, addr.Code)
	// account
	account, err := ab.GetAccount(addr)
	if nil != err {
		logger.Debug(err.Error())
		return shim.Error("failed to get the account")
	}
	if account.IsSuspended() {
		return shim.Error("the account is suspended")
	}

	bb := NewBalanceStub(stub)
	balance, err := bb.GetBalance(account.GetID())
	if nil != err {
		return shim.Error(err.Error())
	}
	ub := NewUtxoStub(stub)

	// start time
	stime := txtime.Unix(0, 0)
	if 0 < len(balance.LastChunkID) {
		lastChunk, err := ub.GetChunk(balance.LastChunkID)
		if nil != err {
			return shim.Error(err.Error())
		}
		stime = lastChunk.CreatedTime
	}
	// end time
	seconds, err := strconv.ParseInt(params[1], 10, 64)
	if nil != err {
		return responseError(err, "failed to parse the end time")
	}
	etime := txtime.Unix(seconds, 0)

	chunkSum, err := ub.GetChunkSumByTime(account.GetID(), stime, etime)
	if nil != err {
		return shim.Error(err.Error())
	}

	ts, err := txtime.GetTime(stub)
	if nil != err {
		return shim.Error(err.Error())
	}
	// Add balance
	balance.Amount.Add(chunkSum.Sum)
	balance.UpdatedTime = ts
	if 0 != len(chunkSum.End) {
		balance.LastChunkID = chunkSum.End
	}

	if err := bb.PutBalance(balance); nil != err {
		return shim.Error(err.Error())
	}

	//get last pruend chunk's time
	lastPrunedChunk, err := ub.GetChunk(chunkSum.End)

	// balance log
	rbl := NewBalanceTransferLog(nil, balance, *chunkSum.Sum, fmt.Sprintf("prune result from %s to %s ", stime.Time.UTC(), lastPrunedChunk.CreatedTime))
	rbl.CreatedTime = ts
	if err = bb.PutBalanceLog(rbl); err != nil {
		return shim.Error(err.Error())
	}

	data, err := json.Marshal(chunkSum)
	if nil != err {
		return shim.Error(err.Error())
	}
	return shim.Success(data)
}

// uxtoList _
// params[0] : token code | account address
// params[1] : bookmark
// params[2] : fetch size (if < 1 => default size, max 200)
// params[3] : start time (time represented by int64 seconds)
// params[4] : end time (time represented by int64 seconds)
func uxtoList(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 1 {
		return shim.Error("incorrect number of parameters. expecting 1+")
	}

	// authentication
	kid, err := kid.GetID(stub, false)
	if err != nil {
		return shim.Error(err.Error())
	}

	bookmark := ""
	fetchSize := 0
	var stime, etime *txtime.Time
	// bookmark
	if len(params) > 1 {
		bookmark = params[1]
		// fetch size
		if len(params) > 2 {
			fetchSize, err = strconv.Atoi(params[2])
			if err != nil {
				return shim.Error("invalid fetch size")
			}
			// start time
			if len(params) > 3 {
				if len(params[3]) > 0 {
					seconds, err := strconv.ParseInt(params[3], 10, 64)
					if err != nil {
						return shim.Error("invalid start time: need seconds since 1970")
					}
					stime = txtime.Unix(seconds, 0)
				}
				// end time
				if len(params) > 4 {
					if len(params[4]) > 0 {
						seconds, err := strconv.ParseInt(params[4], 10, 64)
						if err != nil {
							return shim.Error("invalid end time: need seconds since 1970")
						}
						etime = txtime.Unix(seconds, 0)
						if stime != nil && stime.Cmp(etime) >= 0 {
							return shim.Error("invalid time parameters")
						}
					}
				}
			}
		}
	}

	var addr *Address
	code, err := ValidateTokenCode(params[0])
	if nil == err { // by token code
		addr = NewAddress(code, AccountTypePersonal, kid)
	} else { // by address
		addr, err = ParseAddress(params[0])
		if err != nil {
			return responseError(err, "failed to parse the account address")
		}
	}

	ub := NewUtxoStub(stub)
	res, err := ub.GetUtxoChunksByTime(addr.String(), bookmark, stime, etime, fetchSize)
	if nil != err {
		return responseError(err, "failed to get chunks log")
	}

	data, err := json.Marshal(res)
	if err != nil {
		return responseError(err, "failed to marshal chunks logs")
	}

	return shim.Success(data)

}
