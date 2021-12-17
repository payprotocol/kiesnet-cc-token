package main

import (
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

// CreateUnwrapKey
func (wb *WrapStub) CreateUnwrapKey(txID string) string {
	return "UNWRAP_" + txID
}

// Wrap _
// bCode : bridge token code e.g. WPCI
func (wb *WrapStub) Wrap(sender *Balance, amount, fee Amount, tokenCode, extID string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	bb := NewBalanceStub(wb.stub)

	amount.Neg()
	sender.Amount.Add(&amount)
	sender.Amount.Add(fee.Copy().Neg())
	sender.UpdatedTime = ts
	if err = bb.PutBalance(sender); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	sbl := NewBalanceWrapLog(sender, amount, &fee, tokenCode, extID)
	sbl.CreatedTime = ts
	if err = bb.PutBalanceLog(sbl); err != nil {
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

// Unwrap _
// bCode : bridge token code e.g. WPCI
func (wb *WrapStub) Unwrap(receiver *Balance, amount Amount, tokenCode, extID, extTxID string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	key := wb.CreateUnwrapKey(extTxID)
	data, err := wb.stub.GetState(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve the unwrap")
	}
	if data != nil {
		return nil, DuplicateUnwrapError{}
	}
	if err = wb.stub.PutState(key, []byte{}); err != nil {
		return nil, errors.Wrap(err, "failed to create unwrap")
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

	return rbl, nil
}

// WrapPendingBalance wrap the sender's pending balance. (multi-sig contract)
func (wb *WrapStub) WrapPendingBalance(pb *PendingBalance, sender *Balance, tokenCode, extID string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	if pb.Fee != nil {
		if _, err := NewFeeStub(wb.stub).CreateFee(pb.Account, *pb.Fee); err != nil {
			return nil, err
		}
	}

	bb := NewBalanceStub(wb.stub)

	sbl := NewBalanceWrapLog(sender, *pb.Amount.Copy().Neg(), pb.Fee, tokenCode, extID)
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
