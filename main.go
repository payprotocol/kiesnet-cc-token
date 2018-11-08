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

func ver(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	return shim.Success([]byte("kiesnet token v1.0"))
}

// TxFunc _
type TxFunc func(shim.ChaincodeStubInterface, []string) peer.Response

// routes is the map of invoke functions
var routes = map[string]TxFunc{
	"account/get":  accountGet,
	"account/list": accountList,
	"account/log":  accountLog,
	"account/new":  accountNew,
	// "account/update":    accountUpdated,
	// "account/suspend":   accountSuspend,
	// "account/unsuspend": accountUnsuspend,
	"token/get":    tokenGet,
	"token/create": tokenCreate,
	// "token/mint":   tokenMint,
	// "token/burn":   tokenBurn,
	// transfer
	// transfer/log  records
	// withdraw  withdrawal
	// withdraw/list
	// balance/transfer
	// balance/log
	// balance/withdraw
	// balance/withdraw/list
	"ver": ver,
}

func main() {
	if err := shim.Start(new(Chaincode)); err != nil {
		logger.Criticalf("failed to start chaincode: %s", err)
	}
}
