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

// params[0] : tokencode | target address (empty string = personal account)
// params[1] : start timestamp
// params[2] : end timestamp
// params[3] : bookmark ?
// func prune(stub shim.ChaincodeStubInterface, params []string) peer.Response {
// 	if len(params) < 3 {
// 		return shim.Error("incorrect number of parameters. expecting 3")
// 	}
// 	// authentication. get KID of current user.
// 	kid, err := kid.GetID(stub, true)
// 	if err != nil {
// 		return shim.Error(err.Error())
// 	}
// 	var addr *Address
// 	code, err := ValidateTokenCode(params[0])
// 	if nil == err { // by token code
// 		addr = NewAddress(code, AccountTypePersonal, kid)
// 	} else { // by address
// 		addr, err = ParseAddress(params[0])
// 		if err != nil {
// 			return responseError(err, "failed to get the account")
// 		}
// 	}
// 	ab := NewAccountStub(stub, addr.Code)
// 	account, err := ab.GetAccount(addr)
// 	if err != nil {
// 		return responseError(err, "failed to get the account")
// 	}
// 	account.GetID()

// 	//get the start time for pruning
// 	seconds, err := strconv.ParseInt(params[1], 10, 64)
// 	if nil != err {
// 		return shim.Error(err.Error())
// 	}
// 	stime := txtime.Unix(seconds, 0)

// 	//get the end time for pruning
// 	seconds, err = strconv.ParseInt(params[2], 10, 64)
// 	if nil != err {
// 		return shim.Error(err.Error())
// 	}
// 	etime := txtime.Unix(seconds, 0)

// 	ub := NewUtxoStub(stub)
// 	bookmark := ""
// 	buf := bytes.NewBufferString("")
// 	var res *QueryResult
// 	ub.GetAllUtxoChunks(account.GetID(), stime, etime)
// 	ub.GetQueryUtxoChunks(account.GetID(), "", stime, etime)
// 	for res, err = ub.GetQueryUtxoChunks(account.GetID(), bookmark, stime, etime); res.Meta.FetchedRecordsCount >= 2; {
// 		fmt.Println(res.Meta.Bookmark)
// 		if nil != err {
// 			return shim.Error(err.Error())
// 		}
// 		_, err = buf.Write(res.Records)
// 		if nil != err {
// 			return shim.Error(err.Error())
// 		}
// 		bookmark = res.Meta.Bookmark

// 	}
// 	b, _ := res.MarshalJSON()

// 	fmt.Println(b)
// 	query := CreateQueryPayChunks(account.GetID(), stime, etime)
// 	var sum int64
// 	iter, _, err := stub.GetQueryResultWithPagination(query, 1000, "")
// 	if nil != err {
// 		return shim.Error(err.Error())
// 	}
// 	defer iter.Close()
// 	for iter.HasNext() {
// 		type temp struct {
// 			Amount string `json:"amount"`
// 		}
// 		tmp := temp{}
// 		kv, _ := iter.Next()
// 		err = json.Unmarshal(kv.Value, &tmp)
// 		fmt.Println(tmp.Amount)
// 		val, err := strconv.ParseInt(tmp.Amount, 10, 64)
// 		if nil != err {
// 			return shim.Error(err.Error())
// 		}
// 		sum += val

// 	}
// 	fmt.Println(sum)
// 	// the number of chunck is more than 1000
// 	for 1000 > meta.FetchedRecordsCount {
// 		iter, meta, err = stub.GetQueryResultWithPagination(query, 1000, meta.Bookmark)
// 		if nil != err {
// 			return shim.Error(err.Error())
// 		}
// 	}
// 	return shim.Success([]byte(""))
// }

// prune _
// params[0] : Token Code or Address to prune.
// params[1] : Prune start time.
// params[2] : Prune end time. It is not guaranteed that the prune merge all chunks into one in this given period time. If it reached the threshhold of 500, then it finished the current action expeting the next call from the client.
// params[3] : Optional. Next Chunk Key.
// Description ////////////////////
// 1. It merges UTXO chunks from the last merged datetime until current time or until it reaches 500th chunk.
// 2. If it has more than 500 records, then it only merges the first 500 chunks.
// 3. Those 500 chunk are remained insted of being deleted(writeset performance issue).
// 4. Insted, the newly created merged chunk will be created and inserted in 500.5th place(Meaning between 500th and 501st chunks)
// 5. So the next prune() will start from 500.5th place (this address is stored in the Merge History which stores the "last merged/500th chunk vs newly created chunk" address info.
// Question 1 - do we need to merge the already merged chunk, too or leave it as it is?
// Question 2 - how do we transfer from merchant to mother account? - Create new transfer function for UTXO -> Account/Balance?

func prune(stub shim.ChaincodeStubInterface, params []string) peer.Response {

	//var addr *Address
	//var err error

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
	bb := NewBalanceStub(stub)
	aBal, err := bb.GetBalance(account.GetID())
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the receiver's balance")
	}

	//merge start time.
	//if MergeHistory key presents, then the start time is retrieved from MergeHistory -> ToAddress -> CreateTime.

	var stime *txtime.Time

	//if the next chunk key presents, get the create time and set it as the new start time
	if len(params) > 3 {
		nChunk, err := bb.GetChunk(params[3])
		if err != nil {
			shim.Error("failed to get the next start chunk")
		}
		stime = nChunk.CreatedTime
	} else {
		sec, err := strconv.ParseInt(params[1], 10, 64)
		if err != nil {
			return responseError(err, "failed to parse the start time")
		}
		stime = txtime.Unix(sec, 0)
	}

	fmt.Println("############ Merge search start time: ", stime)

	//merge end time
	sec, err := strconv.ParseInt(params[2], 10, 64)
	if err != nil {
		return responseError(err, "failed to parse the start time")
	}
	etime := txtime.Unix(sec, 0)

	fmt.Println("############ Merge search end time: ", etime)

	//TODO: must get the stime from the MergeHistory document instead of the parameter. Thus prune function will not have any parameters.

	ub := NewUtxoStub(stub)
	gqResult, err := ub.GetQueryUtxoChunk(account.GetID(), stime, etime)
	if err != nil {
		return shim.Error(err.Error())
	}
	if gqResult.MergeCount == 0 {
		return shim.Error("no chunk found to prune in the given time period")
	}
	if gqResult.MergeCount == 1 {
		return shim.Error("only one chunk found to prune. prune aborted.")
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

	// Negative amount should be allowed, too?
	// if amount.Sign() <= 0 {
	// 	return shim.Error("invalid amount. amount must be larger than 0")
	// }

	//TODO: we should get the create time from the last merged chunk so that the next prune() can start from this newly merged chunk.

	ts, err := txtime.GetTime(stub)
	if nil != err {
		return shim.Error(err.Error())
	}

	chunk := NewPayChunkType(bb.stub.GetTxID(), aBal, *amount, ts)
	if err = bb.PutChunk(chunk); nil != err {
		return shim.Error(err.Error())
	}

	//TODO: create the merge history log
	mhl := NewMergeHistory(account.GetID(), gqResult.FromKey, gqResult.ToKey, bb.CreateChunkKey(chunk.DOCTYPEID), bb.CreateChunkKey(gqResult.NextChunkKey), gqResult.Sum)
	mhl.CreatedTime = ts
	if err = bb.PutMergeHistory(mhl); err != nil {
		return shim.Error(err.Error())
	}

	data, err := json.Marshal(mhl)

	return shim.Success(data)

}
