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

// params[0] : token code (3~6 alphanum)
// params[1] : decimal (int string)
// params[2] : max supply (big int string)
// params[3] : initial supply (big int string)
// params[4:] : co-holders (personal account addresses)
func tokenCreate(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 5 {
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
		for addr := range addrs {
			holder, err := ab.GetSignableID(addr)
			if err != nil {
				return shim.Error(err.Error())
			}
			holders.Add(holder)
		}
	}

	// if holders.Size() > 1 {
	// 	// TODO: contract
	//		return ...
	// }

	tb := NewTokenStub(stub)
	token, err := tb.CreateToken(code, decimal, maxSupply, supply, holders)
	if err != nil {
		return shim.Error("failed to create token")
	}

	data, err := json.Marshal(token)
	if err != nil {
		return shim.Error("failed to marshal the token")
	}
	return shim.Success(data)
}
