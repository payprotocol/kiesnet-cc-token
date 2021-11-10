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

// GetWrapId
func GetWrapId(receiverID, txID string) string {
	return fmt.Sprintf("WRAP_%s_%s", receiverID, txID)
}

// NewWrapStub
func NewWrapStub(stub shim.ChaincodeStubInterface) *WrapStub {
	return &WrapStub{stub}
}

// CreateWrap
func (wb *WrapStub) CreateWrap(wrap *Wrap) error {
	data, err := json.Marshal(wrap)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the wrap")
	}
	ok, err := wb.LoadWrap(wrap.DOCTYPEID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve the wrap")
	}
	if ok {
		return errors.New("wrap aleady exists")
	}
	if err = wb.stub.PutState(wrap.DOCTYPEID, data); err != nil {
		return errors.Wrap(err, "failed to put the wrap state")
	}
	return nil
}

// LoadWrap _ ok means exists
func (wb *WrapStub) LoadWrap(id string) (ok bool, err error) {
	ok = false
	data, err := wb.GetWrapState(id)
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

// GetWrapState
func (wb *WrapStub) GetWrapState(id string) ([]byte, error) {
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
func (wb *WrapStub) Wrap(sender, receiver *Balance, amount, fee Amount, extID, memo string) (*WrapResult, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	rId := sender.GetID()
	wrapID := GetWrapId(receiver.GetID(), wb.stub.GetTxID())

	wrap := NewWrap(wrapID, amount, fee, rId, extID, ts)
	if err = wb.CreateWrap(wrap); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	amount.Neg()
	sender.Amount.Add(&amount)
	sender.Amount.Add(fee.Copy().Neg())

	//sender balance change
	sender.UpdatedTime = ts
	if err = NewBalanceStub(wb.stub).PutBalance(sender); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}
	//sender balance log
	sbl := NewBalanceWrapLog(sender, receiver, amount, &fee, memo, wrapID)
	sbl.CreatedTime = ts
	if err = NewBalanceStub(wb.stub).PutBalanceLog(sbl); err != nil {
		logger.Debug(err.Error())
		return nil, err
	}

	// fee
	if _, err := NewFeeStub(wb.stub).CreateFee(sender.GetID(), fee); err != nil {
		return nil, err
	}

	wrapResult := NewWrapResult(wrap, sbl)

	return wrapResult, nil
}

// UnWrap _
// bCode : bridge token code e.g. WPCI
func (wb *WrapStub) UnWrap(sender, receiver *Balance, amount Amount, extID, extTxID, memo string) (*WrapResult, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	rId := receiver.GetID()
	wrapID := GetWrapId(sender.GetID(), extTxID)

	wrap := NewUnWrap(wrapID, amount, rId, extID, extTxID, ts)
	if err = wb.CreateWrap(wrap); err != nil {
		return nil, err
	}

	//receiver balance change
	receiver.Amount.Add(&amount)
	receiver.UpdatedTime = ts
	if err = NewBalanceStub(wb.stub).PutBalance(receiver); err != nil {
		return nil, err
	}
	//receiver balance log
	rbl := NewBalanceUnWrapLog(sender, receiver, amount, memo, wrapID)
	rbl.CreatedTime = ts
	if err = NewBalanceStub(wb.stub).PutBalanceLog(rbl); err != nil {
		return nil, err
	}

	wrapResult := NewWrapResult(wrap, rbl)

	return wrapResult, nil
}

// WrapPendingBalance wrap the sender's pending balance. (multi-sig contract)
func (wb *WrapStub) WrapPendingBalance(pb *PendingBalance, sender, receiver *Balance, extID string) (*WrapResult, error) {
	ts, err := txtime.GetTime(wb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	rId := receiver.GetID()
	wrapID := GetWrapId(rId, wb.stub.GetTxID())

	wrap := NewWrap(wrapID, pb.Amount, *pb.Fee, rId, extID, ts)
	if err := wb.CreateWrap(wrap); err != nil {
		return nil, err
	}

	pb.Amount.Neg()
	sender.Amount.Add(&pb.Amount)
	sender.UpdatedTime = ts

	// fee
	if pb.Fee != nil {
		if _, err := NewFeeStub(wb.stub).CreateFee(pb.Account, *pb.Fee); err != nil {
			return nil, err
		}
	}

	//sender balance log
	sbl := NewBalanceWrapLog(sender, receiver, pb.Amount, pb.Fee, pb.Memo, wrapID)
	sbl.CreatedTime = ts
	if err = NewBalanceStub(wb.stub).PutBalanceLog(sbl); err != nil {
		return nil, err
	}

	// remove pending balance
	if err = wb.stub.DelState(NewBalanceStub(wb.stub).CreatePendingKey(pb.DOCTYPEID)); err != nil {
		return nil, errors.Wrap(err, "failed to delete the pending balance")
	}

	wrapResult := NewWrapResult(wrap, sbl)

	return wrapResult, nil
}
