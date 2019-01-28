// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/contract"
	"github.com/key-inside/kiesnet-ccpkg/kid"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// params[0] : sender's address or Token Code.
// params[1] : receiver's address
// params[2] : amount
// params[3] : optional. memo (max 128 charactors)
// params[4] : optional. expiry (duration represented by int64 seconds, multi-sig only)
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
	code, err := ValidateTokenCode(params[0])
	if nil == err { // by token code
		sAddr = NewAddress(code, AccountTypePersonal, kid)
	} else { // by address
		sAddr, err = ParseAddress(params[0])
		if err != nil {
			return responseError(err, "failed to get the account")
		}
	}

	// amount
	amount, err := NewAmount(params[2])
	if nil != err {
		return shim.Error(err.Error())
	}
	if amount.Sign() < 1 {
		return shim.Error("invalid amount.  must be greater than 0")
	}

	// receiver address validation
	rAddr, err := ParseAddress(params[1])
	if err != nil {
		return responseError(err, "failed to parse the receiver's account address")
	}

	if rAddr.Code != sAddr.Code { // not same token
		return shim.Error("different token accounts")
	}

	// prevent from paying to self
	if sAddr.Equal(rAddr) {
		return shim.Error("can't pay to self")
	}

	ab := NewAccountStub(stub, rAddr.Code)

	// sender account validation
	sender, err := ab.GetAccount(sAddr)
	if nil != err {
		return responseError(err, "failed to get the sender account")
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
		return responseError(err, "failed to get the receiver account")
	}
	if receiver.IsSuspended() {
		return shim.Error("the receiver account is suspended")
	}

	// sender balance
	bb := NewBalanceStub(stub)
	sBal, err := bb.GetBalance(sender.GetID())
	if nil != err {
		return responseError(err, "failed to get the sender's balance")
	}

	if sBal.Amount.Cmp(amount) < 0 {
		return shim.Error("not enough balance")
	}

	// receiver balance
	rBal, err := bb.GetBalance(receiver.GetID())
	if nil != err {
		return responseError(err, "failed to get the receiver's balance")
	}

	// options
	memo := ""
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
		// expiry time
		if len(params) > 4 && len(params[5]) > 0 {
			expiry, err = strconv.ParseInt(params[5], 10, 64)
			if err != nil {
				responseError(err, "invalid expiry: need seconds")
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
		doc := []string{"pay", pbID, sender.GetID(), receiver.GetID(), amount.String(), memo}
		docb, err := json.Marshal(doc)
		if err != nil {
			return responseError(err, "failed to marshal contract do")
		}
		con, err := contract.CreateContract(stub, docb, expiry, signers)
		if err != nil {
			return responseError(err, "failed to create the contract")
		}
		// pending balance
		log, err = bb.Deposit(pbID, sBal, con, *amount, memo)
		if err != nil {
			return responseError(err, "failed to create a pending balance")
		}

	} else {
		log, err = NewUtxoStub(stub).Pay(sBal, rBal, *amount, memo, "")
		if err != nil {
			return responseError(err, "failed to pay")
		}
	}

	// log is not nil
	data, err := json.Marshal(log)
	if nil != err {
		return responseError(err, "failed to marshal the log")
	}

	return shim.Success(data)
}

