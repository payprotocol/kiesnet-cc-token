package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
)

// WrapStub
type WrapStub struct {
	stub shim.ChaincodeStubInterface
}

// NewWrapStub
func NewWrapStub(stub shim.ChaincodeStubInterface) *WrapStub {
	return &WrapStub{stub}
}

// GetUnWrapId
func GetUnWrapId(txID string) string {
	return fmt.Sprintf("UNWRAP_%s", txID)
}

// createUnWrap
func (wb *WrapStub) createUnWrap(wrapID string) error {
	data, err := json.Marshal(&Wrap{})
	if err != nil {
		return errors.Wrap(err, "failed to marshal unwrap")
	}
	ok, err := wb.loadUnWrap(wrapID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve the unwrap")
	}
	if ok {
		return DuplicateWrapError{}
	}
	if err = wb.stub.PutState(wrapID, data); err != nil {
		return errors.Wrap(err, "failed to create unwrap")
	}
	return nil
}

// loadUnWrap _ ok means exists
func (wb *WrapStub) loadUnWrap(id string) (ok bool, err error) {
	ok = false
	data, err := wb.getWrapState(id)
	if err != nil {
		return
	}
	wrap := &Wrap{}
	if data != nil {
		if err = json.Unmarshal(data, wrap); err != nil {
			return
		}
		ok = true
	}
	return
}

// getWrapState
func (wb *WrapStub) getWrapState(id string) ([]byte, error) {
	data, err := wb.stub.GetState(id)
	if err != nil {
		logger.Debug(err.Error())
		return nil, errors.Wrap(err, "failed to get the wrap state")
	}
	if data != nil {
		return data, nil
	}
	return nil, nil
}

// Wrap _
// bCode : bridge token code e.g. WPCI
func (wb *WrapStub) Wrap(sender *Balance, amount, fee Amount, tokenCode, extID string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	amount.Neg()
	sender.Amount.Add(&amount)
	sender.Amount.Add(fee.Copy().Neg())

	// sender balance change
	sender.UpdatedTime = ts
	if err = NewBalanceStub(wb.stub).PutBalance(sender); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}
	// sender balance log
	sbl := NewBalanceWrapLog(sender, amount, &fee, tokenCode, extID)
	sbl.CreatedTime = ts
	if err = NewBalanceStub(wb.stub).PutBalanceLog(sbl); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	_, err = NewFeeStub(wb.stub).CreateFee(sender.GetID(), fee)
	if err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	return sbl, nil
}

// UnWrap _
// bCode : bridge token code e.g. WPCI
func (wb *WrapStub) UnWrap(receiver *Balance, amount Amount, tokenCode, extID, extTxID string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	wrapID := GetUnWrapId(extTxID)

	if err = wb.createUnWrap(wrapID); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	// receiver balance change
	receiver.Amount.Add(&amount)
	receiver.UpdatedTime = ts
	if err = NewBalanceStub(wb.stub).PutBalance(receiver); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}
	// receiver balance log
	rbl := NewBalanceUnWrapLog(receiver, amount, tokenCode, extID, extTxID)
	rbl.CreatedTime = ts
	if err = NewBalanceStub(wb.stub).PutBalanceLog(rbl); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	return rbl, nil
}

// WrapPendingBalance wrap the sender's pending balance. (multi-sig contract)
func (wb *WrapStub) WrapPendingBalance(pb *PendingBalance, sender *Balance, tokenCode, extID string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	// fee gen
	if pb.Fee != nil {
		if _, err := NewFeeStub(wb.stub).CreateFee(pb.Account, *pb.Fee); err != nil {
			return nil, err
		}
	}

	// sender balance log
	sbl := NewBalanceWrapLog(sender, pb.Amount, pb.Fee, tokenCode, extID)
	sbl.CreatedTime = ts
	if err = NewBalanceStub(wb.stub).PutBalanceLog(sbl); err != nil {
		return nil, err
	}

	// remove pending balance
	if err = wb.stub.DelState(NewBalanceStub(wb.stub).CreatePendingKey(pb.DOCTYPEID)); err != nil {
		return nil, errors.Wrap(err, "failed to delete the pending balance")
	}

	return sbl, nil
}
