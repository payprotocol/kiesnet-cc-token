// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// BalanceStub _
type BalanceStub struct {
	stub shim.ChaincodeStubInterface
}

// NewBalanceStub _
func NewBalanceStub(stub shim.ChaincodeStubInterface) *BalanceStub {
	bb := &BalanceStub{}
	bb.stub = stub
	return bb
}

// CreateKey _
func (bb *BalanceStub) CreateKey(id string) string {
	return "BLC_" + id
}
