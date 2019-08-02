package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
)

func rollbackAllFeePrune20190805(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	// Halt if flag is set. If not, set it.
	flagkey := "rollbackAllFeePrune20190805"
	flag, err := stub.GetState(flagkey)
	if err != nil {
		return shim.Error(err.Error())
	}
	if flag != nil {
		return shim.Error("This function must be executed one and only one time.")
	}
	flag = []byte{1}
	err = stub.PutState(flagkey, flag)
	if err != nil {
		return shim.Error(err.Error())
	}

	ts, err := txtime.GetTime(stub)
	if err != nil {
		return shim.Error(errors.Wrap(err, "failed to get the timestamp").Error())
	}

	tokenStub := NewTokenStub(stub)
	balanceStub := NewBalanceStub(stub)

	// Delete token.LastPrunedFeeID
	token, err := tokenStub.GetToken("PCI")
	if err != nil {
		return shim.Error(err.Error())
	}
	token.LastPrunedFeeID = ""
	token.UpdatedTime = ts
	err = tokenStub.PutToken(token)
	if err != nil {
		return shim.Error(err.Error())
	}

	feeHolderAddressStr := "PCI01639F0B0A494B6040CE8B5B0DC4C56ACA7E78F6BAB02271AF"
	balance, err := balanceStub.GetBalance(feeHolderAddressStr)
	if err != nil {
		return shim.Error(err.Error())
	}

	// Set balance 0
	burningAmount := *(balance.Amount.Copy()) // real diff
	burningAmount.Neg()
	balance.Amount.Add(&burningAmount)
	balance.UpdatedTime = ts
	err = balanceStub.PutBalance(balance)
	if err != nil {
		return shim.Error(err.Error())
	}

	// Save BalanceLog of type burn
	burnLog := NewBalanceSupplyLog(balance, burningAmount)
	burnLog.Memo = "rollback all fee/prune at 20190805"
	burnLog.CreatedTime = ts
	err = balanceStub.PutBalanceLog(burnLog)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}
