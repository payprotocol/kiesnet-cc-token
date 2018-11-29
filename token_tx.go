// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/kid"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
)

// tokenMeta is a trivial test function.
// params[0] : chainode name of token instance
func tokenMeta(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 1 {
		return shim.Error("incorrect number of parameters. expecting 1")
	}

	// authentication
	_, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	chaincodeName := params[0]
	tokenMeta, err := QueryTokenMeta(stub, chaincodeName)
	if err != nil {
		return shim.Error(err.Error())
	}
	payload, err := json.Marshal(tokenMeta._map)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(payload)
}

// params[0] : token code
// params[1] : amount (big int string)
func tokenBurn(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 2 {
		return shim.Error("incorrect number of parameters. expecting 2")
	}

	if err := AssertInvokedByChaincode(stub); err != nil {
		return shim.Error(err.Error())
	}

	code, err := ValidateTokenCode(params[0])
	if err != nil {
		return shim.Error(err.Error())
	}
	amount, err := NewAmount(params[1])
	if err != nil || amount.Sign() < 0 {
		return shim.Error("amount must be positive integer")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	// token
	tb := NewTokenStub(stub)
	token, err := tb.GetToken(code)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the token")
	}
	if token.Supply.Cmp(amount) < 0 {
		return shim.Error("amount must be less or equal than total supply")
	}

	// genesis account
	addr, _ := ParseAddress(token.GenesisAccount) // err is nil
	ab := NewAccountStub(stub, code)
	account, err := ab.GetAccount(addr)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the genesis account")
	}
	if !account.HasHolder(kid) { // authority
		return shim.Error("no authority")
	}

	// balance
	bb := NewBalanceStub(stub)
	bal, err := bb.GetBalance(account.GetID())
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the genesis account balance")
	}
	if bal.Amount.Sign() == 0 {
		return shim.Error("genesis account balance is 0")
	}

	jac := account.(*JointAccount)
	if jac.Holders.Size() > 1 {
		// contract
		doc := []interface{}{"token/burn", code, amount.String()}
		return invokeContract(stub, doc, jac.Holders)
	}

	// burn
	token, err = tb.Burn(token, bal, *amount)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to burn: " + err.Error())
	}

	data, err := json.Marshal(token)
	if err != nil {
		return shim.Error("failed to marshal the token")
	}
	return shim.Success(data)
}

// params[0] : token code (3~6 alphanum)
// params[1] : decimal (int string, min 0, max 18)
// params[2] : max supply (big int string)
// params[3] : initial supply (big int string)
// params[4:] : co-holders (personal account addresses)
func tokenCreate(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if err := AssertInvokedByChaincode(stub); err != nil {
		return shim.Error(err.Error())
	}

	if len(params) < 4 {
		return shim.Error("incorrect number of parameters. expecting 4+")
	}

	code, err := ValidateTokenCode(params[0])
	if err != nil {
		return shim.Error(err.Error())
	}
	decimal, err := strconv.Atoi(params[1])
	if err != nil || decimal < 0 || decimal > 18 {
		return shim.Error("decimal must be integer between 0 and 18")
	}
	maxSupply, err := NewAmount(params[2])
	if err != nil || maxSupply.Sign() < 0 {
		return shim.Error("max supply must be positive integer")
	}
	supply, err := NewAmount(params[3])
	if err != nil || supply.Sign() < 0 || supply.Cmp(maxSupply) > 0 {
		return shim.Error("initial supply must be positive integer and less(or equal) than max supply")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	// co-holders
	holders := stringset.New(kid)
	if len(params) > 4 {
		ab := NewAccountStub(stub, code)
		addrs := stringset.New(params[4:]...) // remove duplication
		// validate co-holders
		for addr := range addrs.Map() {
			kids, err := ab.GetSignableIDs(addr)
			if err != nil {
				return shim.Error(err.Error())
			}
			holders.AppendSlice(kids)
		}
	}

	if holders.Size() > 1 {
		// contract
		doc := []interface{}{"token/create", code, decimal, maxSupply.String(), supply.String(), holders.Strings()}
		return invokeContract(stub, doc, holders)
	}

	tb := NewTokenStub(stub)
	token, err := tb.CreateToken(code, decimal, *maxSupply, *supply, holders)
	if err != nil {
		return shim.Error("failed to create token")
	}

	data, err := json.Marshal(token)
	if err != nil {
		return shim.Error("failed to marshal the token")
	}
	return shim.Success(data)
}

// params[0] : token code
func tokenGet(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 1 {
		return shim.Error("incorrect number of parameters. expecting 1")
	}

	code, err := ValidateTokenCode(params[0])
	if err != nil {
		return shim.Error(err.Error())
	}

	// authentication
	if _, err = kid.GetID(stub, false); err != nil {
		return shim.Error(err.Error())
	}

	tb := NewTokenStub(stub)
	data, err := tb.GetTokenState(code)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(data)
}

// params[0] : token code
// params[1] : amount (big int string)
func tokenMint(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 2 {
		return shim.Error("incorrect number of parameters. expecting 2")
	}

	if err := AssertInvokedByChaincode(stub); err != nil {
		return shim.Error(err.Error())
	}

	code, err := ValidateTokenCode(params[0])
	if err != nil {
		return shim.Error(err.Error())
	}
	amount, err := NewAmount(params[1])
	if err != nil || amount.Sign() < 0 {
		return shim.Error("amount must be positive integer")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	// token
	tb := NewTokenStub(stub)
	token, err := tb.GetToken(code)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the token")
	}
	if token.Supply.Cmp(&token.MaxSupply) >= 0 {
		return shim.Error("max supplied")
	}

	// genesis account
	addr, _ := ParseAddress(token.GenesisAccount) // err is nil
	ab := NewAccountStub(stub, code)
	account, err := ab.GetAccount(addr)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the genesis account")
	}
	if !account.HasHolder(kid) { // authority
		return shim.Error("no authority")
	}
	jac := account.(*JointAccount)
	if jac.Holders.Size() > 1 {
		// contract
		doc := []interface{}{"token/mint", code, amount.String()}
		return invokeContract(stub, doc, jac.Holders)
	}

	// balance
	bb := NewBalanceStub(stub)
	bal, err := bb.GetBalance(account.GetID())
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the genesis account balance")
	}

	// mint
	token, err = tb.Mint(token, bal, *amount)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to mint: " + err.Error())
	}

	data, err := json.Marshal(token)
	if err != nil {
		return shim.Error("failed to marshal the token")
	}
	return shim.Success(data)
}

