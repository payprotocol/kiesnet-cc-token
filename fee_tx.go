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

// ISSUE : token/fee/list?
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
	// genesis account is never suspended

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
							return shim.Error("invalid end time: need seconds since 1970")
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
	res, err := fb.GetQueryFees(token.DOCTYPEID, bookmark, fetchSize, stime, etime)
	if nil != err {
		return responseError(err, "failed to get fees")
	}

	data, err := json.Marshal(res)
	if nil != err {
		return responseError(err, "failed to marshal fees")
	}

	return shim.Success(data)
}

// ISSUE : token/fee/prune?
// Only genesis account holder is able to prune fee list.
// params[0] : token code
// params[1] : 10 minutes limit flag. if the value is true, 10 minutes check is activated.
// params[2] : optional. end time
func feePrune(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 2 {
		return shim.Error("incorrect number of parameters. expecting 2+")
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
	kid, err := kid.GetID(stub, true)
	if nil != err {
		return shim.Error(err.Error())
	}

	// genesis account
	addr, _ := ParseAddress(token.GenesisAccount) // err is nil
	ab := NewAccountStub(stub, code)
	account, err := ab.GetAccount(addr)
	if nil != err {
		return responseError(err, "failed to get the genesis account")
	}
	if !account.HasHolder(kid) { // authority
		return shim.Error("no authority")
	}
	// genesis account is never suspended

	stime := txtime.Unix(0, 0)
	if len(token.LastPrunedFeeID) > 0 {
		//TODO check fee id parsing logic
		s, err := strconv.ParseInt(token.LastPrunedFeeID[0:10], 10, 64)
		if nil != err {
			return responseError(err, "failed to get timestamp of last pruned fee")
		}
		//TODO check fee id parsing logic
		n, err := strconv.ParseInt(token.LastPrunedFeeID[10:19], 10, 64)
		if nil != err {
			return responseError(err, "failed to get nanoseconds of last pruned fee")
		}
		stime = txtime.Unix(s, n)
	}

	ts, err := txtime.GetTime(stub)
	if nil != err {
		return responseError(err, "failed to get tx timestamp")
	}

	var etime *txtime.Time
	if len(params) > 2 {
		seconds, err := strconv.ParseInt(params[2], 10, 64)
		if nil != err {
			return responseError(err, "failed to parse the end time")
		}
		etime = txtime.Unix(seconds, 0)
	} else {
		etime = ts
	}

	safely, err := strconv.ParseBool(params[1])
	if nil != err {
		return shim.Error("malformed boolean flag")
	}

	if safely {
		// safe time is current transaction time minus 10 minutes. this is to prevent missing pay(s) because of the time differences(+/- 5min) on different servers/devices
		safeTime := txtime.New(ts.Add(-6e+11))
		if nil == etime || etime.Cmp(safeTime) > 0 {
			etime = safeTime
		}
	}

	// calculate fee sum
	fb := NewFeeStub(stub)
	feeSum, err := fb.GetFeeSumByTime(code, stime, etime)
	if nil != err {
		return responseError(err, "failed to get fees to prune")
	}

	bb := NewBalanceStub(stub)
	bal, err := bb.GetBalance(account.GetID())
	if nil != err {
		return responseError(err, "failed to get the genesis account balance")
	}

	if feeSum.Count > 0 {
		bal.Amount.Add(feeSum.Sum)
		bal.UpdatedTime = ts
		err = bb.PutBalance(bal)
		if nil != err {
			return responseError(err, "failed to update genesis account balance")
		}
		token.LastPrunedFeeID = feeSum.End
		err = tb.PutToken(token)
		if nil != err {
			return responseError(err, "failed to update token")
		}
	}

	// balance log
	pruneLog := NewBalanceLogTypePruneFee(bal, *feeSum.Sum, feeSum.Start, feeSum.End)
	pruneLog.CreatedTime = ts
	err = bb.PutBalanceLog(pruneLog)
	if nil != err {
		return responseError(err, "failed to save balance log")
	}

	data, err := json.Marshal(feeSum)
	if nil != err {
		return shim.Error(err.Error())
	}
	return shim.Success(data)
}

func feeMock(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 2 {
		return shim.Error("incorrect number of parameters. expecting 2")
	}

	// authentication
	kid, err := kid.GetID(stub, false)
	if nil != err {
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

	amount, err := NewAmount(params[1])
	if nil != err {
		return shim.Error(err.Error())
	}

	fb := NewFeeStub(stub)
	fee, err := fb.CreateFee(addr, *amount)
	if nil != err {
		return shim.Error("failed to create fee")
	}

	data, err := json.Marshal(fee)
	if nil != err {
		logger.Debug(err.Error())
		return shim.Error("failed to marshal the fee")
	}

	return shim.Success(data)
}
