// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"fmt"
	"time"

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

// CreateLogKey _
func (bb *BalanceStub) CreateLogKey(log *BalanceLog) string {
	return fmt.Sprintf("BLOG_%s_%d", log.DOCTYPEID, log.CreatedTime.UnixNano())
}

// PutBalanceLog _
func (bb *BalanceStub) PutBalanceLog(log *BalanceLog) error {
	data, err := json.Marshal(log)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the balance log")
	}
	if err = bb.stub.PutState(bb.CreateLogKey(log), data); err != nil {
		return errors.Wrap(err, "failed to put the balance log state")
	}
	return nil
}

// CreatePendingKey _
func (bb *BalanceStub) CreatePendingKey(balance *PendingBalance) string {
	return fmt.Sprintf("PBLC_%s_%d", balance.DOCTYPEID, balance.CreatedTime.UnixNano())
}

// PutPendingBalance _
func (bb *BalanceStub) PutPendingBalance(balance *PendingBalance) error {
	data, err := json.Marshal(balance)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the pending balance")
	}
	if err = bb.stub.PutState(bb.CreatePendingKey(balance), data); err != nil {
		return errors.Wrap(err, "failed to put the pending balance state")
	}
	return nil
}

// Transfer _
func (bb *BalanceStub) Transfer(sender, receiver *Balance, amount Amount, memo string, pendingTime *time.Time) (*BalanceLog, error) {
	ts, err := txtime.GetTime(bb.stub)
	if err != nil {
		return nil, err
	}

	if pendingTime != nil && pendingTime.Before(*ts) { // time lock
		pb := &PendingBalance{
			DOCTYPEID:   receiver.DOCTYPEID,
			RID:         sender.DOCTYPEID,
			Amount:      amount,
			Memo:        memo,
			CreatedTime: ts,
			PendingTime: pendingTime,
		}
		if err = bb.PutPendingBalance(pb); err != nil {
			return nil, err
		}
	} else {
		receiver.Amount.Add(&amount) // deposit
		receiver.UpdatedTime = ts
		if err = bb.PutBalance(receiver); err != nil {
			return nil, err
		}
		rbl := &BalanceLog{
			DOCTYPEID:   receiver.DOCTYPEID,
			Type:        BalanceLogTypeReceive,
			RID:         sender.DOCTYPEID,
			Diff:        amount,
			Amount:      receiver.Amount,
			Memo:        memo,
			CreatedTime: ts,
		}
		if err = bb.PutBalanceLog(rbl); err != nil {
			return nil, err
		}
	}

	amount.Neg()
	sender.Amount.Add(&amount) // withdraw
	sender.UpdatedTime = ts
	if err = bb.PutBalance(sender); err != nil {
		return nil, err
	}
	sbl := &BalanceLog{
		DOCTYPEID:   sender.DOCTYPEID,
		Type:        BalanceLogTypeSend,
		RID:         receiver.DOCTYPEID,
		Diff:        amount,
		Amount:      sender.Amount,
		Memo:        memo,
		CreatedTime: ts,
	}
	if err = bb.PutBalanceLog(sbl); err != nil {
		return nil, err
	}

	return sbl, nil
}
