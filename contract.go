// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
	"github.com/pkg/errors"
)

// Contract _
type Contract struct {
	_map map[string]interface{}
}

// GetID implements Identifiable
func (c *Contract) GetID() string {
	return c._map["@contract"].(string)
}

// GetExpiryTime _
func (c *Contract) GetExpiryTime() (*time.Time, error) {
	t, err := time.Parse(time.RFC3339, c._map["expiry_time"].(string))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// MarshalJSON _
func (c *Contract) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString(`{"@contract":"`)
	if _, err := buf.WriteString(c.GetID()); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(`"}`); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ContractCfg is configuration for 'contract' chaincode
var ContractCfg struct {
	CC string // chaincode name
}

func init() {
	if os.Getenv("DEV_CHANNEL_NAME") != "" { // dev mode
		ContractCfg.CC = "kiesnet-cc-contract"
	} else {
		ContractCfg.CC = "kiesnet-contract"
	}
}

// InvokeContractChaincode invokes contract chaincode returns contract ID
func InvokeContractChaincode(stub shim.ChaincodeStubInterface, doc []byte, expiry int64, signers *stringset.Set) (*Contract, error) {
	expb := []byte(strconv.FormatInt(expiry, 10))
	args := [][]byte{[]byte("create"), doc, expb}
	for signer := range signers.Map() {
		args = append(args, []byte(signer))
	}
	// invoke
	res := stub.InvokeChaincode(ContractCfg.CC, args, "")
	if res.GetStatus() == 200 {
		m := make(map[string]interface{})
		err := json.Unmarshal(res.GetPayload(), &m)
		if err != nil {
			return nil, err
		}
		contract := &Contract{_map: m}
		// ISSUE: contract-document fingerprint
		return contract, nil
	}
	return nil, errors.New(res.GetMessage())
}
