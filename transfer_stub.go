// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/kid"
)

func getAccounts(stub shim.ChaincodeStubInterface, params []string) (AccountInterface, AccountInterface, error) {
	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return nil, nil, err
	}

	// addresses
	rAddr, err := ParseAddress(params[1])
	if err != nil {
		logger.Debug(err.Error())
		return nil, nil, err
	}
	var sAddr *Address
	if len(params[0]) > 0 {
		sAddr, err = ParseAddress(params[0])
		if err != nil {
			logger.Debug(err.Error())
			return nil, nil, err
		}
		if rAddr.Code != sAddr.Code { // not same token
			return nil, nil, err
		}
	} else {
		sAddr = NewAddress(rAddr.Code, AccountTypePersonal, kid)
	}

	// IMPORTANT: assert(sender != receiver)
	if sAddr.Equal(rAddr) {
		return nil, nil, err
	}

	ab := NewAccountStub(stub, rAddr.Code)

	// sender
	sender, err := ab.GetAccount(sAddr)
	if err != nil {
		logger.Debug(err.Error())
		return nil, nil, err
	}
	if !sender.HasHolder(kid) {
		return nil, nil, err
	}
	if sender.IsSuspended() {
		return nil, nil, err
	}

	// receiver
	receiver, err := ab.GetAccount(rAddr)
	if err != nil {
		logger.Debug(err.Error())
		return nil, nil, err
	}
	if receiver.IsSuspended() {
		return nil, nil, err
	}

	return sender, receiver, nil

}
