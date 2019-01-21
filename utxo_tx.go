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

	//get Account from the address
	ab := NewAccountStub(stub, addr.Code)
	account, err := ab.GetAccount(addr)
	if err != nil {
		return responseError(err, "failed to get the account")
	}

	//Account balance
	//bb := NewBalanceStub(stub)
	//aBal, err := bb.GetBalance(account.GetID())
	//if err != nil {
	//	logger.Debug(err.Error())
	//	return shim.Error("failed to get the receiver's balance")
	//}

	ub := NewUtxoStub(stub)

	stime, err := getPruneStartTime(ub, account.GetID())
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

	//TODO: must get the stime from the MergeHistory document instead of the parameter. Thus prune function will not have any parameters.

	gqResult, err := ub.GetUtxoChunksByTime(account.GetID(), stime, etime)
	if err != nil {
		return shim.Error(err.Error())
	}
	if gqResult.MergeCount == 0 {
		return shim.Error("no chunk found to prune in the given time period")
	}
	if gqResult.MergeCount == 1 {
		return shim.Error("all chunks are already pruned. prune aborted.")
	}

	fmt.Println("############ GetQueryUtxoChunk Result. From Address : ", gqResult.FromKey)
	fmt.Println("############ GetQueryUtxoChunk Result. To Address : ", gqResult.ToKey)
	fmt.Println("############ GetQueryUtxoChunk Result. Sum: ", gqResult.Sum)
	fmt.Println("############ GetQueryUtxoChunk Result. Merge Count : ", gqResult.MergeCount-1)
	fmt.Println("############ GetQueryUtxoChunk Result. Next Chunk Key : ", gqResult.NextChunkKey)

	amount, err := NewAmount(strconv.FormatInt(int64(gqResult.Sum), 10))
	if err != nil {
		return shim.Error(err.Error())
	}

	ts, err := txtime.GetTime(stub)
	if nil != err {
		return shim.Error(err.Error())
	}

	//TODO: Instead of creating the new chunk, we need to add the amount to the receiver's account
	//chunk := NewPayChunkType(account.GetID(), aBal, *amount, ts)
	//if err = ub.PutChunk(chunk); nil != err {
	//	return shim.Error(err.Error())
	//}

	//delete this. it's just the placeholder to prevent the not used error message
	if amount == nil {
		return shim.Success([]byte(""))
	}

	//create the merge history log
	mhl := NewMergeHistory(account.GetID(), gqResult.FromKey, gqResult.ToKey, gqResult.NextChunkKey, gqResult.Sum)
	mhl.CreatedTime = ts
	if err = ub.PutMergeHistory(mhl); err != nil {
		return shim.Error(err.Error())
	}

	data, err := json.Marshal(mhl)

	return shim.Success(data)

}

// getPruneStartTime _
// Merge start time is always retrieved from MergeHistory regardless of the next chunk key presence.
// Next chunk key is used just to indicate there are remaining chunks to merge in the given time period.
func getPruneStartTime(ub *UtxoStub, id string) (*txtime.Time, error) {
	dsTime := "2019-01-01T12:00:00.000000000Z"
	var stime *txtime.Time
	mh, err := ub.GetLatestMergeHistory(id)
	if err != nil {
		return nil, err
	} else if mh == nil { //There is no merge history yet.
		stime, err = txtime.Parse(dsTime)
		fmt.Println("######getPruneStartTime debug 1")
		if err != nil {
			fmt.Println("######getPruneStartTime debug 2")
			return nil, err
		}
	} else { //MergeHistory exists
		nChunk, err := ub.GetChunk(mh.ToAddress)
		if err != nil {
			return nil, err
		}
		stime = nChunk.CreatedTime
	}

	fmt.Println("############ Merge search start time: ", stime)
	return stime, nil
}
