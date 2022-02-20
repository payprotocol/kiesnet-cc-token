package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/contract"
	"github.com/key-inside/kiesnet-ccpkg/kid"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
)

// params[0] : sender address | token code
// params[1] : external token code(wpci, ...)
// params[2] : external address(EOA)
// params[3] : amount (big int string) must bigger than 0
// params[4] : memo (see MemoMaxLength)
// params[5] : expiry (duration represented by int64 seconds, multi-sig only)
// params[6:] : extra signers (personal account addresses)
func wrap(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	// param check
	if len(params) < 4 {
		return shim.Error("incorrect number of parameters. expecting 4+")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	// external code
	extCode := strings.ToUpper(params[1])

	// external address
	extID, err := NormalizeExtAddress(params[2])
	if err != nil {
		return shim.Error("invalid ext address")
	}

	// amount check
	amount, err := NewAmount(params[3])
	if err != nil {
		return shim.Error(err.Error())
	}
	if amount.Sign() <= 0 {
		return shim.Error("invalid amount. must be greater than 0")
	}

	// addresses
	var sAddr *Address
	code, err := ValidateTokenCode(params[0])
	if err == nil { // by token code
		sAddr = NewAddress(code, AccountTypePersonal, kid)
	} else { // by address
		sAddr, err = ParseAddress(params[0])
		if err != nil {
			return responseError(err, "failed to parse the sender's account address")
		}
	}
	token, err := NewTokenStub(stub).GetToken(sAddr.Code)
	if err != nil {
		return responseError(err, "failed to get the token")
	}
	wAddr, err := token.GetWrapAddress(extCode)
	if err != nil {
		return shim.Error(err.Error())
	}

	ab := NewAccountStub(stub, sAddr.Code)

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

	// IMPORTANT: assert(sender != wrapper)
	if sAddr.Equal(wAddr) {
		return shim.Error("wrap address cannot wrap self")
	}

	// wrapper(wrap account)
	wrapper, err := ab.GetAccount(wAddr)
	if err != nil {
		return responseError(err, "failed to get the wrap account")
	}
	// can wrapper be suspended?
	if wrapper.IsSuspended() {
		return shim.Error("the wrap account is suspended")
	}

	// balance check
	bb := NewBalanceStub(stub)
	sBal, err := bb.GetBalance(sender.GetID())
	if err != nil {
		return shim.Error("failed to get the sender's balance")
	}

	// balance must bigger than amount
	if sBal.Amount.Cmp(amount) < 0 {
		return shim.Error("not enough balance")
	}

	memo := ""
	var expiry int64
	signers := stringset.New(kid)
	if a, ok := sender.(*JointAccount); ok {
		signers.AppendSet(a.Holders)
	}
	// memo
	if len(params) > 4 {
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
		// multisig
		if signers.Size() > 128 {
			return shim.Error("too many signers")
		}
		// pending balance id
		pbID := stub.GetTxID()
		doc := []string{"wrap", pbID, sender.GetID(), amount.String(), extCode, extID, memo}
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
		log, err = bb.Deposit(pbID, sBal, con, *amount, nil, "")
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to create the pending balance")
		}
	} else {
		wb := NewWrapStub(stub)
		log, err = wb.Wrap(sBal, *amount, extCode, extID, memo)
		if err != nil {
			return shim.Error("failed to wrap")
		}
	}

	data, err := json.Marshal(log)
	if err != nil {
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)

}

