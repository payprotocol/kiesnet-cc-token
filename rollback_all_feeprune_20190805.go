package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
)

// This func rolls back the state caused by the malfunction of fee prune logic.
// It must be executed one and only one time.
func rollbackAllFeePrune20190805(stub shim.ChaincodeStubInterface) error {
	logger.Info("rollbackAllFeePrune20190805")
	ts, err := txtime.GetTime(stub)
	if err != nil {
		logger.Warningf("%s\n", errors.Wrap(err, "failed to get the timestamp").Error())
		return nil
	}

	tokenStub := NewTokenStub(stub)
	balanceStub := NewBalanceStub(stub)

	// Delete token.LastPrunedFeeID
	token, err := tokenStub.GetToken("PCI")
	if err != nil {
		logger.Warningf("%s\n", err.Error())
		return nil
	}
	token.LastPrunedFeeID = ""
	token.UpdatedTime = ts

	// If we don't have the account which address is as below, we're not on the
	// payprotocol mainnet. exit with warning.
	feeHolderAddressStr := "PCI01639F0B0A494B6040CE8B5B0DC4C56ACA7E78F6BAB02271AF"
	balance, err := balanceStub.GetBalance(feeHolderAddressStr)
	if err != nil {
		logger.Warningf("%s\n", err.Error())
		return nil
	}
	// The function must be executed one and only one time. Previous chaincode
	// was wrong so it has given not-minted 205933297950 to the fee holder. If
	// the fee holder doesn't have 205933297950 tokens, rollback is already
	// done.
	rollbackAmount, err := NewAmount("205933297950")
	if err != nil {
		logger.Warningf("%s\n", err.Error())
		return nil
	}
	if balance.Amount.Cmp(rollbackAmount) != 0 {
		logger.Warningf("not the rollback situation. exit\n")
		return nil
	}

	// Set balance 0
	burningAmount := *(balance.Amount.Copy()) // real diff
	burningAmount.Neg()
	balance.Amount.Add(&burningAmount)
	balance.UpdatedTime = ts

	// Save BalanceLog of type burn
	burnLog := NewBalanceSupplyLog(balance, burningAmount)
	burnLog.Memo = "rollback all fee/prune at 20190805"
	burnLog.CreatedTime = ts

	err = tokenStub.PutToken(token)
	if err != nil {
		logger.Errorf("%s\n", err.Error())
		return err
	}
	err = balanceStub.PutBalance(balance)
	if err != nil {
		logger.Errorf("%s\n", err.Error())
		return err
	}
	err = balanceStub.PutBalanceLog(burnLog)
	if err != nil {
		logger.Errorf("%s\n", err.Error())
		return err
	}

	logger.Info("rollbackAllFeePrune20190805 end")
	return nil
}
