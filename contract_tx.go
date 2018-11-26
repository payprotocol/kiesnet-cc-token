// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/kid"
)

// CtrFunc _
type CtrFunc func(stub shim.ChaincodeStubInterface, cid string, doc []string) peer.Response

// routes is the map of contract functions
var ctrRoutes = map[string][]CtrFunc{
	"transfer": []CtrFunc{cancelTransfer, executeTransfer},
}

func contractCallback(stub shim.ChaincodeStubInterface, fnIdx int, params []string) peer.Response {
	if len(params) != 2 {
		return shim.Error("incorrect number of parameters. expecting 2")
	}

	// ISSUE: validate ccid ('kiesnet-contract', 'kiesnet-cc-contract') ?
	if err := AssertInvokedByChaincode(stub); err != nil {
		return shim.Error(err.Error())
	}

	// authentication
	_, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	cid := params[0] // contract ID
	doc := []string{}
	err = json.Unmarshal([]byte(params[1]), &doc)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to unmarshal contract document")
	}

	if ctrFn := ctrRoutes[doc[0]][fnIdx]; ctrFn != nil {
		return ctrFn(stub, cid, doc)
	}
	return shim.Error("unknown contract: '" + doc[0] + "'")
}

func contractCancel(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	return contractCallback(stub, 0, params)
}

func contractExecute(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	return contractCallback(stub, 1, params)
}
