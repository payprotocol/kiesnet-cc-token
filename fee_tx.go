// Copyright Key Inside Co., Ltd. 2019 All Rights Reserved.

package main

import (
	"encoding/json"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/kid"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// ISSUE : Only genesis account holder should be able to query fee list?
// params[0] : token code
// params[1] : optional. bookmark
// params[2] : optional. fetch size (if less than 1, default size. max 200)
// params[3] : optional. start time (timestamp represented by int64 seconds)
// params[4] : optional. end time (timestamp represented by in64 seconds)
func feeList(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 1 {
		return shim.Error("incorrect number of parameters. expecting 1+")
	}

	code, err := ValidateTokenCode(params[0])
	if nil != err {
		return shim.Error(err.Error())
	}

	// token
	tb := NewTokenStub(stub)
	token, err := tb.GetToken(code)
	if nil != err {
		return responseError(err, "failed to get the token")
	}

	// authentication
	_, err = kid.GetID(stub, false)
	if nil != err {
		return shim.Error(err.Error())
	}

	// genesis account
	// addr, _ := ParseAddress(token.GenesisAccount) // err is nil
	// ab := NewAccountStub(stub, code)
	// account, err := ab.GetAccount(addr)
	// if nil != err {
	// 	return responseError(err, "failed to get the genesis account")
	// }
	// if !account.HasHolder(kid) { // authority
	// 	return shim.Error("no authority")
	// }

	bookmark := ""
	fetchSize := 0
	var stime, etime *txtime.Time
	// bookmark
	if len(params) > 1 {
		bookmark = params[1]
		// fetch size
		if len(params) > 2 {
			fetchSize, err = strconv.Atoi(params[2])
			if nil != err {
				return shim.Error("invalid fetch size")
			}
			// start time
			if len(params) > 3 {
				if len(params[3]) > 0 {
					seconds, err := strconv.ParseInt(params[3], 10, 64)
					if nil != err {
						return shim.Error("invalid start time: need seconds since 1970")
					}
					stime = txtime.Unix(seconds, 0)
				}
				// end time
				if len(params) > 4 {
					if len(params[4]) > 0 {
						seconds, err := strconv.ParseInt(params[4], 10, 64)
						if nil != err {
							return shim.Error("invalid end t ime: need seconds since 1970")
						}
						etime = txtime.Unix(seconds, 0)
						if nil != stime && stime.Cmp(etime) >= 0 {
							return shim.Error("invalid time parameters")
						}
					}
				}
			}
		}
	}

	fb := NewFeeStub(stub)
	res, err := fb.GetQueryFees(token.GenesisAccount, bookmark, fetchSize, stime, etime)
	if nil != err {
		return responseError(err, "failed to get fees")
	}

	data, err := json.Marshal(res)
	if nil != err {
		return responseError(err, "failed to marshal fees")
	}

	return shim.Success(data)
}
