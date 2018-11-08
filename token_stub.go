// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"math/big"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
	"github.com/pkg/errors"
)

// TokenStub _
type TokenStub struct {
	stub shim.ChaincodeStubInterface
}

// NewTokenStub _
func NewTokenStub(stub shim.ChaincodeStubInterface) *TokenStub {
	tb := &TokenStub{}
	tb.stub = stub
	return tb
}

// CreateKey _
func (tb *TokenStub) CreateKey(code string) string {
	return "TKN_" + code
}

// CreateToken _
func (tb *TokenStub) CreateToken(code string, decimal int, maxSupply, supply *big.Int, holders stringset.Set) (*Token, error) {
	return nil, nil
}

// GetToken _
func (tb *TokenStub) GetToken(code string) (*Token, error) {
	data, err := tb.GetTokenState(code)
	if err != nil {
		return nil, err
	}
	// data is not nil
	token := &Token{}
	if err = json.Unmarshal(data, token); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal the token")
	}
	return token, nil
}

// GetTokenState _
func (tb *TokenStub) GetTokenState(code string) ([]byte, error) {
	data, err := tb.stub.GetState(tb.CreateKey(code))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the token state")
	}
	if data != nil {
		return data, nil
	}
	return nil, errors.Errorf("token '%s' is not issued", code)
}