// contract callbacks

// doc: ["token/burn", code, amount]
func executeTokenBurn(stub shim.ChaincodeStubInterface, cid string, doc []interface{}) peer.Response {
	if len(doc) < 3 {
		return shim.Error("invalid contract document")
	}

	code := doc[1].(string)
	amount, err := NewAmount(doc[2].(string))
	if err != nil {
		return shim.Error("invalid amount")
	}

	// token
	tb := NewTokenStub(stub)
	token, err := tb.GetToken(code)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the token")
	}

	// balance
	bb := NewBalanceStub(stub)
	bal, err := bb.GetBalance(token.GenesisAccount)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the genesis account balance")
	}

	if _, err = tb.Burn(token, bal, *amount); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// doc: ["token/create", code, decimal(int), max-supply, supply, [co-holders...]]
func executeTokenCreate(stub shim.ChaincodeStubInterface, cid string, doc []interface{}) peer.Response {
	if len(doc) < 6 {
		return shim.Error("invalid contract document")
	}

	code := doc[1].(string)
	decimal := int(doc[2].(float64))
	maxSupply, err := NewAmount(doc[3].(string))
	if err != nil {
		return shim.Error("invalid max supply")
	}
	supply, err := NewAmount(doc[4].(string))
	if err != nil {
		return shim.Error("invalid initial supply")
	}
	kids := doc[5].([]interface{})
	holders := stringset.New()
	for _, kid := range kids {
		holders.Add(kid.(string))
	}

	tb := NewTokenStub(stub)
	if _, err = tb.CreateToken(code, decimal, *maxSupply, *supply, holders); err != nil {
		return shim.Error("failed to create token")
	}

	return shim.Success(nil)
}

// doc: ["token/mint", code, amount]
func executeTokenMint(stub shim.ChaincodeStubInterface, cid string, doc []interface{}) peer.Response {
	if len(doc) < 3 {
		return shim.Error("invalid contract document")
	}

	code := doc[1].(string)
	amount, err := NewAmount(doc[2].(string))
	if err != nil {
		return shim.Error("invalid amount")
	}

	// token
	tb := NewTokenStub(stub)
	token, err := tb.GetToken(code)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the token")
	}

	// balance
	bb := NewBalanceStub(stub)
	bal, err := bb.GetBalance(token.GenesisAccount)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the genesis account balance")
	}

	if _, err = tb.Mint(token, bal, *amount); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}
