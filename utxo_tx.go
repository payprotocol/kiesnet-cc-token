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
// TODO: 나노초가 똑같은 트랜젝션이 같은수가 있으므로, 쿼리로 같은 청크가 있는지 체크를 한다음에 같은키의 청크가 있으면 이 트랜잭션을 켄슬/대러 처리 해야한다.
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

	// TODO: REFUND/CANCEL.
	// if amount.Sign() <= 0 {
	// 	return shim.Error("invalid amount. amount must be larger than 0")
	// }

	// validate sender addresses
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
	fmt.Println(kid)
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

	ub := NewUtxoStub(stub)
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
// params[0] : Token Code or Address to prune.
// params[1] : Address of receiver / Master Account.
// params[2] : Prune end time. It is not guaranteed that the prune merge all chunks into one in this given period time. If it reaches the threshhold of 500, then it finishes the current action expecting the next call from the client.
// params[3] : Optional. Next Chunk Key. If the key exists, this method should be called recursively until the empty string is returned on the response.
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
	etime := txtime.Unix(sec, 0)

	fmt.Println("############ Merge query end time: ", etime)

	qResult, err := ub.GetUtxoChunksByTime(sender.GetID(), stime, etime)
	if err != nil {
		return shim.Error(err.Error())
	}
	if qResult.MergeCount == 0 {
		return shim.Error("no chunk found to prune in the given time period")
	}
	if qResult.MergeCount == 1 {
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
