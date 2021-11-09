package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/contract"
	"github.com/key-inside/kiesnet-ccpkg/kid"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
)

// params[0] : sender address (not empty)
// params[1] : external token code(wpci, ...)
// params[2] : external adress(wpci, ...)
// params[3] : amount (big int string) must bigger than 0
// params[4] : memo (see MemoMaxLength)
// params[5] : expiry (duration represented by int64 seconds, multi-sig only)
// params[6:] : extra signers (personal account addresses)
func wrap(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	//1. param check
	if len(params) < 4 {
		return shim.Error("incorrect number of parameters. expecting 3+")
	}

	//authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	//token code check code must be WPCI
	extCode, err := ValidateExtTokenCode(params[1])
	if err != nil {
		return shim.Error(err.Error())
	}

	//external address, need to check validate?
	extID := params[2]

	//amount check
	amount, err := NewAmount(params[3])
	if err != nil {
		return shim.Error(err.Error())
	}
	if amount.Sign() <= 0 {
		return shim.Error("invalid amount. must be greater than 0")
	}

	//sender address check
	sAddr, err := ParseAddress(params[0])
	if err != nil {
		return shim.Error("failed to parse the sender's account address")
	}
	code, err := ValidateTokenCode(sAddr.Code)
	if nil != err {
		return shim.Error(err.Error())
	}

	// receiver address get
	tb := NewTokenStub(stub)
	token, err := tb.GetToken(code)
	if nil != err {
		return responseError(err, "failed to get the token")
	}
	//if add another brigde token, change this code
	rAddr, _ := ParseAddress(token.WrapInfo.WPCIAddress)

	//sender = receiver
	if sAddr.Equal(rAddr) {
		return shim.Error("can't wrap to self")
	}

	// account check
	ab := NewAccountStub(stub, code)
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
		return responseError(err, "failed to get the target account")
	}
	// can be suspended?
	if receiver.IsSuspended() {
		return shim.Error("the receiver account is suspended")
	}

	// balance check
	bb := NewBalanceStub(stub)
	sBal, err := bb.GetBalance(sender.GetID())
	if err != nil {
		return shim.Error("failed to get the sender's balance")
	}
	rBal, err := bb.GetBalance(receiver.GetID())
	if err != nil {
		return shim.Error("failed to get the receiver's balance")
	}

	// asume there's no fee in here
	if sBal.Amount.Cmp(amount) < 0 {
		return shim.Error("not enough balance")
	}

	memo := ""
	var expiry int64
	signers := stringset.New(kid)
	if len(params) > 4 {
		// memo
		if len(params[4]) > MemoMaxLength { // length limit
			memo = params[4][:MemoMaxLength]
		} else {
			memo = params[4]
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

	var log *BalanceLog // log for response
	if signers.Size() > 1 {
		//TODO multisig
		fmt.Printf(fmt.Sprintf("%s", expiry))
		if signers.Size() > 128 {
			return shim.Error("too many signers")
		}
		// pending balance id
		pbID := stub.GetTxID()
		doc := []string{"wrap", pbID, sender.GetID(), receiver.GetID(), amount.String(), extCode, extID, memo}
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
		log, err = bb.Deposit(pbID, sBal, con, *amount, nil, memo)
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to create the pending balance")
		}
	} else {
		log, err = bb.Wrap(sBal, rBal, *amount, extCode, extID, memo)
		if err != nil {
			return shim.Error("failed to wrap")
		}
	}

	// log is not nil
	data, err := json.Marshal(log)
	if err != nil {
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)

}

// params[0] : sender address (not empty)
// params[1] : external token code(wpci, ...)
// params[2] : external adress(wpci, ...)
// params[3] : external token txid
// params[4] : amount (big int string) must bigger than 0
// params[5] : memo (see MemoMaxLength)
func unwrap(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	//1. param check
	if len(params) < 4 {
		return shim.Error("incorrect number of parameters. expecting 3+")
	}

	//authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	//token code check code must be WPCI
	extCode, err := ValidateExtTokenCode(params[1])
	if err != nil {
		return shim.Error(err.Error())
	}

	//external address, txId, need to check validate?
	extID := params[2]
	extTxID := params[3]

	//amount check
	amount, err := NewAmount(params[4])
	if err != nil {
		return shim.Error(err.Error())
	}
	if amount.Sign() <= 0 {
		return shim.Error("invalid amount. must be greater than 0")
	}

	// reciever address check
	rAddr, err := ParseAddress(params[0])
	if err != nil {
		return shim.Error("failed to parse the sender's account address")
	}
	code, err := ValidateTokenCode(rAddr.Code)
	if nil != err {
		return shim.Error(err.Error())
	}

	// receiver address get
	tb := NewTokenStub(stub)
	token, err := tb.GetToken(code)
	if nil != err {
		return responseError(err, "failed to get the token")
	}
	//if add another brigde token, change this code
	sAddr, _ := ParseAddress(token.WrapInfo.WPCIAddress)

	//sender = receiver
	if sAddr.Equal(rAddr) {
		return shim.Error("can't unwrap to self")
	}

	// account check
	ab := NewAccountStub(stub, code)
	// sender(wrap account)
	sender, err := ab.GetAccount(sAddr)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the sender account")
	}
	if !sender.HasHolder(kid) {
		return shim.Error("invoker is not holder")
	}
	if sender.IsSuspended() { //need suspend check?
		return shim.Error("the sender account is suspended")
	}

	// receiver
	receiver, err := ab.GetAccount(rAddr)
	if err != nil {
		return responseError(err, "failed to get the target account")
	}
	if receiver.IsSuspended() {
		return shim.Error("the receiver account is suspended")
	}

	// balance no need to balance check
	bb := NewBalanceStub(stub)
	sBal, err := bb.GetBalance(sender.GetID())
	if err != nil {
		return shim.Error("failed to get the sender's balance")
	}
	rBal, err := bb.GetBalance(receiver.GetID())
	if err != nil {
		return shim.Error("failed to get the receiver's balance")
	}

	memo := ""
	if len(params) > 5 {
		// memo
		if len(params[5]) > MemoMaxLength { // length limit
			memo = params[5][:MemoMaxLength]
		} else {
			memo = params[5]
		}
	}

	log, err := bb.UnWrap(sBal, rBal, *amount, extCode, extID, extTxID, memo)
	if err != nil {
		return shim.Error("failed to unwrap")
	}

	// log is not nil
	data, err := json.Marshal(log)
	if err != nil {
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}

// doc: ["wrap", pending-balance-ID, sender-ID, receiver-ID, amount, bridge-Code, bridge-ID, memo]
func executeWrap(stub shim.ChaincodeStubInterface, cid string, doc []interface{}) peer.Response {
	// param check
	if len(doc) < 8 {
		return shim.Error("invalid contract document")
	}

	// pending balance
	bb := NewBalanceStub(stub)
	pb, err := bb.GetPendingBalance(doc[1].(string))
	if err != nil {
		return shim.Error("failed to get the pending balance")
	}

	// validate
	if pb.Type != PendingBalanceTypeContract || pb.RID != cid {
		return shim.Error("invalid pending balance")
	}

	// receiver balance
	rBal, err := bb.GetBalance(doc[3].(string))
	if err != nil {
		return shim.Error("failed to get the receiver's balance")
	}

	log, err := bb.WrapPendingBalance(pb, rBal, doc[5].(string), doc[6].(string))
	if err != nil {
		return shim.Error("failed to wrap")
	}

	// log is not nil
	data, err := json.Marshal(log)
	if err != nil {
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}