// params[0] : sender's address
// params[1] : original pay key
// params[2] : refund amount.
// params[3] : optional. memo (max 128 charactors)
func refund(stub shim.ChaincodeStubInterface, params []string) peer.Response {
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
	code, err := ValidateTokenCode(params[0])
	if nil == err { // by token code
		sAddr = NewAddress(code, AccountTypePersonal, kid)
	} else { // by address
		sAddr, err = ParseAddress(params[0])
		if err != nil {
			return responseError(err, "failed to get the account")
		}
	}

	// amount
	amount, err := NewAmount(params[2])
	if nil != err {
		return shim.Error(err.Error())
	}
	if amount.Sign() < 1 {
		return shim.Error("invalid amount. must be greater than 0")
	}

	ub := NewUtxoStub(stub)
	pkey := params[1]

	parentPay, err := ub.GetPay(pkey)
	if err != nil {
		return shim.Error("failed to get the original payment from the key")
	}

	totalRefund := parentPay.TotalRefund

	if parentPay.Amount.Cmp(totalRefund.Copy().Add(amount)) < 0 {
		return shim.Error("can't exceed the original pay amount")
	}

	rid := parentPay.RID //getting the receiver's id from the original pay

	// receiver address validation
	rAddr, err := ParseAddress(rid)
	if err != nil {
		return responseError(err, "failed to parse the receiver's account address")
	}

	if rAddr.Code != sAddr.Code { // not same token
		return shim.Error("different token accounts")
	}

	if sAddr.Equal(rAddr) {
		return shim.Error("can't refund to self")
	}

	ab := NewAccountStub(stub, rAddr.Code)

	// sender account validation
	sender, err := ab.GetAccount(sAddr)
	if nil != err {
		return responseError(err, "failed to get the sender account")
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
		return responseError(err, "failed to get the receiver account")
	}
	if receiver.IsSuspended() {
		return shim.Error("the receiver account is suspended")
	}

	// sender balance
	bb := NewBalanceStub(stub)
	sBal, err := bb.GetBalance(sender.GetID())
	if nil != err {
		return responseError(err, "failed to get the sender's balance")
	}

	// receiver balance
	rBal, err := bb.GetBalance(receiver.GetID())
	if nil != err {
		return responseError(err, "failed to get the receiver's balance")
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

	var log *BalanceLog
	log, err = ub.Refund(sBal, rBal, *amount.Neg(), memo, parentPay)

	if err != nil {
		return responseError(err, "failed to pay")
	}

	// log is not nil
	data, err := json.Marshal(log)
	if nil != err {
		return responseError(err, "failed to marshal the log")
	}

	return shim.Success(data)
}

// params[0] : Token Code or Address to prune.
// params[1] : Prune end timestamp(in UTC)
// ???: next_key를 다음 콜에 파라미터를 쓰지 않는다면 has_more, not_complete 같은 이름의 boolean값이 나을 듯
func prune(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	//클라이언트가 실수로 2번째 파라미터를 입력하지 않을것을 막기위해 endtime을 입력하게 한다.
	// ???:  opional etime
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
	// account
	account, err := NewAccountStub(stub, addr.Code).GetAccount(addr)
	if nil != err {
		return responseError(err, "failed to get the account")
	}
	if account.IsSuspended() {
		return shim.Error("the account is suspended")
	}

	bb := NewBalanceStub(stub)
	balance, err := bb.GetBalance(account.GetID())
	if nil != err {
		return responseError(err, "failed to get the balance")
	}
	ub := NewUtxoStub(stub)

	// start time
	stime := txtime.Unix(0, 0)
	if 0 < len(balance.LastPrunedPayID) {
		lastTime := strings.Split(balance.LastPrunedPayID, "_")[2]
		s, err := strconv.ParseInt(lastTime[0:10], 10, 64)
		if nil != err {
			return responseError(err, "failed to get seconds")
		}
		n, err := strconv.ParseInt(lastTime[10:], 10, 64)
		if nil != err {
			return responseError(err, "failed to get seconds")
		}
		stime = txtime.Unix(s, n)
	}
	// end time
	seconds, err := strconv.ParseInt(params[1], 10, 64)
	if nil != err {
		return responseError(err, "failed to parse the end time")
	}

	etime := txtime.Unix(seconds, 0)
	//add 10 minutes to the end time and compare with the UTC current time. if etime+10minutes is greater than current time, set the end time to 10 minuest before current time.
	ts, err := txtime.GetTime(stub)
	if nil != err {
		return responseError(err, "failed to get the timestamp")
	}

	safeTime := txtime.New(ts.Add(6e+11)) // 6e+11 = time.Duration(-10)*time.Minute
	if etime.Cmp(safeTime) > 0 {
		etime = safeTime
	}

	paySum, err := ub.GetPaySumByTime(account.GetID(), stime, etime)
	if nil != err {
		return responseError(err, "failed to prune pays")
	}
	// Add balance
	balance.Amount.Add(paySum.Sum)
	balance.UpdatedTime = ts
	if 0 != len(paySum.End) {
		balance.LastPrunedPayID = paySum.End
	}

	if err := bb.PutBalance(balance); nil != err {
		return responseError(err, "failed to update balance")
	}

	// balance log
	rbl := NewBalanceWithPruneLog(balance, *paySum.Sum, paySum.Start, paySum.End)
	rbl.CreatedTime = ts
	if err = bb.PutBalanceLog(rbl); err != nil {
		return shim.Error(err.Error())
	}

	data, err := json.Marshal(paySum)
	if nil != err {
		return shim.Error(err.Error())
	}
	return shim.Success(data)
}

// payList _
// params[0] : token code | account address
// params[1] : bookmark
// params[2] : fetch size (if < 1 => default size, max 200)
// params[3] : start time (time represented by int64 seconds)
// params[4] : end time (time represented by int64 seconds)
func payList(stub shim.ChaincodeStubInterface, params []string) peer.Response {
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
				return responseError(err, "invalid fetch size")
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

	res, err := NewUtxoStub(stub).GetUtxoPaysByTime(addr.String(), bookmark, stime, etime, fetchSize)
	if nil != err {
		return responseError(err, "failed to get pays log")
	}

	data, err := json.Marshal(res)
	if err != nil {
		return responseError(err, "failed to marshal pays logs")
	}

	return shim.Success(data)
}

// contract callbacks

// doc: ["pay", pending-balance-ID, sender-ID, receiver-ID, amount, memo]
func executePay(stub shim.ChaincodeStubInterface, cid string, doc []interface{}) peer.Response {
	if len(doc) < 5 {
		return shim.Error("invalid contract document")
	}

	// pending balance
	bb := NewBalanceStub(stub)
	pb, err := bb.GetPendingBalance(doc[1].(string))
	if err != nil {
		return responseError(err, "failed to get the pending balance")
	}
	// validate
	if pb.Type != PendingBalanceTypeContract || pb.RID != cid {
		return shim.Error("invalid pending balance")
	}

	// ???: GetBalance -> receiver-ID
	// ISSUE: check accounts ? (suspended) Business...
	if err = NewUtxoStub(stub).PayPendingBalance(pb, doc[3].(string), doc[5].(string)); err != nil {
		return responseError(err, "failed to pay a pending balance")
	}

	return shim.Success(nil)
}
