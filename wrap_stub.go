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

// WrapTx
type WrapTx struct{}

// NewWrapStub
func NewWrapStub(stub shim.ChaincodeStubInterface) *WrapStub {
	return &WrapStub{stub}
}

// CreateWrapTxKey
func (wb *WrapStub) CreateWrapTxKey(prefix, txID string) string {
	return fmt.Sprintf("%s_%s", prefix, txID)
}

// createWrapTx
func (wb *WrapStub) createWrapTx(wrapID string) error {
	data, err := json.Marshal(&WrapTx{})
	if err != nil {
		return errors.Wrap(err, "failed to marshal wrap tx")
	}
	ok, err := wb.loadWrapTx(wrapID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve the wrap tx")
	}
	if ok {
		return DuplicateWrapTxError{}
	}
	if err = wb.stub.PutState(wrapID, data); err != nil {
		return errors.Wrap(err, "failed to create wrap tx")
	}
	return nil
}

// loadWrapTx _ ok means exists(handling mvcc conflict)
func (wb *WrapStub) loadWrapTx(id string) (ok bool, err error) {
	ok = false
	data, err := wb.stub.GetState(id)
	if err != nil {
		return
	}
	wrap := &WrapTx{}
	if data != nil {
		if err = json.Unmarshal(data, wrap); err != nil {
			return
		}
		ok = true
	}
	return
}

// Wrap _
func (wb *WrapStub) Wrap(sender *Balance, amount Amount, tokenCode, extID, memo string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	bb := NewBalanceStub(wb.stub)

	amount.Neg()
	sender.Amount.Add(&amount)
	sender.UpdatedTime = ts
	if err = bb.PutBalance(sender); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	sbl := NewBalanceWrapLog(sender, amount, tokenCode, extID, memo)
	sbl.CreatedTime = ts
	if err = bb.PutBalanceLog(sbl); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	return sbl, nil
}

// WrapComplete _
func (wb *WrapStub) WrapComplete(wrapper *Balance, amount, fee Amount, tokenCode, extID, txID string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	key := wb.CreateWrapTxKey("WRAP_COMPL", txID)
	err = wb.createWrapTx(key)
	if err != nil {
		return nil, err
	}

	bb := NewBalanceStub(wb.stub)

	diff := amount.Copy()
	diff.Neg()
	diff.Add(fee.Copy().Neg())
	if diff.Cmp(ZeroAmount()) < 0 { // diff is less than 0
		return nil, errors.New("wrap amount is less than fee")
	}

	wrapper.Amount.Add(diff)
	wrapper.UpdatedTime = ts
	if err = bb.PutBalance(wrapper); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	sbl := NewBalanceWrapCompleteLog(wrapper, *diff, tokenCode, extID)
	sbl.CreatedTime = ts
	if err = bb.PutBalanceLog(sbl); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	_, err = NewFeeStub(wb.stub).CreateFee(wrapper.GetID(), fee)
	if err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	return sbl, nil
}

// Unwrap _
func (wb *WrapStub) Unwrap(wrapper, receiver *Balance, amount Amount, tokenCode, extID, extTxID string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	key := wb.CreateWrapTxKey("UNWRAP", extTxID)
	err = wb.createWrapTx(key)
	if err != nil {
		return nil, err
	}

	bb := NewBalanceStub(wb.stub)

	receiver.Amount.Add(&amount)
	receiver.UpdatedTime = ts
	if err = bb.PutBalance(receiver); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	rbl := NewBalanceUnWrapLog(receiver, amount, tokenCode, extID, extTxID)
	rbl.CreatedTime = ts
	if err = bb.PutBalanceLog(rbl); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	amount.Neg()
	wrapper.Amount.Add(&amount)
	wrapper.UpdatedTime = ts
	if err = bb.PutBalance(wrapper); err != nil {
		return nil, err
	}
	wbl := NewBalanceUnWrapCompleteLog(wrapper, amount, tokenCode, extID, extTxID)
	wbl.CreatedTime = ts
	if err = bb.PutBalanceLog(wbl); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}
	return rbl, nil
}

// UnwrapComplete _
func (wb *WrapStub) UnwrapComplete(wrapper *Balance, tokenCode, extID, extTxID string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	key := wb.CreateWrapTxKey("UNWRAP", extTxID)
	err = wb.createWrapTx(key)
	if err != nil {
		return nil, err
	}

	bb := NewBalanceStub(wb.stub)
	wbl := NewBalanceUnWrapCompleteLog(wrapper, *ZeroAmount(), tokenCode, extID, extTxID)
	wbl.CreatedTime = ts
	if err = bb.PutBalanceLog(wbl); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}
	return wbl, nil
}

// WrapPendingBalance wrap the sender's pending balance. (multi-sig contract)
func (wb *WrapStub) WrapPendingBalance(pb *PendingBalance, sender *Balance, tokenCode, extID, memo string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	bb := NewBalanceStub(wb.stub)

	sbl := NewBalanceWrapLog(sender, *pb.Amount.Copy().Neg(), tokenCode, extID, memo)
	sbl.CreatedTime = ts
	if err = bb.PutBalanceLog(sbl); err != nil {
		return nil, err
	}

	// remove pending balance
	if err = wb.stub.DelState(bb.CreatePendingKey(pb.DOCTYPEID)); err != nil {
		return nil, errors.Wrap(err, "failed to delete the pending balance")
	}

	return sbl, nil
}
