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

	//TODO: should create new UTXO balance log instead of the existing balance log
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
func prune(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	// if len(params) < 3 {
	// 	return shim.Error("incorrect number of parameters. expecting 3")
	// }
	// // authentication
	// kid, err := kid.GetID(stub, true)
	// if err != nil {
	// 	return shim.Error(err.Error())
	// }
	// var addr *Address
	// code, err := ValidateTokenCode(params[0])
	// if nil == err { // by token code
	// 	addr = NewAddress(code, AccountTypePersonal, kid)
	// } else { // by address
	// 	addr, err = ParseAddress(params[0])
	// 	if err != nil {
	// 		return responseError(err, "failed to get the account")
	// 	}
	// }
	// ab := NewAccountStub(stub, addr.Code)
	// account, err := ab.GetAccount(addr)
	// if err != nil {
	// 	return responseError(err, "failed to get the account")
	// }
	// account.GetID()

	// seconds, err := strconv.ParseInt(params[1], 10, 64)
	// if nil != err {
	// 	return shim.Error(err.Error())
	// }
	// stime := txtime.Unix(seconds, 0)

	// seconds, err = strconv.ParseInt(params[2], 10, 64)
	// if nil != err {
	// 	return shim.Error(err.Error())
	// }
	// etime := txtime.Unix(seconds, 0)

	// ub := NewUtxoStub(stub)
	// bookmark := ""
	// buf := bytes.NewBufferString("")
	// var res *QueryResult
	//ub.GetAllUtxoChunks(account.GetID(), stime, etime)
	// ub.GetQueryUtxoChunks(account.GetID(), "", stime, etime)
	// for res, err = ub.GetQueryUtxoChunks(account.GetID(), bookmark, stime, etime); res.Meta.FetchedRecordsCount >= 2; {
	// 	fmt.Println(res.Meta.Bookmark)
	// 	if nil != err {
	// 		return shim.Error(err.Error())
	// 	}
	// 	_, err = buf.Write(res.Records)
	// 	if nil != err {
	// 		return shim.Error(err.Error())
	// 	}
	// 	bookmark = res.Meta.Bookmark

	// }
	// b, _ := res.MarshalJSON()

	// fmt.Println(b)
	// query := CreateQueryPayChunks(account.GetID(), stime, etime)
	// var sum int64
	// iter, _, err := stub.GetQueryResultWithPagination(query, 1000, "")
	// if nil != err {
	// 	return shim.Error(err.Error())
	// }
	// defer iter.Close()
	// for iter.HasNext() {
	// 	type temp struct {
	// 		Amount string `json:"amount"`
	// 	}
	// 	tmp := temp{}
	// 	kv, _ := iter.Next()
	// 	err = json.Unmarshal(kv.Value, &tmp)
	// 	fmt.Println(tmp.Amount)
	// 	val, err := strconv.ParseInt(tmp.Amount, 10, 64)
	// 	if nil != err {
	// 		return shim.Error(err.Error())
	// 	}
	// 	sum += val

	// }
	// fmt.Println(sum)
	// // the number of chunck is more than 1000
	// for 1000 > meta.FetchedRecordsCount {
	// 	iter, meta, err = stub.GetQueryResultWithPagination(query, 1000, meta.Bookmark)
	// 	if nil != err {
	// 		return shim.Error(err.Error())
	// 	}
	// }
	return shim.Success([]byte(""))
}