// params[0] : token code
// params[1] : external token code(wpci, ...)
// params[2] : external address(EOA)
// params[3] : hlf txid
// params[4] : balance log _id
// params[5] : fee (big int string) must bigger than or equal to 0
func wrapComplete(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	// param check
	if len(params) < 6 {
		return shim.Error("incorrect number of parameters. expecting 6+")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	// external code
	extCode := strings.ToUpper(params[1])

	// external address
	extID, err := NormalizeExtAddress(params[2])
	if err != nil {
		return shim.Error("invalid ext address")
	}

	// hlf txid (need to validate?)
	txID := params[3]

	// balance log id
	blId := params[4]

	// fee check
	fee, err := NewAmount(params[5])
	if err != nil {
		return shim.Error(err.Error())
	}
	if fee.Sign() < 0 {
		return shim.Error("invalid fee. must be greater than or equal to 0")
	}

	// addresses
	code, err := ValidateTokenCode(params[0])
	if err != nil { // invalide token code
		return shim.Error(err.Error())
	}
	token, err := NewTokenStub(stub).GetToken(code)
	if err != nil {
		return responseError(err, "failed to get the token")
	}
	wAddr, err := token.GetWrapAddress(extCode)

	ab := NewAccountStub(stub, code)

	// wrapper(wrap account)
	wrapper, err := ab.GetAccount(wAddr)
	if err != nil {
		return shim.Error("failed to get the wrap account")
	}
	if !wrapper.HasHolder(kid) {
		return shim.Error("invoker is not holder")
	}
	// can wrapper be suspended?
	if wrapper.IsSuspended() {
		return shim.Error("the wrap account is suspended")
	}

	// balance no need to balance check
	bb := NewBalanceStub(stub)

	// balance log query
	bl, err := bb.GetQueryBalaceLogByDocumentID(blId)
	if err != nil {
		// return responseError(err, "failed to get balance logs")
		return shim.Error(err.Error())
	}

	wBal, err := bb.GetBalance(wrapper.GetID())
	if err != nil {
		return shim.Error("failed to get the receiver's balance")
	}

	wb := NewWrapStub(stub)
	log, err := wb.WrapComplete(wBal, bl.Diff, *fee, extCode, extID, txID)
	if err != nil {
		return shim.Error(err.Error())
	}

	data, err := json.Marshal(log)
	if err != nil {
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}

// params[0] : receiver address | token code (bridge error handling)
// params[1] : external token code(wpci, ...)
// params[2] : external address(EOA)
// params[3] : external token txid
// params[4] : amount (big int string) must bigger than 0
func unwrap(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	// param check
	if len(params) < 5 {
		return shim.Error("incorrect number of parameters. expecting 5+")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	// external code
	extCode := strings.ToUpper(params[1])

	// external address
	extID, err := NormalizeExtAddress(params[2])
	if err != nil {
		return shim.Error("invalid ext address")
	}

	// txid (need to validate?)
	extTxID := params[3]

	// amount check
	amount, err := NewAmount(params[4])
	if err != nil {
		return shim.Error(err.Error())
	}
	if amount.Sign() <= 0 {
		return shim.Error("invalid amount. must be greater than 0")
	}

	// addresses
	var rAddr *Address
	code, err := ValidateTokenCode(params[0])
	if err != nil { // by address
		rAddr, err = ParseAddress(params[0])
		if err != nil {
			return responseError(err, "failed to parse the receiver's account address")
		}
		code = rAddr.Code
	}
	token, err := NewTokenStub(stub).GetToken(code)
	if err != nil {
		return responseError(err, "failed to get the token")
	}
	wAddr, err := token.GetWrapAddress(extCode)
	if err != nil {
		return shim.Error(err.Error())
	}
	if rAddr == nil { // param[0] was token code
		rAddr = wAddr
	}

	ab := NewAccountStub(stub, code)
	// wrapper(wrap account)
	wrapper, err := ab.GetAccount(wAddr)
	if err != nil {
		return shim.Error("failed to get the wrap account")
	}
	if !wrapper.HasHolder(kid) {
		return shim.Error("invoker is not holder")
	}
	// can wrapper be suspended?
	if wrapper.IsSuspended() {
		return shim.Error("the wrap account is suspended")
	}
	if rAddr == nil { // param[0] was token code
		rAddr = wAddr
	}

	// receiver
	receiver, err := ab.GetAccount(rAddr)
	if err != nil {
		return responseError(err, "failed to get the receiver account")
	}
	if receiver.IsSuspended() {
		return shim.Error("the receiver account is suspended")
	}

	// balance
	bb := NewBalanceStub(stub)
	rBal, err := bb.GetBalance(receiver.GetID())
	if err != nil {
		return shim.Error("failed to get the receiver's balance")
	}

	// wrapper balance
	wBal, err := bb.GetBalance(wrapper.GetID())
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the wrap balance")
	}

	wb := NewWrapStub(stub)
	var log *BalanceLog
	if !wAddr.Equal(rAddr) {
		// normal unwrap
		// need balace check
		if wBal.Amount.Cmp(amount) < 0 {
			return shim.Error("not enough balance")
		}
		log, err = wb.Unwrap(wBal, rBal, *amount, extCode, extID, extTxID)
		if err != nil {
			return responseError(err, "failed to unwrap")
		}
	} else {
		// unwrap error handle
		log, err = wb.UnwrapComplete(wBal, extCode, extID, extTxID)
		if err != nil {
			return responseError(err, "failed to unwrap complete")
		}
	}

	data, err := json.Marshal(log)
	if err != nil {
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}

// doc: ["wrap", pending-balance-ID, sender-ID, amount, external-code, external-address, memo]
func executeWrap(stub shim.ChaincodeStubInterface, cid string, doc []interface{}) peer.Response {
	if len(doc) < 6 {
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
	sBal, err := bb.GetBalance(doc[2].(string))
	if err != nil {
		return shim.Error("failed to get the sender's balance")
	}

	log, err := NewWrapStub(stub).WrapPendingBalance(pb, sBal, doc[4].(string), doc[5].(string), doc[6].(string))
	if err != nil {
		return shim.Error("failed to wrap")
	}

	data, err := json.Marshal(log)
	if err != nil {
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}
