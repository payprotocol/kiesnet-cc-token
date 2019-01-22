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
// params[1] : receiver address. Positive amount: address to receives the amount. Negative amount: Original chunk ID.
// params[2] : amount (big int string). Positive amount: pay from user to merchant. Negative: full/partial refund from merchant to user.
// params[3] : optional. memo (max 128 charactors)
func pay(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 3 {
		return shim.Error("incorrect number of parameters. expecting 3+")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	//address validation
	var sAddr *Address
	sAddr, err = ParseAddress(params[0])
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to parse the sender's account address")
	}

	// amount
	amount, err := NewAmount(params[2])
	if err != nil {
		return shim.Error(err.Error())
	}

	if amount.Sign() == 0 {
		return shim.Error("invalid amount. amount shouldn't be 0")
	}

	ub := NewUtxoStub(stub)
	rid := params[1]
	pkey := ""

	//validate the refund amount. The refund amound can't exceed the original amount
	if amount.Sign() < 0 {

		ck, err := ub.GetChunk(params[1]) //this case params[1] is the original chunk key
		if err != nil {
			return shim.Error("valid chunk id is required for proper refund process")
		}

		rid = ck.RID     //getting the user's id from the original chunk
		pkey = params[1] //save parent key if this is refund for the later-created negatived chunk

		//if the refund amount is greater than the original amount
		amountCmp := amount.Copy()
		if ck.Amount.Cmp(amountCmp.Neg()) < 0 {
			return shim.Error("refund amount can't be greater than the original amount")
		}

		totalRefundAmount, err := ub.GetTotalRefundAmount(params[0], pkey)

		if err != nil {
			return shim.Error("failed to get the total refund amount")
		}

		if ck.Amount.Cmp(totalRefundAmount.Add(amountCmp)) < 0 {
			return shim.Error("can't exceed the sum of past refund amounts")
		}
	}

	// validate sender/receiver addresses
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

	log, err = ub.Pay(sBal, rBal, *amount, memo, pkey)
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
// params[0] : Token Code or Address to prune.
// params[1] : Address of receiver / Master Account.
// params[2] : Prune end time. It is not guaranteed that the prune merge all chunks into one in this given period time. If it reaches the threshhold of 500, then it finishes the current action expecting the next call from the client.
// params[3] : Optional. Next Chunk Key. If the key exists, this method should be called recursively until the empty string is returned on the response. //TODO: next chunk 대신 마지막 청크의 크리에잇 타임으로 할것인가?
func prune(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 3 {
		return shim.Error("incorrect number of parameters. expecting at least 3 parameters")
	}
	// authentication. get KID of current user.
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	var sAddr *Address
	code, err := ValidateTokenCode(params[0])
	if nil == err { // by token code
		sAddr = NewAddress(code, AccountTypePersonal, kid)
	} else { // by address
		sAddr, err = ParseAddress(params[0])
		if err != nil {
			return responseError(err, "failed to get the account")
		}
	}

	rAddr, err := ParseAddress(params[1])
	if err != nil {
		return shim.Error("failed to parse the receiver's account address")
	}
	if sAddr.Code != rAddr.Code {
		return shim.Error("sender and receiver's token do not match")
	}
	if sAddr.Equal(rAddr) {
		return shim.Error("can't pay to self")
	}

	ab := NewAccountStub(stub, rAddr.Code)
	// sender
	sender, err := ab.GetAccount(sAddr)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the sender account")
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

	ub := NewUtxoStub(stub)

	// start time
	stime, err := getPruneStartTime(ub, sender.GetID())
	if err != nil {
		return shim.Error("failed to get the start time")
	}

	//merge end time
	sec, err := strconv.ParseInt(params[2], 10, 64)
	if err != nil {
		return responseError(err, "failed to parse the end time")
	}
	etime := txtime.Unix(sec, 0) //

	fmt.Println("############ Merge query end time: ", etime)

	qResult, err := ub.GetUtxoChunksByTime(sender.GetID(), stime, etime)
	if err != nil {
		return shim.Error(err.Error())
	}
	if qResult.MergeCount == 0 {
		return shim.Error("no chunk found to prune in the given time period")
	}
	if qResult.MergeCount == 1 { //1개가 있을떄도 프룬을 해줘야 한다.
		return shim.Error("all chunks are already pruned. prune aborted.")
	}

	fmt.Println("############ GetQueryUtxoChunk Result. From Address : ", qResult.FromKey)
	fmt.Println("############ GetQueryUtxoChunk Result. To Address : ", qResult.ToKey)
	fmt.Println("############ GetQueryUtxoChunk Result. Sum: ", qResult.Sum)
	fmt.Println("############ GetQueryUtxoChunk Result. Merge Count : ", qResult.MergeCount-1)
	fmt.Println("############ GetQueryUtxoChunk Result. Next Chunk Key : ", qResult.NextChunkKey)

	amount, err := NewAmount(strconv.FormatInt(int64(qResult.Sum), 10))
	if err != nil {
		return shim.Error(err.Error())
	}

	//TODO: Merged chunk is no mored created. Thus, commented out.
	//chunk := NewPayChunkType(account.GetID(), aBal, *amount, ts)
	//if err = ub.PutChunk(chunk); nil != err {
	//	return shim.Error(err.Error())
	//}

	// sender balance
	bb := NewBalanceStub(stub)

	// receiver balance
	rBal, err := bb.GetBalance(receiver.GetID())
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the receiver's balance")
	}
	pLog, err := ub.Prune(sender.GetID(), rBal, *amount, qResult)

	if err != nil {
		return shim.Error("failed to prune balance")
	}

	data, err := json.Marshal(pLog)
	return shim.Success(data)
}
