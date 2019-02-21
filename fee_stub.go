// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

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
