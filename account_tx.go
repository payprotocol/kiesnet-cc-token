// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
)

// params[0] : token code | account address
func accountGet(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 1 {
		return shim.Error("incorrect number of parameters. expecting 1")
	}

	kid, err := GetKID(stub, false) // verify invoker
	if err != nil {
		return shim.Error(err.Error())
	}

	ab := NewAccountStub(stub)

	code, err := ValidateTokenCode(params[0])
	if nil == err { // by token code
		data, err := ab.GetQueryMainAccount(code, kid)
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to get the account")
		}
		// data is not nil
		return shim.Success(data)
	}

	// by address
	addr, err := ParseAddress(params[0])
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to parse the account address")
	}
	account, err := ab.GetAccount(addr)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the account")
	}
	return responseAccount(account)
}

// params[0] : "" | token code
// params[1] : bookmark
// ISSUE: list by an account address (privacy problem)
func accountList(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	kid, err := GetKID(stub, false)
	if err != nil {
		return shim.Error(err.Error())
	}

	ab := NewAccountStub(stub)

	code := ""
	bookmark := ""
	if len(params) > 0 {
		code, err = ValidateTokenCode(params[0])
		if err != nil {
			return shim.Error(err.Error())
		}
		if len(params) > 1 {
			bookmark = params[1]
		}
	}

	res, err := ab.GetQueryAccountsResult(code, kid, bookmark)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get account list")
	}

	data, err := json.Marshal(res)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to marshal account list")
	}
	return shim.Success(data)
}

// params[0] : account address
func accountLog(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 1 {
		return shim.Error("incorrect number of parameters. expecting 1")
	}

	addr, err := ParseAddress(params[0])
	if err != nil {
		return shim.Error("failed to parse the account address")
	}
	_ = addr

	_, err = GetKID(stub, false)
	if err != nil {
		return shim.Error(err.Error())
	}

	// TODO

	return shim.Success([]byte("account/log"))
}

// params[0] : token code
// params[1:] : co-holders' personal account addresses
func accountNew(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 1 {
		return shim.Error("incorrect number of parameters. expecting 1 or more")
	}

	code, err := ValidateTokenCode(params[0])
	if err != nil {
		return shim.Error(err.Error())
	}

	kid, err := GetKID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	ab := NewAccountStub(stub)

	if len(params) < 2 { // personal account
		account, err := ab.CreateAccount(code, kid)
		if err != nil {
			return shim.Error(err.Error())
		}
		return responseAccount(account)
	}

	// joint account
	addrs := stringset.New(params[1:]...).Strings() // remove duplication
	holders := stringset.New(kid)                   // KIDs
	// validate co-holders
	for _, addr := range addrs {
		holder, err := ab.GetSignableID(addr)
		if err != nil {
			return shim.Error(err.Error())
		}
		holders.Add(holder)
	}

	if holders.Size() < 2 {
		return shim.Error("joint account needs more then 0 co-holders")
	}

	// TODO: contract
	account, err := ab.CreateJointAccount(code, *holders)
	if err != nil {
		return shim.Error(err.Error())
	}
	return responseAccount(account)
}

func responseAccount(account AccountInterface) peer.Response {
	data, err := json.Marshal(account)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to marshal the account")
	}
	return shim.Success(data)
}
