// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"fmt"
	"math/big"

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
func (fb *FeeStub) CalcFee(payer *Address, fn string, amount Amount) (*Amount, error) {
	token, err := NewTokenStub(fb.stub).GetToken(payer.Code)
	if err != nil {
		return nil, err
	}
	// The amount is 0 if policy does't exist.
	if token.FeePolicy == nil {
		return &Amount{Int: *big.NewInt(0)}, nil
	}
	if valid := isValidFn(fn); !valid {
		return nil, errors.New("invalid fee rate type")
	}
	// The amount is 0 if the fee payer is the target address of fee policy on transfer.
	if token.FeePolicy.TargetAddress == payer.String() {
		return &Amount{Int: *big.NewInt(0)}, nil
	}
	feeRate, ok := token.FeePolicy.Rates[fn]
	if !ok { // no such fn
		return &Amount{Int: *big.NewInt(0)}, nil
	}
	// We've already checked validity of Rate on GetFeePolicy()
	feeRateRat, _ := new(big.Rat).SetString(feeRate.Rate)
	// feeAmount = amount * rate
	feeAmount := &Amount{Int: *new(big.Int).Div(new(big.Int).Mul(&amount.Int, feeRateRat.Num()), feeRateRat.Denom())}
	if feeRate.MaxAmount == 0 { // unlimited fee
		return feeAmount, nil
	}
	// limited to MaxAmount
	maxAmount := &Amount{Int: *big.NewInt(feeRate.MaxAmount)}
	if feeAmount.Cmp(maxAmount) > 0 { // feeAmount is gt.
		return maxAmount, nil
	}
	// maxAmount is gte.
	return feeAmount, nil
}
