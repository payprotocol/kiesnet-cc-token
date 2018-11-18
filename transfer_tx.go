// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/kid"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
)

// params[0] : sender address (empty string = personal account)
// params[1] : receiver address
// params[2] : amount (big int string)
// params[3] : memo (max 128 charactors)
// params[4] : lock-until (time represented by int64 seconds)
// params[5] : expiry (duration represented by int64 seconds, multi-sig only)
// params[6:] : extra signers (personal account addresses)
func transfer(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 3 {
		return shim.Error("incorrect number of parameters. expecting 3+")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	// amount
	amount, err := NewAmount(params[2])
	if err != nil {
		return shim.Error(err.Error())
	}
	if amount.Sign() <= 0 {
		return shim.Error("invalid amount. amount must be larger than 0")
	}

	// receiver
	rAddr, err := ParseAddress(params[1])
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to parse the receiver's account address")
	}
	ab := NewAccountStub(stub, rAddr.Code)
	receiver, err := ab.GetAccount(rAddr)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the receiver account")
	}
	if receiver.IsSuspended() {
		return shim.Error("the receiver account is suspended")
	}

	// sender
	var sAddr *Address
	var sender AccountInterface
	if len(params[0]) < 1 { // from personal account
		sender, err = ab.GetPersonalAccount(kid)
	} else {
		sAddr, err = ParseAddress(params[0])
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to parse the sender's account address")
		}
		if sAddr.Code != rAddr.Code {
			return shim.Error("different token accounts")
		}
		sender, err = ab.GetAccount(sAddr)
	}
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the sender account")
	}
	if sender.IsSuspended() {
		return shim.Error("the sender account is suspended")
	}

	// IMPORTANT: assert(sender != receiver)
	if receiver.GetAddress() == sender.GetAddress() {
		return shim.Error("can't transfer to self")
	}

	// sender balance
	bb := NewBalanceStub(stub)
	sBal, err := bb.GetBalance(sender.GetID())
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the sender's balance")
	}
	if sBal.Amount.Cmp(amount) < 0 {
		return shim.Error("not enough balance")
	}

	// receiver balance
	rBal, err := bb.GetBalance(receiver.GetID())
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the receiver's balance")
	}

	// options
	memo := ""
	var lockUntil time.Time
	var expiry int64
	signers := stringset.New(kid)
	if a, ok := sender.(*JointAccount); ok {
		signers.AppendSet(a.Holders)
	}
	// memo
	if len(params) > 3 {
		if len(params[3]) > 128 { // 128 charactors limit
			memo = params[3][:128]
		} else {
			memo = params[3]
		}
		// lock-until
		if len(params) > 4 {
			seconds, err := strconv.ParseInt(params[4], 10, 64)
			if err != nil {
				return shim.Error("invalid timelock: need seconds since 1970")
			}
			lockUntil = time.Unix(seconds, 0)
			// expiry
			if len(params) > 5 && len(params[5]) > 0 {
				expiry, err = strconv.ParseInt(params[5], 10, 64)
				if err != nil {
					return shim.Error("invalid expiry: need seconds")
				}
				// extra signers
				if len(params) > 6 {
					addrs := stringset.New(params[6:]...) // remove duplication
					for addr := range addrs {
						signer, err := ab.GetSignableID(addr)
						if err != nil {
							return shim.Error(err.Error())
						}
						signers.Add(signer)
					}
				}
			}
		}
	}
	_ = expiry

	if signers.Size() > 1 {
		if signers.Size() > 128 {
			return shim.Error("too many signers")
		}
		// TODO: contract
	}

	tb := NewTransferStub(stub)
	tb.Transfer(sBal, rBal, amount, memo, &lockUntil)

	return shim.Success([]byte("transfer"))
}

func transferLog(stub shim.ChaincodeStubInterface, params []string) peer.Response {

	return shim.Success([]byte("transfer/log"))
}
