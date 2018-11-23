// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/kid"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

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
	ts, err := txtime.GetTime(stub)
	if err != nil {
		return shim.Error(err.Error())
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
	if token.Supply.Sign() == 0 {
		return shim.Error("no supply")
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
	if bal.Amount.Cmp(amount) < 0 {
		amount = bal.Amount.Copy() // real diff
	}
	amount.Neg()                     // -
	_, err = bb.Supply(bal, *amount) // burn
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to burn")
	}

	// supply
	token.Supply.Add(amount)
	token.UpdatedTime = ts
	if err = tb.PutToken(token); err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to burn")
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
			kids, err := ab.GetHolders(addr)
			if err != nil {
				return shim.Error(err.Error())
			}
			holders.AppendSlice(kids)
		}
	}

	if holders.Size() > 1 {
		// TODO: contract
		//return ...
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
		return shim.Error("failed to marshal the token")
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
	ts, err := txtime.GetTime(stub)
	if err != nil {
		return shim.Error(err.Error())
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

	// supply
	token.Supply.Add(amount)
	if token.MaxSupply.Cmp(&token.Supply) < 0 {
		amount.Add(&token.MaxSupply)
		amount.Add(token.Supply.Neg()) // real diff
		token.Supply = token.MaxSupply
	}
	token.UpdatedTime = ts
	if err = tb.PutToken(token); err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to mint")
	}

	// balance
	bb := NewBalanceStub(stub)
	bal, err := bb.GetBalance(account.GetID())
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the genesis account balance")
	}
	_, err = bb.Supply(bal, *amount) // mint
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to mint")
	}

	data, err := json.Marshal(token)
	if err != nil {
		return shim.Error("failed to marshal the token")
	}
	return shim.Success(data)
}
