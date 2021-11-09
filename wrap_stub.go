package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
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
	logger.Debug("createWrap")
	if err != nil {
		logger.Debug("createWrap-1")
		return errors.Wrap(err, "failed to marshal the wrap")
	}
	ok, err := wb.LoadWrap(wrap.DOCTYPEID)
	if err != nil {
		logger.Debug("createWrap-2")
		return errors.Wrap(err, "failed to retrieve the wrap")
	}
	if ok {
		logger.Debug("createWrap-3")
		return errors.New("wrap aleady exists")
	}
	logger.Debug("createWrap-4")
	if err = wb.stub.PutState(wrap.DOCTYPEID, data); err != nil {
		logger.Debug("createWrap-5")
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
