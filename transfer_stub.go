// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// TransferStub _
type TransferStub struct {
	stub shim.ChaincodeStubInterface
}

// NewTransferStub _
func NewTransferStub(stub shim.ChaincodeStubInterface) *TransferStub {
	return &TransferStub{stub}
}

// CreateLogKey _
func (tb *TransferStub) CreateLogKey(code string) string {
	return "TRL_" + code
}

// func (tb *TransferStub) Contract(sbal, rbal *Balance, amount *Amount, memo string, timeLock *time.Time, expiry int, singers *stringset.Set) {
// }

// Transfer _
func (tb *TransferStub) Transfer(from, to *Balance, amount *Amount, memo string, withdrawal *time.Time) ([]byte, error) {
	// ts, err := txtime.GetTime(tb.stub)
	// if err != nil {
	// 	return nil, err
	// }

	// if from != nil {
	// 	from.Amount.Add(amount.Neg())
	// }

	return nil, nil

}
