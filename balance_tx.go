// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/kid"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// params[0] : token code | account address
// params[1] : bookmark
// params[2] : start_time (time represented by int64 seconds)
// params[3] : end_time (time represented by int64 seconds)
func balanceLogs(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 1 {
		return shim.Error("incorrect number of parameters. expecting 1+")
	}

	// authentication
	kid, err := kid.GetID(stub, false)
	if err != nil {
		return shim.Error(err.Error())
	}

	// options
	bookmark := ""
	var stime, etime *time.Time
	if len(params) > 1 {
		bookmark = params[1]
		// start time
		if len(params) > 2 {
			if len(params[2]) > 0 {
				seconds, err := strconv.ParseInt(params[2], 10, 64)
				if err != nil {
					return shim.Error("invalid start time: need seconds since 1970")
				}
				ut := time.Unix(seconds, 0)
				stime = &ut
			}
			// end time
			if len(params) > 3 {
				if len(params[3]) > 0 {
					seconds, err := strconv.ParseInt(params[3], 10, 64)
					if err != nil {
						return shim.Error("invalid end time: need seconds since 1970")
					}
					ut := time.Unix(seconds, 0)
					if stime != nil && stime.After(ut) {
						return shim.Error("invalid time parameters")
					}
					etime = &ut
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

	bb := NewBalanceStub(stub)
	res, err := bb.GetQueryBalanceLogs(addr.String(), bookmark, stime, etime)
	if err != nil {
		return responseError(err, "failed to get balance logs")
	}

	data, err := json.Marshal(res)
	if err != nil {
		return responseError(err, "failed to marshal balance logs")
	}
	return shim.Success(data)
}

// params[0] : token code | account address
// params[1] : sort ('created_time' | 'pending_time')
// params[2] : bookmark
func balancePendingList(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 1 {
		return shim.Error("incorrect number of parameters. expecting 1+")
	}

	// authentication
	kid, err := kid.GetID(stub, false)
	if err != nil {
		return shim.Error(err.Error())
	}

	sort := "pending_time"
	if len(params) > 1 {
		sort = params[1]
	}

	bookmark := ""
	if len(params) > 2 {
		bookmark = params[2]
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

	bb := NewBalanceStub(stub)
	res, err := bb.GetQueryPendingBalances(addr.String(), sort, bookmark)
	if err != nil {
		return responseError(err, "failed to get pending balances")
	}

	data, err := json.Marshal(res)
	if err != nil {
		return responseError(err, "failed to marshal pending balances")
	}
	return shim.Success(data)
}

// params[0] : pending balance id
func balancePendingWithdraw(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 1 {
		return shim.Error("incorrect number of parameters. expecting 1")
	}

	ts, err := txtime.GetTime(stub)
	if err != nil {
		return responseError(err, "failed to get the timestamp")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	// pending balance
	bb := NewBalanceStub(stub)
	pb, err := bb.GetPendingBalance(params[0])
	if err != nil {
		return responseError(err, "failed to get the pending balance")
	}
	if pb.PendingTime.After(*ts) {
		return shim.Error("too early to withdraw")
	}

	// account
	addr, _ := ParseAddress(pb.Account) // err is nil
	ab := NewAccountStub(stub, addr.Code)
	account, err := ab.GetAccount(addr)
	if err != nil {
		return responseError(err, "failed to get the account")
	}
	if !account.HasHolder(kid) {
		return shim.Error("invoker is not holder")
	}
	if account.IsSuspended() {
		return shim.Error("the account is suspended")
	}

	// withdraw
	log, err := bb.Withdraw(pb)
	if err != nil {
		return responseError(err, "failed to withdraw")
	}

	data, err := json.Marshal(log)
	if err != nil {
		return responseError(err, "failed to marshal the log")
	}

	return shim.Success(data)
}
