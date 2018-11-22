// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"bytes"
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/kid"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
)

// params[0] : token code
// params[1:] : co-holders' personal account addresses (exclude invoker, max 127)
func accountCreate(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 1 {
		return shim.Error("incorrect number of parameters. expecting 1+")
	}

	code, err := ValidateTokenCode(params[0])
	if err != nil {
		return shim.Error(err.Error())
	}

	if err = AssertInvokedByChaincode(stub); err != nil {
		if _, ok := err.(InvalidAccessError); !ok {
			return shim.Error(err.Error())
		}
		// check token issued
		tb := NewTokenStub(stub)
		_, err = tb.GetTokenState(code)
		if err != nil {
			return shim.Error(err.Error())
		}
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	ab := NewAccountStub(stub, code)

	if len(params) < 2 { // personal account
		account, balance, err := ab.CreateAccount(kid)
		if err != nil {
			return shim.Error(err.Error())
		}
		return responseAccountWithBalance(account, balance)
	}

	// joint account

	// check invoker's main(personal) account
	if _, err = ab.GetPersonalAccount(kid); err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get invoker's personal account")
	}

	holders := stringset.New(kid) // KIDs

	addrs := stringset.New(params[1:]...) // remove duplication
	if addrs.Size() > 128 {
		return shim.Error("too many holders")
	}
	// validate & get kid of co-holders
	for addr := range addrs {
		holder, err := ab.GetSignableID(addr)
		if err != nil {
			return shim.Error(err.Error())
		}
		holders.Add(holder)
	}

	if holders.Size() < 2 { // addrs had invoker's addr
		return shim.Error("joint account needs co-holders")
	}

	// TODO: contract

	account, balance, err := ab.CreateJointAccount(holders)
	if err != nil {
		return shim.Error(err.Error())
	}
	return responseAccountWithBalance(account, balance)
}

// information of the account
// params[0] : token code | account address
func accountGet(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 1 {
		return shim.Error("incorrect number of parameters. expecting 1")
	}

	// authentication
	kid, err := kid.GetID(stub, false)
	if err != nil {
		return shim.Error(err.Error())
	}

	var account AccountInterface

	code, err := ValidateTokenCode(params[0])
	if nil == err { // by token code
		ab := NewAccountStub(stub, code)
		account, err = ab.GetPersonalAccount(kid)
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to get the account")
		}
	} else { // by address
		addr, err := ParseAddress(params[0])
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to parse the account address")
		}
		ab := NewAccountStub(stub, addr.Code)
		account, err = ab.GetAccount(addr)
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to get the account")
		}
	}

	// balance state
	bb := NewBalanceStub(stub)
	balance, err := bb.GetBalanceState(account.GetID())
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the balance")
	}

	return responseAccountWithBalanceState(account, balance)
}

// update holders
// params[0] : token code | account address
// params[1:] : co-holders' personal account addresses (exclude invoker, max 127)
func accountHolders(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	return shim.Error("not yet")
}

// list of account's addresses
// params[0] : "" | token code
// params[1] : bookmark
// ISSUE: list by an account address (privacy problem)
func accountList(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	// authentication
	kid, err := kid.GetID(stub, false)
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
func accountLogs(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 1 {
		return shim.Error("incorrect number of parameters. expecting 1")
	}

	addr, err := ParseAddress(params[0])
	if err != nil {
		return shim.Error("failed to parse the account address")
	}
	_ = addr

	// authentication
	_, err = kid.GetID(stub, false)
	if err != nil {
		return shim.Error(err.Error())
	}

	// TODO

	return shim.Success([]byte("account/logs"))
}

// helpers

func responseAccountWithBalance(account AccountInterface, balance *Balance) peer.Response {
	var data []byte
	var err error
	if a, ok := account.(*JointAccount); ok {
		data, err = json.Marshal(&struct {
			*JointAccount
			Balance *Balance `json:"balance"`
		}{a, balance})
	} else if a, ok := account.(*Account); ok {
		data, err = json.Marshal(&struct {
			*Account
			Balance *Balance `json:"balance"`
		}{a, balance})
	} else { // never here
		return shim.Error("unknown account type")
	}
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to marshal the payload")
	}
	return shim.Success(data)
}

func responseAccountWithBalanceState(account AccountInterface, balance []byte) peer.Response {
	data, err := json.Marshal(account)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to marshal the account")
	}
	buf := bytes.NewBuffer(data[:(len(data) - 1)]) // eliminate last '}'
	if _, err := buf.WriteString(`,"balance":`); nil == err {
		if _, err := buf.Write(balance); nil == err {
			if err := buf.WriteByte('}'); nil == err {
				return shim.Success(buf.Bytes())
			}
		}
	}
	return shim.Error("failed to marshal the payload")
}
