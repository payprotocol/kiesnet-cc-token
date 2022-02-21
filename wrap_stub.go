package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
)

// WrapStub _
type WrapStub struct {
	stub shim.ChaincodeStubInterface
}

// NewWrapStub _
func NewWrapStub(stub shim.ChaincodeStubInterface) *WrapStub {
	return &WrapStub{stub}
}

// CreateWrapKey _
func (wb *WrapStub) CreateWrapKey(txid string) string {
	return fmt.Sprintf("WRAP_%s", txid)
}

// CreateUnwrapKey _
func (wb *WrapStub) CreateUnwrapKey(extTxID string) string {
	return fmt.Sprintf("UNWRAP_%s", extTxID)
}

// GetWrap _
func (wb *WrapStub) GetWrap(txid string) (*Wrap, error) {
	data, err := wb.stub.GetState(wb.CreateWrapKey(txid))
	if err != nil {
		return nil, err
	}
	if nil == data {
		return nil, errors.New("wrap is not exist")
	}
	wrap := &Wrap{}
	if err = json.Unmarshal(data, wrap); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal the wrap")
	}
	return wrap, nil
}

// PutWrap _
func (wb *WrapStub) PutWrap(wrap *Wrap) error {
	data, err := json.Marshal(wrap)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the wrap")
	}
	if err = wb.stub.PutState(wb.CreateWrapKey(wrap.DOCTYPEID), data); err != nil {
		return errors.Wrap(err, "failed to put the wrap state")
	}
	return nil
}

// PutUnwrap _
func (wb *WrapStub) PutUnwrap(unwrap *Unwrap) error {
	data, err := json.Marshal(unwrap)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the unwrap")
	}
	if err = wb.stub.PutState(wb.CreateUnwrapKey(unwrap.DOCTYPEID), data); err != nil {
		return errors.Wrap(err, "failed to put the unwrap state")
	}
	return nil
}

// Wrap _
func (wb *WrapStub) Wrap(sender *Balance, amount Amount, extCode, extID, memo string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	wrap := &Wrap{
		DOCTYPEID: wb.stub.GetTxID(),
		Address:   sender.GetID(),
		Amount:    amount,
		ExtCode:   extCode,
		ExtID:     extID,
	}
	if err = wb.PutWrap(wrap); err != nil {
		return nil, err
	}

	bb := NewBalanceStub(wb.stub)

	amount.Neg()
	sender.Amount.Add(&amount)
	sender.UpdatedTime = ts
	if err = bb.PutBalance(sender); err != nil {
		return nil, err
	}

	sbl := NewBalanceWrapLog(sender, amount, extCode, extID, memo)
	sbl.CreatedTime = ts
	if err = bb.PutBalanceLog(sbl); err != nil {
		return nil, err
	}

	return sbl, nil
}

// WrapComplete _
//func (wb *WrapStub) WrapComplete(wrapKey string, wrapper *Balance, amount, fee Amount, extCode, extID, extTxID string) (*BalanceLog, error) {
func (wb *WrapStub) WrapComplete(wrap *Wrap, wBal *Balance, fee Amount, extTxID string) (*BalanceLog, error) {
	if wrap.CompleteTxID != "" {
		return nil, DuplicateWrapCompleteError{}
	}

	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	diff := wrap.Amount.Copy()
	diff.Add(fee.Copy().Neg())
	if diff.Sign() <= 0 {
		return nil, errors.New("wrap amount is less than or equal to fee")
	}

	// update wrap state
	if extTxID != "" { // successful wrap
		wrap.CompleteTxID = extTxID
	} else { // impossible wrap
		wrap.CompleteTxID = wrap.DOCTYPEID
	}
	if err = wb.PutWrap(wrap); err != nil {
		return nil, err
	}

	bb := NewBalanceStub(wb.stub)

	wBal.Amount.Add(diff)
	wBal.UpdatedTime = ts
	if err = bb.PutBalance(wBal); err != nil {
		return nil, err
	}

	sbl := NewBalanceWrapCompleteLog(wBal, wrap, fee.Copy())
	sbl.CreatedTime = ts
	if err = bb.PutBalanceLog(sbl); err != nil {
		return nil, err
	}

	_, err = NewFeeStub(wb.stub).CreateFee(wBal.GetID(), fee)
	if err != nil {
		return nil, err
	}

	return sbl, nil
}

// Unwrap _
func (wb *WrapStub) Unwrap(wrapper, receiver *Balance, amount Amount, extCode, extID, extTxID string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	bb := NewBalanceStub(wb.stub)

	receiver.Amount.Add(&amount)
	receiver.UpdatedTime = ts
	if err = bb.PutBalance(receiver); err != nil {
		return nil, err
	}

	rbl := NewBalanceUnwrapLog(receiver, amount, extCode, extID, extTxID)
	rbl.CreatedTime = ts
	if err = bb.PutBalanceLog(rbl); err != nil {
		return nil, err
	}

	amount.Neg()
	wrapper.Amount.Add(&amount)
	wrapper.UpdatedTime = ts
	if err = bb.PutBalance(wrapper); err != nil {
		return nil, err
	}
	wbl := NewBalanceUnwrapCompleteLog(wrapper, receiver, amount, extCode, extTxID)
	wbl.CreatedTime = ts
	if err = bb.PutBalanceLog(wbl); err != nil {
		return nil, err
	}
	return rbl, nil
}

// UnwrapImpossible _
func (wb *WrapStub) UnwrapImpossible(wrapper *Balance, extCode, extID, extTxID string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	bb := NewBalanceStub(wb.stub)
	wbl := NewBalanceUnwrapCompleteLog(wrapper, wrapper, *ZeroAmount(), extCode, extTxID)
	wbl.CreatedTime = ts
	if err = bb.PutBalanceLog(wbl); err != nil {
		return nil, err
	}
	return wbl, nil
}

// WrapPendingBalance wrap the sender's pending balance. (multi-sig contract)
func (wb *WrapStub) WrapPendingBalance(pb *PendingBalance, sender *Balance, extCode, extID string) (*Wrap, error) {
	wrap := &Wrap{
		DOCTYPEID: wb.stub.GetTxID(),
		Address:   sender.GetID(),
		Amount:    pb.Amount,
		ExtCode:   extCode,
		ExtID:     extID,
	}
	if err := wb.PutWrap(wrap); err != nil {
		return nil, err
	}

	// remove pending balance
	if err := NewBalanceStub(wb.stub).DeletePendingBalance(pb); err != nil {
		return nil, errors.Wrap(err, "failed to delete the pending balance")
	}

	return wrap, nil
}
