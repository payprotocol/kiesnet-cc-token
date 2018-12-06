// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/contract"
	"github.com/key-inside/kiesnet-ccpkg/kid"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
)

// params[0] : sender address (empty string = personal account)
// params[1] : receiver address
// params[2] : amount (big int string)
// params[3] : memo (max 128 charactors)
// params[4] : pending time (time represented by int64 seconds)
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

	// addresses
	rAddr, err := ParseAddress(params[1])
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to parse the receiver's account address")
	}
	var sAddr *Address
	if len(params[0]) > 0 {
		sAddr, err = ParseAddress(params[0])
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to parse the sender's account address")
		}
		if rAddr.Code != sAddr.Code { // not same token
			return shim.Error("different token accounts")
		}
	} else {
		sAddr = NewAddress(rAddr.Code, AccountTypePersonal, kid)
	}

	// IMPORTANT: assert(sender != receiver)
	if sAddr.Equal(rAddr) {
		return shim.Error("can't transfer to self")
	}

	ab := NewAccountStub(stub, rAddr.Code)

	// sender
	sender, err := ab.GetAccount(sAddr)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the sender account")
	}
	if !sender.HasHolder(kid) {
		return shim.Error("invoker is not holder")
	}
	if sender.IsSuspended() {
		return shim.Error("the sender account is suspended")
	}

	// receiver
	receiver, err := ab.GetAccount(rAddr)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the receiver account")
	}
	if receiver.IsSuspended() {
		return shim.Error("the receiver account is suspended")
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
	var pendingTime *time.Time
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
		// pending time
		if len(params) > 4 {
			seconds, err := strconv.ParseInt(params[4], 10, 64)
			if err != nil {
				return shim.Error("invalid pending time: need seconds since 1970")
			}
			ts, err := stub.GetTxTimestamp()
			if err != nil {
				return shim.Error("failed to get the timestamp")
			}
			if ts.GetSeconds() < seconds { // meaning pending time
				ut := time.Unix(seconds, 0)
				pendingTime = &ut
			}
			// expiry
			if len(params) > 5 && len(params[5]) > 0 {
				expiry, err = strconv.ParseInt(params[5], 10, 64)
				if err != nil {
					return shim.Error("invalid expiry: need seconds")
				}
				// extra signers
				if len(params) > 6 {
					addrs := stringset.New(params[6:]...) // remove duplication
					for addr := range addrs.Map() {
						kids, err := ab.GetSignableIDs(addr)
						if err != nil {
							return shim.Error(err.Error())
						}
						signers.AppendSlice(kids)
					}
				}
			}
		}
	}

	var log *BalanceLog // log for response

	if signers.Size() > 1 { // multi-sig
		if signers.Size() > 128 {
			return shim.Error("too many signers")
		}
		// pending balance id
		pbID := stub.GetTxID()
		// contract
		ptStr := "0"
		if pendingTime != nil {
			ptStr = params[4]
		}
		doc := []string{"transfer", pbID, sender.GetID(), receiver.GetID(), amount.String(), memo, ptStr}
		docb, err := json.Marshal(doc)
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to create a contract")
		}
		con, err := contract.CreateContract(stub, docb, expiry, signers)
		if err != nil {
			return shim.Error(err.Error())
		}
		// pending balance
		log, err = bb.Deposit(pbID, sBal, con, *amount, memo)
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to create a pending balance")
		}
	} else { // instant sending
		log, err = bb.Transfer(sBal, rBal, *amount, memo, pendingTime)
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to transfer")
		}
	}

	// log is not nil
	data, err := json.Marshal(log)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}

// contract callbacks

// doc: ["transfer", pending-balance-ID, sender-ID, receiver-ID, amount, memo, pending-time]
func cancelTransfer(stub shim.ChaincodeStubInterface, cid string, doc []interface{}) peer.Response {
	if len(doc) < 2 {
		return shim.Error("invalid contract document")
	}

	// pending balance
	bb := NewBalanceStub(stub)
	pb, err := bb.GetPendingBalance(doc[1].(string))
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the pending balance")
	}
	// validate
	if pb.Type != PendingBalanceTypeContract || pb.RID != cid {
		return shim.Error("invalid pending balance")
	}

	// ISSUE: check account ?

	// withdraw
	if _, err = bb.Withdraw(pb); err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to withdraw")
	}

	return shim.Success(nil)
}

// doc: ["transfer", pending-balance-ID, sender-ID, receiver-ID, amount, memo, pending-time]
func executeTransfer(stub shim.ChaincodeStubInterface, cid string, doc []interface{}) peer.Response {
	if len(doc) < 7 {
		return shim.Error("invalid contract document")
	}

	// pending balance
	bb := NewBalanceStub(stub)
	pb, err := bb.GetPendingBalance(doc[1].(string))
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the pending balance")
	}
	// validate
	if pb.Type != PendingBalanceTypeContract || pb.RID != cid {
		return shim.Error("invalid pending balance")
	}

	// ISSUE: check accounts ? (suspended)

	// receiver balance
	rBal, err := bb.GetBalance(doc[3].(string))
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the receiver's balance")
	}

	// pending time
	ptStr := doc[6].(string)
	var pendingTime *time.Time
	if ptStr != "" && ptStr != "0" {
		seconds, err := strconv.ParseInt(ptStr, 10, 64)
		if err != nil {
			return shim.Error("invalid pending time")
		}
		ut := time.Unix(seconds, 0)
		pendingTime = &ut
	}

	// transfer
	if err = bb.TransferPendingBalance(pb, rBal, pendingTime); err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to transfer a pending balance")
	}

	return shim.Success(nil)
}
