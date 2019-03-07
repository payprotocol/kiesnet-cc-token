// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
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

// CreateKey _
func (fb *FeeStub) CreateKey(id string) string {
	return "FEE_" + id
}

// CreateFee creates new fee utxo of given amount and puts the state.
// If give amount is zero, it puts nothing and returns nil.
func (fb *FeeStub) CreateFee(payer *Address, amount Amount) (*Fee, error) {
	if amount.Int.Cmp(big.NewInt(0)) == 0 {
		return nil, nil
	}

	ts, err := txtime.GetTime(fb.stub)
	if nil != err {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	fee := &Fee{}
	fee.DOCTYPEID = payer.Code
	fee.FeeID = fmt.Sprintf("%d%s", ts.UnixNano(), fb.stub.GetTxID())
	fee.Account = payer.String()
	fee.Amount = amount
	fee.CreatedTime = ts

	err = fb.PutFee(fee)
	if nil != err {
		return nil, errors.Wrap(err, "failed to create fee")
	}

	return fee, nil
}

//ISSUE : Do we have to fetch individual Fee via chaincode call?
// // GetFee _
// func (fb *FeeStub) GetFee(id string) (*Fee, error) {
// 	data, err := fb.GetFeeState(id)
// 	if nil != err {
// 		return nil, err
// 	}
// 	// data is not nil
// 	fee := &Fee{}
// 	err = json.Unmarshal(data, fee)
// 	if nil != err {
// 		return nil, errors.Wrap(err, "failed to unmarshal the fee")
// 	}
// 	return fee, nil
// }

// // GetFeeState _
// func (fb *FeeStub) GetFeeState(id string) ([]byte, error) {
// 	data, err := fb.stub.GetState(fb.CreateKey(id))
// 	if nil != err {
// 		return nil, errors.Wrap(err, "failed to get the fee state")
// 	}
// 	if data != nil {
// 		return data, nil
// 	}
// 	return nil, NotExistedFeeError{id: id}
// }

// PutFee _
func (fb *FeeStub) PutFee(fee *Fee) error {
	data, err := json.Marshal(fee)
	if nil != err {
		return errors.Wrap(err, "failed to marshal the fee")
	}
	err = fb.stub.PutState(fb.CreateKey(fee.FeeID), data)
	if nil != err {
		return errors.Wrap(err, "failed to put the fee state")
	}
	return nil
}

// GetFeePolicy returns fee policy of given code token.
// If target policy does not exist in cache, invoke knt.
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
				rate := rm[0]
				if _, ok := new(big.Float).SetString(rate); !ok {
					return nil, errors.New("failed to parse rate")
				}
				max := int64(0)
				if len(rm) > 1 {
					max, err = strconv.ParseInt(rm[1], 10, 64)
					if err != nil {
						return nil, err
					}
				}
				rates[kv[0]] = FeeRate{
					Rate:      rate,
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
func (fb *FeeStub) GetQueryFees(tokenCode, bookmark string, fetchSize int, stime, etime *txtime.Time) (*QueryResult, error) {
	if fetchSize < 1 {
		fetchSize = FeeFetchSize
	}
	if fetchSize > 200 {
		fetchSize = 200
	}
	query := ""
	if nil != stime || nil != etime {
		query = CreateQueryFeesByCodeAndTimes(tokenCode, stime, etime)
	} else {
		query = CreateQueryFeesByCode(tokenCode)
	}
	iter, meta, err := fb.stub.GetQueryResultWithPagination(query, int32(fetchSize), bookmark)
	if nil != err {
		return nil, err
	}
	defer iter.Close()
	return NewQueryResult(meta, iter)
}

// GetFeeSumByTime returns FeeSum from stime to etime.
func (fb *FeeStub) GetFeeSumByTime(tokenCode string, stime, etime *txtime.Time) (*FeeSum, error) {
	query := CreateQueryPruneFee(tokenCode, stime, etime)
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

// CalcFee returns calculated fee amount from transfer/pay amount
func (fb *FeeStub) CalcFee(feePolicy FeePolicy, fn, account string, amount Amount) (*Amount, error) {
	feeRate := feePolicy.Rates[fn]
	amountFloat := new(big.Float).SetInt(&amount.Int)
	// We've already checked validity of Rate on GetFeePolicy()
	feeRateFloat, _ := new(big.Float).SetString(feeRate.Rate)
	feeAmountFloat := new(big.Float).Mul(amountFloat, feeRateFloat)
	feeAmountString := strings.Split(feeAmountFloat.Text('f', 64), ".")[0] // floor
	feeAmountInt, _ := new(big.Int).SetString(feeAmountString, 10)

	if feeRate.MaxAmount > 0 { // 0 is unlimit
		feeMaxAmountBigInt := big.NewInt(feeRate.MaxAmount)
		if feeAmountInt.Cmp(feeMaxAmountBigInt) > 0 {
			feeAmountInt = feeMaxAmountBigInt
		}
	}

	feeAmount, err := NewAmount(feeAmountInt.String())
	if nil != err {
		return nil, err
	}

	return feeAmount, nil
}
