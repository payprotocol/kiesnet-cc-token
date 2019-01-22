// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/kid"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// params[0] : sender address (empty string = personal account)
// params[1] : receiver address
// params[2] : amount (big int string)
// params[3] : memo (max 128 charactors)
// params[4] : expiry (duration represented by int64 seconds, multi-sig only)
// params[5:] : extra signers (personal account addresses)
func pay(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 3 {
		return shim.Error("incorrect number of parameters. expecting 3+")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	// amount
	amount, err := NewAmount(params[2])
	if err != nil {
		return shim.Error(err.Error())
	}
	if amount.Sign() <= 0 {
		return shim.Error("invalid amount. amount must be larger than 0")
	}

	// addresses
	rAddr, err := ParseAddress(params[1])
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to parse the receiver's account address")
	}
	var sAddr *Address
	if len(params[0]) > 0 {
		sAddr, err = ParseAddress(params[0])
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to parse the sender's account address")
		}
		if rAddr.Code != sAddr.Code { // not same token
			return shim.Error("different token accounts")
		}
	} else {
		sAddr = NewAddress(rAddr.Code, AccountTypePersonal, kid)
	}

	// IMPORTANT: assert(sender != receiver)
	if sAddr.Equal(rAddr) {
		return shim.Error("can't transfer to self")
	}

	ab := NewAccountStub(stub, rAddr.Code)

	// sender
	sender, err := ab.GetAccount(sAddr)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the sender account")
	}
	if !sender.HasHolder(kid) {
		return shim.Error("invoker is not holder")
	}
	if sender.IsSuspended() {
		return shim.Error("the sender account is suspended")
	}

	// receiver
	receiver, err := ab.GetAccount(rAddr)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the receiver account")
	}
	if receiver.IsSuspended() {
		return shim.Error("the receiver account is suspended")
	}

	// sender balance
	bb := NewBalanceStub(stub)
	sBal, err := bb.GetBalance(sender.GetID())
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the sender's balance")
	}
	if sBal.Amount.Cmp(amount) < 0 {
		return shim.Error("not enough balance")
	}

	// receiver balance
	rBal, err := bb.GetBalance(receiver.GetID())
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the receiver's balance")
	}

	// options
	memo := ""

	// memo
	if len(params) > 3 {
		if len(params[3]) > 128 { // 128 charactors limit
			memo = params[3][:128]
		} else {
			memo = params[3]
		}
	}

	ub := NewUtxoStub(stub)
	var log *BalanceLog // log for response
	log, err = ub.Pay(sBal, rBal, *amount, memo)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to transfer")
	}

	// log is not nil
	data, err := json.Marshal(log)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}

// prune _
// params[0] : token | owned address. address/kid to prune. If there is no parameter or empty string, then current user's account will be pruned.
// params[1] : to account address
// params[2] : end date in seconds form
// params[3] : [optional] merge result key - bookmark(end chunk key)
// Description ////////////////////
func merge(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 3 {
		return shim.Error("incorrect number of parameters. expecting 3+")
	}
	// authentication
	kid, err := kid.GetID(stub, false)
	if err != nil {
		return shim.Error(err.Error())
	}

	var account AccountInterface

	var addr *Address
	code, err := ValidateTokenCode(params[0])
	if nil == err { // by token code
		addr = NewAddress(code, AccountTypePersonal, kid)
	} else { // by address
		addr, err = ParseAddress(params[0])
		if err != nil {
			return responseError(err, "failed to get the account")
		}
	}
	ab := NewAccountStub(stub, addr.Code)
	account, err = ab.GetAccount(addr)
	if err != nil {
		return responseError(err, "failed to get the account")
	}

	ub := NewUtxoStub(stub)

	// 1. There is no bookmark key
	esec, err := strconv.ParseInt(params[2], 10, 64)
	if nil != err {
		return shim.Error(err.Error())
	}
	etime := txtime.Unix(esec, 0)

	var stime *txtime.Time
	if len(params) > 3 && len(params[3]) > 0 {
		// bookmark
		key := ub.GetMergeResultKey(account.GetID(), params[3])
		mr, err := ub.GetMergeResultChunk(key)
		if nil != err {
			return shim.Error(err.Error())
		}
		if nil == mr {
			return shim.Error("invalid merge result id")
		}
		stime = mr.End.CreatedTime
	} else {
		mr, err := ub.GetLastestMergeResultByID(account)
		if nil != err {
			return shim.Error(err.Error())
		}
		if nil != mr {
			stime = mr.End.CreatedTime
		} else {
			stime = txtime.Unix(0, 0)
		}
	}

	// Validate merge range
	// 1. stime < etime
	if stime.Cmp(etime) >= 0 {
		return shim.Error("invalid merge range")
	}
	// 2. Already merged range need?
	if ok, err := ub.MergeRangeValidator(account.GetID(), stime, etime); !ok {
		if nil != err {
			return shim.Error(err.Error())
		}
		return shim.Error("Already merge range")
	}

	// 3. Get sum of chunks
	rAddr, err := ParseAddress(params[1])
	if nil != err {
		return shim.Error(err.Error())
	}
	receiver, err := ab.GetAccount(rAddr)
	if nil != err {
		return shim.Error(err.Error())
	}
	sum, startChunk, endChunk, err := ub.GetSumOfUtxoChunksByRange(account, receiver.GetID(), stime, etime)
	if nil != err {
		return shim.Error(err.Error())
	}

	// 4. Create MergeResult
	ts, err := txtime.GetTime(stub)
	if nil != err {
		return shim.Error(err.Error())
	}
	mrKey := ub.GetMergeResultKey(account.GetID(), endChunk.DOCTYPEID)
	mergeResult := NewMergeResultType(mrKey, params[1], account, startChunk, endChunk, ts, *sum)
	if err := ub.PutMergeResult(mergeResult); nil != err {
		return shim.Error(err.Error())
	}

	// 5. Transfer
	bb := NewBalanceStub(stub)
	sBal, err := bb.GetBalance(account.GetID())
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the receiver's balance")
	}
	rBal, err := bb.GetBalance(receiver.GetID())
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the receiver's balance")
	}
	rBal.Amount.Add(sum)
	rBal.UpdatedTime = ts
	if err := bb.PutBalance(rBal); err != nil {
		return shim.Error(err.Error())
	}
	memo := fmt.Sprintf("Merged chunks send. Check result %s", mrKey)
	rbl := NewBalanceTransferLog(sBal, rBal, *sum, memo)
	rbl.CreatedTime = ts
	if err = bb.PutBalanceLog(rbl); err != nil {
		return shim.Error(err.Error())
	}

	data, err := json.Marshal(mergeResult)
	if nil != err {
		return shim.Error(err.Error())
	}

	return shim.Success(data)

}
