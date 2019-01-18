// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
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

	var log *BalanceLog // log for response
	log, err = bb.Pay(sBal, rBal, *amount, memo)
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
// params[1] : start date in seconds form
// params[2] : end date in seconds form
// params[3] : [optional] merge result key
// Description ////////////////////
// 1. It merges UTXO chunks from the last merged datetime until current time or until it reaches 500th chunk.
// 2. If it has more than 500 records, then it only merges the first 500 chunks.
// 3. Those 500 chunk are remained insted of being deleted(writeset performance issue).
// 4. Insted, the newly created merged chunk will be created and inserted in 500.5th place(Meaning between 500th and 501st chunks)
// 5. So the next prune() will start from 500.5th place (this address is stored in the Merge History which stores the "last merged/500th chunk vs newly created chunk" address info.
// Question 1 - do we need to merge the already merged chunk, too or leave it as it is?
// Question 2 - how do we transfer from merchant to mother account? - Create new transfer function for UTXO -> Account/Balance?

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

	var stime *txtime.Time
	if len(params) > 3 && len(params[3]) != 0 {
		mergeResult, err := ub.GetMergeResultChunk(params[3])
		if nil != err {
			return shim.Error(err.Error())
		}
		s, err := ub.GetChunk(mergeResult.Start)
		if nil != err {
			return shim.Error(err.Error())
		}
		stime = s.CreatedTime
	} else {
		seconds, err := strconv.ParseInt(params[1], 10, 64)
		if nil != err {
			return shim.Error("invalid start time: need seconds since 1970")
		}
		stime = txtime.Unix(seconds, 0)
	}

	seconds, err := strconv.ParseInt(params[2], 10, 64)
	if nil != err {
		return shim.Error("invalid start time: need seconds since 1970")
	}
	etime := txtime.Unix(seconds, 0)

	if stime.Cmp(etime) >= 0 {
		return shim.Error("invalid merge range")
	}
	// Already Merged or not check
	// from_date < star
	valid, err := ub.MergeRangeValidator(account.GetID(), stime)
	if !valid {
		return shim.Error("invalid merge range")
	}

	sum, start, end, err := ub.GetSumOfUtxoChunksByRange(account.GetID(), stime, etime)
	if nil != err {
		return shim.Error(err.Error())
	}

	ts, err := txtime.GetTime(stub)
	if nil != err {
		return shim.Error(err.Error())
	}

	mKey := ""
	mergedChunk := NewChunkType(ub.CreateKey(stub.GetTxID()), account, nil, *sum, ts)
	if mKey, err = ub.PutChunk(mergedChunk); nil != err {
		return shim.Error(err.Error())
	}

	mrKey := ub.CreateMergeResultKey(account.GetID(), mKey)
	mergeResult := NewMergeResultType(mrKey, mKey, start, end, ts)

	data, err := json.Marshal(mergeResult)
	if nil != err {
		return shim.Error(err.Error())
	}
	if err := stub.PutState(mrKey, data); nil != err {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte(""))

}
