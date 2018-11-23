// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

var logger = shim.NewLogger("kiesnet-token")

// Chaincode _
type Chaincode struct {
}

// Init implements shim.Chaincode interface.
func (cc *Chaincode) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

// Invoke implements shim.Chaincode interface.
func (cc *Chaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	fn, params := stub.GetFunctionAndParameters()
	if txFn := routes[fn]; txFn != nil {
		return txFn(stub, params)
	}
	return shim.Error("unknown function: '" + fn + "'")
}

// TxFunc _
type TxFunc func(shim.ChaincodeStubInterface, []string) peer.Response

// routes is the map of invoke functions
var routes = map[string]TxFunc{
	"account/create":           accountCreate,
	"account/get":              accountGet,
	"account/list":             accountList,
	"account/logs":             accountLogs, // TODO
	"account/suspend":          accountSuspend,
	"account/unsuspend":        accountUnsuspend,
	"account/update/holders":   accountUpdateHolders, // TODO
	"balance/logs":             balanceLogs,
	"balance/pending/list":     balancePendingList,
	"balance/pending/withdraw": balancePendingWithdraw,
	"token/burn":               tokenBurn,
	"token/create":             tokenCreate,
	"token/get":                tokenGet,
	"token/mint":               tokenMint,
	"transfer":                 transfer,
	"ver":                      ver,
}

func ver(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	return shim.Success([]byte("Kiesnet Token v1.0 created by Key Inside Co., Ltd."))
}

// AssertInvokedByChaincode _
func AssertInvokedByChaincode(stub shim.ChaincodeStubInterface) error {
	// TODO: un-comment
	// ccid, err := ccid.GetID(stub)
	// if err != nil {
	// 	return err
	// }
	// if "kiesnet-token" == ccid || "kiesnet-cc-token" == ccid {
	// 	return InvalidAccessError{}
	// }
	return nil
}

func main() {
	if err := shim.Start(new(Chaincode)); err != nil {
		logger.Criticalf("failed to start chaincode: %s", err)
	}
}
