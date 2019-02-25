// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// FeePruneSize is number of fee utxo that one prune request can handle.
const FeePruneSize = 900

// FeeFetchSize _
const FeeFetchSize = 20

// FeeStub _
type FeeStub struct {
	stub shim.ChaincodeStubInterface
}

// NewFeeStub _
func NewFeeStub(stub shim.ChaincodeStubInterface) *FeeStub {
	return &FeeStub{stub}
}

// GetFeePolicy _
func (fb *FeeStub) GetFeePolicy(code string) (*FeePolicy, error) {
	// if nil == _feePolicies {
	// 	_feePolicies = map[string]*FeePolicy{}
	// 	return nil, nil
	// }
	policy := _feePolicies[code]
	if nil == policy {
		logger.Debugf("caching %s fee policy", code)
		// check issued token
		tb := NewTokenStub(fb.stub)
		token, err := tb.GetToken(code)
		if err != nil {
			return nil, err
		}
		// get token meta
		_, _, _, fee, err := getValidatedTokenMeta(fb.stub, code)
		if err != nil {
			return nil, err
		}
		// fees -> map
		rates := map[string]FeeRate{}
		fees := strings.Split(fee, ";")
		for _, f := range fees {
			kv := strings.Split(f, "=")
			if len(kv) > 1 {
				rm := strings.Split(kv[1], ",")
				rate, err := strconv.ParseFloat(rm[0], 32)
				if err != nil {
					return nil, err
				}
				max := int64(0)
				if len(rm) > 1 {
					max, err = strconv.ParseInt(rm[1], 10, 64)
					if err != nil {
						return nil, err
					}
				}
				rates[kv[0]] = FeeRate{
					Rate:      float32(rate),
					MaxAmount: max,
				}
			}
		}

		policy = &FeePolicy{
			TargetAddress: token.GenesisAccount,
			Rates:         rates,
		}
		_feePolicies[code] = policy
	}

	return policy, nil
}

// RefreshFeePolicy _
func (fb *FeeStub) RefreshFeePolicy(code string) (*FeePolicy, error) {
	_feePolicies[code] = nil
	return fb.GetFeePolicy(code)
}

// GetQueryFees _
func (fb *FeeStub) GetQueryFees(id, bookmark string, fetchSize int, stime, etime *txtime.Time) (*QueryResult, error) {
	if fetchSize < 1 {
		fetchSize = FeeFetchSize
	}
	if fetchSize > 200 {
		fetchSize = 200
	}
	query := ""
	if nil != stime || nil != etime {
		query = CreateQueryFeesByIDAndTimes(id, stime, etime)
	} else {
		query = CreateQueryFeesByID(id)
	}
	iter, meta, err := fb.stub.GetQueryResultWithPagination(query, int32(fetchSize), bookmark)
	if nil != err {
		return nil, err
	}
	defer iter.Close()
	return NewQueryResult(meta, iter)
}

// GetFeeSumByTime returns FeeSum from stime to etime.
func (fb *FeeStub) GetFeeSumByTime(id string, stime, etime *txtime.Time) (*FeeSum, error) {
	query := CreateQueryPruneFee(id, stime, etime)
	iter, err := fb.stub.GetQueryResult(query)
	if nil != err {
		return nil, err
	}
	defer iter.Close()

	feeSum := &FeeSum{HasMore: false}
	fee := &Fee{}
	cnt := 0
	sum, _ := NewAmount("0")

	for iter.HasNext() {
		cnt++
		kv, err := iter.Next()
		if nil != err {
			return nil, err
		}
		err = json.Unmarshal(kv.Value, fee)
		if nil != err {
			return nil, err
		}
		if 1 == cnt {
			feeSum.Start = fee.FeeID
		}
		if FeePruneSize+1 == cnt {
			feeSum.HasMore = true
			cnt--
			break
		}
		sum = sum.Add(&fee.Amount)
		feeSum.End = fee.FeeID
	}
	feeSum.Count = cnt
	feeSum.Sum = sum
	return feeSum, nil
}
