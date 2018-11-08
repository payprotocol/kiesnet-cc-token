// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"os"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

// name of the KID chaincode
var kidCcName = "kiesnet-id"

// arguments bytes array for invoking KID chaincode
var kidCcArgs = map[bool][][]byte{
	false: [][]byte{[]byte("kid")},              // non-secure
	true:  [][]byte{[]byte("kid"), []byte("1")}, // secure
}

func init() {
	if os.Getenv("DEV_CHANNEL_NAME") != "" {
		kidCcName = "kiesnet-cc-id"
	}
}

// GetKID invokes KID chaincode and returns the kiesnet id.
func GetKID(stub shim.ChaincodeStubInterface, secure bool) (string, error) {
	res := stub.InvokeChaincode(kidCcName, kidCcArgs[secure], "")
	if res.GetStatus() == 200 {
		return string(res.GetPayload()), nil
	}
	return "", errors.New(res.GetMessage())
}
