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
// params[0] : optional. address/kid to prune. If there is no parameter or empty string, then current user's account will be pruned.
// Description ////////////////////
// 1. It merges UTXO chunks from the last merged datetime until current time or until it reaches 500th chunk.
// 2. If it has more than 500 records, then it only merges the first 500 chunks.
// 3. Those 500 chunk are remained insted of being deleted(writeset performance issue).
// 4. Insted, the newly created merged chunk will be created and inserted in 500.5th place(Meaning between 500th and 501st chunks)
// 5. So the next prune() will start from 500.5th place (this address is stored in the Merge History which stores the "last merged/500th chunk vs newly created chunk" address info.
// Question 1 - do we need to merge the already merged chunk, too or leave it as it is?
// Question 2 - how do we transfer from merchant to mother account? - Create new transfer function for UTXO -> Account/Balance?

func prune(stub shim.ChaincodeStubInterface, params []string) peer.Response {

	var addr *Address
	var err error

	//no params or empty string params[0]
	if len(params) == 0 || params[0] == "" {
		//get kid of current user
		kid, err := kid.GetID(stub, true)
		if err != nil {
			return shim.Error(err.Error())
		}
		//create Address from kid
		addr = NewAddress("", AccountTypePersonal, kid)
	} else {
		addr, err = ParseAddress(params[0])
		if err != nil {
			return responseError(err, "the address is invalid")
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

	//TODO: get start time from MergeHistory. If the Log doesnt exist, then search from 1546300800 or 2019-01-01 12:00:00.
	stimeStamp := "1546300800" //default start time. This should be overried by the time from MergeHistory.

	//TODO: Get MergeHistory then override sTime.
	sec, err := strconv.ParseInt(stimeStamp, 10, 64)
	if err != nil {
		return responseError(err, "failed to parse the start time")
	}
	stime := txtime.Unix(sec, 0)

	fmt.Println("############ Merge search start time: ", stime)

	//get end time is the current time
	etime, _ := txtime.GetTime(stub)

	fmt.Println("############ Merge search end time: ", etime)

	//ub := NewUtxoStub(stub)
	//bookmark := ""
	//buf := bytes.NewBufferString("")
	//var qResult *QueryResult

	// bookmark := ""
	// query := CreateQueryPayChunks(account.GetID(), params[1], params[2])
	// var pageSize int32
	// var sum int64
	// var totalRecords int32
	// pageSize = 2

	// for {
	// 	iter, meta, err := stub.GetQueryResultWithPagination(query, pageSize, bookmark)
	// 	if err != nil {
	// 		return shim.Error(err.Error())
	// 	}
	// 	defer iter.Close()
	// 	for iter.HasNext() {
	// 		type temp struct {
	// 			Amount string `json:"amount`
	// 		}
	// 		tmp := temp{}
	// 		kv, _ := iter.Next()
	// 		err = json.Unmarshal(kv.Value, &tmp)
	// 		fmt.Println("######### tmp.Amount : ", tmp.Amount)
	// 		val, err := strconv.ParseInt(tmp.Amount, 10, 64)
	// 		if err != nil {
	// 			return shim.Error(err.Error())
	// 		}
	// 		sum += val
	// 	}

	// 	totalRecords = totalRecords + meta.FetchedRecordsCount
	// 	if meta.FetchedRecordsCount == 0 {
	// 		break
	// 	}

	// 	bookmark = meta.Bookmark
	// }

	//fmt.Println("page size: ", pageSize)
	//fmt.Println("total # records:", totalRecords)
	//fmt.Println("total amount:", sum)
	// ########### end of query with pagination test. if query with pagination is used, fabric doesnt allow to perform PUTSTATE. Thus instead of GetQueryResultWithPagination, we must use GetQueryResult ###########

	//TODO: must get the stime from the MergeHistory document instead of the parameter. Thus prune function will not have any parameters.
	var sum int64
	query := CreateQueryPayChunks(account.GetID(), stime, etime)
	iter, err := stub.GetQueryResult(query)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer iter.Close()
	recCounter := 0
	const numMergeLimit = 500

	for iter.HasNext() {
		recCounter++

		//TODO: need to get the address and create time, too
		type temp struct {
			Amount string `json:"amount`
		}
		tmp := temp{}
		kv, _ := iter.Next()
		err = json.Unmarshal(kv.Value, &tmp)
		fmt.Println("######### tmp.Amount : ", tmp.Amount)
		val, err := strconv.ParseInt(tmp.Amount, 10, 64)
		if err != nil {
			return shim.Error(err.Error())
		}
		sum += val
		if recCounter == numMergeLimit {
			break
		}
	}

	fmt.Println("total amount:", sum)
	// amount
	amount, err := NewAmount(strconv.FormatInt(int64(sum), 10))
	if err != nil {
		return shim.Error(err.Error())
	}
	if amount.Sign() <= 0 {
		return shim.Error("invalid amount. amount must be larger than 0")
	}

	//TODO: we should get the create time from the last merged chunk so that the next prune() can start from this newly merged chunk.

	ts, err := txtime.GetTime(stub)
	if nil != err {
		return shim.Error(err.Error())
	}

	chunk := NewPayChunkType(bb.stub.GetTxID(), aBal, *amount, ts)
	if err = bb.PutChunk(chunk); nil != err {
		return shim.Error(err.Error())
	}

	//TODO: delete existing chunks

	//TODO: create the prune log

	//ub.GetAllUtxoChunks(account.GetID(), params[1], params[2])
	//ub.GetQueryUtxoChunks(account.GetID(), "", params[1], params[2])

	return shim.Success([]byte(""))

}
