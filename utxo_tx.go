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
	if nil != err {
		return shim.Error(err.Error())
	}

	// amount
	amount, err := NewAmount(params[2])
	if nil != err {
		return shim.Error(err.Error())
	}

	// TODO: REFUND/CANCEL.
	// if amount.Sign() <= 0 {
	// 	return shim.Error("invalid amount. amount must be larger than 0")
	// }

	// validate sender addresses
	rAddr, err := ParseAddress(params[1])
	if nil != err {
		logger.Debug(err.Error())
		return shim.Error("failed to parse the receiver's account address")
	}
	var sAddr *Address
	if len(params[0]) > 0 {
		sAddr, err = ParseAddress(params[0])
		if nil != err {
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
	if nil != err {
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
	if nil != err {
		logger.Debug(err.Error())
		return shim.Error("failed to transfer")
	}

	// log is not nil
	data, err := json.Marshal(log)
	if nil != err {
		logger.Debug(err.Error())
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}

// prune _
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

	// balance log
	rbl := NewBalanceTransferLog(nil, balance, *chunkSum.Sum, "")
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
