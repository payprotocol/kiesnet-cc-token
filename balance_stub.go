// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
)

// BalanceStub _
type BalanceStub struct {
	stub shim.ChaincodeStubInterface
}

// NewBalanceStub _
func NewBalanceStub(stub shim.ChaincodeStubInterface) *BalanceStub {
	return &BalanceStub{stub}
}

// CreateKey _
func (bb *BalanceStub) CreateKey(id string) string {
	return "BLC_" + id
}

// CreateBalance _
func (bb *BalanceStub) CreateBalance(id string) (*Balance, error) {
	ts, err := txtime.GetTime(bb.stub)
	if err != nil {
		return nil, err
	}

	balance := &Balance{}
	balance.DOCTYPEID = id
	balance.CreatedTime = ts
	balance.UpdatedTime = ts
	if err = bb.PutBalance(balance); err != nil {
		return nil, errors.Wrap(err, "failed to create a balance")
	}

	return balance, nil
}

// GetBalance _
func (bb *BalanceStub) GetBalance(id string) (*Balance, error) {
	data, err := bb.GetBalanceState(id)
	if err != nil {
		return nil, err
	}
	// data is not nil
	balance := &Balance{}
	if err = json.Unmarshal(data, balance); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal the balance")
	}
	return balance, nil
}

// GetBalanceState _
func (bb *BalanceStub) GetBalanceState(id string) ([]byte, error) {
	data, err := bb.stub.GetState(bb.CreateKey(id))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the balance state")
	}
	if data != nil {
		return data, nil
	}
	// ISSUE: failover
	logger.Criticalf("nil balance: %s", id)
	return nil, errors.New("balance is not exists")
}

// PutBalance _
func (bb *BalanceStub) PutBalance(balance *Balance) error {
	data, err := json.Marshal(balance)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the balance")
	}
	if err = bb.stub.PutState(bb.CreateKey(balance.DOCTYPEID), data); err != nil {
		return errors.Wrap(err, "failed to put the balance state")
	}
	return nil
}

// func (bb *BalanceStub) Deposit(amount Amount, memo string) error {
// 	return nil
// }

// func (bb *BalanceStub) Withdraw(amount Amount, memo string) error {
// 	return nil
// }
