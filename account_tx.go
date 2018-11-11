// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
)

// information of the account
// params[0] : token code | account address
func accountGet(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 1 {
		return shim.Error("incorrect number of parameters. expecting 1")
	}

	kid, err := GetKID(stub, false) // verify invoker
	if err != nil {
		return shim.Error(err.Error())
	}

	var ab *AccountStub

	code, err := ValidateTokenCode(params[0])
	if nil == err { // by token code
		ab = NewAccountStub(stub, code)
		data, err := ab.GetQueryMainAccount(kid)
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
	ab = NewAccountStub(stub, addr.Code)
	account, err := ab.GetAccount(addr)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the account")
	}
	return responseAccount(account)
}

// list of account's addresses
// params[0] : "" | token code
// params[1] : bookmark
// ISSUE: list by an account address (privacy problem)
func accountList(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	kid, err := GetKID(stub, false)
	if err != nil {
		return shim.Error(err.Error())
	}

	code := ""
	bookmark := ""
	if len(params) > 0 {
		if len(params[0]) > 0 {
			code, err = ValidateTokenCode(params[0])
			if err != nil {
				return shim.Error(err.Error())
			}
		}
		if len(params) > 1 {
			bookmark = params[1]
		}
	}

	ab := NewAccountStub(stub, code)
	res, err := ab.GetQueryHolderAccounts(kid, bookmark)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get account addresses list")
	}

	data, err := json.Marshal(res)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to marshal account addresses list")
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

	ab := NewAccountStub(stub, code)

	// TODO: check token issued

	if len(params) < 2 { // personal account
		account, err := ab.CreateAccount(kid)
		if err != nil {
			return shim.Error(err.Error())
		}
		return responseAccount(account)
	}

	// joint account

	// check invoker's main account
	_, err = ab.GetQueryMainAccount(kid)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get invoker's personal account")
	}
	holders := stringset.New(kid) // KIDs

	addrs := *stringset.New(params[1:]...) // remove duplication
	// validate co-holders
	for addr := range addrs {
		holder, err := ab.GetSignableID(addr)
		if err != nil {
			return shim.Error(err.Error())
		}
		holders.Add(holder)
	}

	if holders.Size() < 2 {
		return shim.Error("joint account needs co-holders")
	}

	// TODO: contract
	account, err := ab.CreateJointAccount(holders)
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
