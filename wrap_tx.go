package main

import (
	"encoding/json"
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
// params[4] : fee (big int string) must bigger than 0
// params[5] : memo (see MemoMaxLength)
// params[6] : expiry (duration represented by int64 seconds, multi-sig only)
// params[7:] : extra signers (personal account addresses)
func wrap(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	//1. param check
	if len(params) < 5 {
		return shim.Error("incorrect number of parameters. expecting 3+")
	}

	//authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	//external code
	extCode := params[1]
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
	if err != nil {
		return shim.Error(err.Error())
	}

	// receiver address get
	tb := NewTokenStub(stub)
	token, err := tb.GetToken(code)
	if err != nil {
		return responseError(err, "failed to get the token")
	}
	//token code check and get wrap address
	rAddr, err := ParseWrapAddress(params[1], *token)
	if err != nil {
		return shim.Error(err.Error())
	}

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

	//fee check
	fee, err := NewAmount(params[4])
	if err != nil {
		return shim.Error(err.Error())
	}
	if amount.Sign() <= 0 {
		return shim.Error("invalid fee. must be greater than 0")
	}

	applied := amount.Copy().Add(fee)

	// asume there's no fee in here
	if sBal.Amount.Cmp(applied) < 0 {
		return shim.Error("not enough balance")
	}

	memo := ""
	var expiry int64
	signers := stringset.New(kid)
	if a, ok := sender.(*JointAccount); ok {
		signers.AppendSet(a.Holders)
	}
	if len(params) > 5 {
		// memo
		if len(params[5]) > MemoMaxLength { // length limit
			memo = params[5][:MemoMaxLength]
		} else {
			memo = params[5]
		}
		// expiry
		if len(params) > 6 && len(params[6]) > 0 {
			expiry, err = strconv.ParseInt(params[6], 10, 64)
			if err != nil {
				return shim.Error("invalid expiry: need seconds")
			}
			// extra signers
			if len(params) > 6 {
				addrs := stringset.New(params[7:]...) // remove duplication
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
	wrapResult := &WrapResult{}
	if signers.Size() > 1 {
		//TODO multisig
		if signers.Size() > 128 {
			return shim.Error("too many signers")
		}
		// pending balance id
		pbID := stub.GetTxID()
		doc := []string{"wrap", pbID, sender.GetID(), receiver.GetID(), extCode, extID, amount.String(), fee.String(), memo}
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
		log, err = bb.Deposit(pbID, sBal, con, *amount, fee, memo)
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to create the pending balance")
		}
	} else {
		wb := NewWrapStub(stub)
		wrapResult, err = wb.Wrap(sBal, rBal, *amount, *fee, extID, memo)
		if err != nil {
			return shim.Error("failed to wrap")
		}
	}

	if wrapResult.BalanceLog == nil {
		wrapResult.BalanceLog = log
	}

	data, err := json.Marshal(wrapResult)
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
	if err != nil {
		return shim.Error(err.Error())
	}

	// receiver address get
	tb := NewTokenStub(stub)
	token, err := tb.GetToken(code)
	if err != nil {
		return responseError(err, "failed to get the token")
	}
	//check wrap address with external code
	sAddr, err := ParseWrapAddress(params[1], *token)
	if err != nil {
		return shim.Error(err.Error())
	}

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

	wb := NewWrapStub(stub)
	wrapResult, err := wb.UnWrap(sBal, rBal, *amount, extID, extTxID, memo)
	if err != nil {
		return shim.Error("failed to unwrap")
	}

	data, err := json.Marshal(wrapResult)
	if err != nil {
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}

// doc: ["wrap", pending-balance-ID, sender-ID, receiver-ID, external-Code, external-adress, amount, fee, memo]
func executeWrap(stub shim.ChaincodeStubInterface, cid string, doc []interface{}) peer.Response {
	// param check
	if len(doc) < 9 {
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

	// sender balance
	sBal, err := bb.GetBalance(doc[3].(string))
	if err != nil {
		return shim.Error("failed to get the sender's balance")
	}

	// receiver balance
	rBal, err := bb.GetBalance(doc[3].(string))
	if err != nil {
		return shim.Error("failed to get the receiver's balance")
	}

	wrapResult, err := NewWrapStub(stub).WrapPendingBalance(pb, sBal, rBal, doc[5].(string))
	if err != nil {
		return shim.Error("failed to wrap")
	}

	data, err := json.Marshal(wrapResult)
	if err != nil {
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}
