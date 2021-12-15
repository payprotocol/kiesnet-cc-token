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
// params[2] : external adress(EOA)
// params[3] : amount (big int string) must bigger than 0
// params[4] : expiry (duration represented by int64 seconds, multi-sig only)
// params[5:] : extra signers (personal account addresses)
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

	// sender address check
	// sAddr, err := ParseAddress(params[0])
	// if err != nil {
	// 	return shim.Error("failed to parse the sender's account address")
	// }
	// code, err := ValidateTokenCode(sAddr.Code)
	// if err != nil {
	// 	return shim.Error(err.Error())
	// }

	//
	var sAddr *Address
	code, err := ValidateTokenCode(params[0])
	if err != nil { // by address
		// receiver address check
		sAddr, err = ParseAddress(params[0])
		if err != nil {
			return responseError(err, "failed to parse the receiver account address")
		}
		code, err = ValidateTokenCode(sAddr.Code)
		if err != nil {
			return shim.Error(err.Error())
		}
	} else {
		sAddr = NewAddress(code, AccountTypePersonal, kid)
	}
	//

	// receiver address get
	tb := NewTokenStub(stub)
	token, err := tb.GetToken(code)
	if err != nil {
		return responseError(err, "failed to get the token")
	}
	// token code check and get wrap address
	wAddr, err := ParseWrapAddress(params[1], *token)
	if err != nil {
		return shim.Error(err.Error())
	}

	// sender = receiver
	if sAddr.Equal(wAddr) {
		return shim.Error("wrap address cannot wrap self")
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

	// wrapper(wrap account)
	wrapper, err := ab.GetAccount(wAddr)
	if err != nil {
		return responseError(err, "failed to get the wrap account")
	}
	// can be suspended?
	if wrapper.IsSuspended() {
		return shim.Error("the wrap account is suspended")
	}

	// balance check
	bb := NewBalanceStub(stub)
	sBal, err := bb.GetBalance(sender.GetID())
	if err != nil {
		return shim.Error("failed to get the sender's balance")
	}

	// fee check
	fee, err := ParseWrapFee(params[1], *token)
	if err != nil {
		logger.Debug(err.Error())
		return shim.Error("failed to get the fee amount")
	}

	// banlance must bigger than amount+fee
	applied := amount.Copy().Add(fee)
	if sBal.Amount.Cmp(applied) < 0 {
		return shim.Error("not enough balance")
	}

	var expiry int64
	signers := stringset.New(kid)
	if a, ok := sender.(*JointAccount); ok {
		signers.AppendSet(a.Holders)
	}
	if len(params) > 3 {
		// expiry
		if len(params) > 4 && len(params[4]) > 0 {
			expiry, err = strconv.ParseInt(params[4], 10, 64)
			if err != nil {
				return shim.Error("invalid expiry: need seconds")
			}
			// extra signers
			if len(params) > 5 {
				addrs := stringset.New(params[5:]...) // remove duplication
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
		doc := []string{"wrap", pbID, sender.GetID(), amount.String(), extCode, extID}
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
		log, err = bb.Deposit(pbID, sBal, con, *amount, fee, "")
		if err != nil {
			logger.Debug(err.Error())
			return shim.Error("failed to create the pending balance")
		}
	} else {
		wb := NewWrapStub(stub)
		log, err = wb.Wrap(sBal, *amount, *fee, extCode, extID)
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

// params[0] : receiver address | token code (bridge error handling)
// params[1] : external token code(wpci, ...)
// params[2] : external adress(wpci, ...)
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

	// txid, need to check validate?
	extTxID := params[3]

	// amount check
	amount, err := NewAmount(params[4])
	if err != nil {
		return shim.Error(err.Error())
	}
	if amount.Sign() <= 0 {
		return shim.Error("invalid amount. must be greater than 0")
	}

	var rAddr *Address
	code, err := ValidateTokenCode(params[0])
	if err != nil { // by address
		// receiver address check
		rAddr, err = ParseAddress(params[0])
		if err != nil {
			return responseError(err, "failed to parse the receiver account address")
		}
		code, err = ValidateTokenCode(rAddr.Code)
		if err != nil {
			return shim.Error(err.Error())
		}
	}

	// get token
	token, err := NewTokenStub(stub).GetToken(code)
	if err != nil {
		return responseError(err, "failed to get the token")
	}

	// check wrap address with external code
	wAddr, err := ParseWrapAddress(extCode, *token)
	if err != nil {
		return shim.Error(err.Error())
	}

	// unwrap address get amount
	if rAddr == nil {
		rAddr = wAddr
	}

	// account check
	ab := NewAccountStub(stub, code)
	// wrapper(wrap account)
	wrapper, err := ab.GetAccount(wAddr)
	if err != nil {
		return shim.Error("failed to get the wrap account")
	}
	if !wrapper.HasHolder(kid) {
		return shim.Error("invoker is not holder")
	}
	if wrapper.IsSuspended() { //need suspend check?
		return shim.Error("the wrap account is suspended")
	}

	// receiver
	receiver, err := ab.GetAccount(rAddr)
	if err != nil {
		return responseError(err, "failed to get the receiver account")
	}
	if receiver.IsSuspended() {
		return shim.Error("the receiver account is suspended")
	}

	// balance no need to balance check
	bb := NewBalanceStub(stub)
	rBal, err := bb.GetBalance(receiver.GetID())
	if err != nil {
		return shim.Error("failed to get the receiver's balance")
	}

	wb := NewWrapStub(stub)
	log, err := wb.UnWrap(rBal, *amount, extCode, extID, extTxID)
	if err != nil {
		return responseError(err, "failed to unwrap")
	}

	data, err := json.Marshal(log)
	if err != nil {
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}

// doc: ["wrap", pending-balance-ID, sender-ID, amount, external-Code, external-adress]
func executeWrap(stub shim.ChaincodeStubInterface, cid string, doc []interface{}) peer.Response {
	// param check
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

	log, err := NewWrapStub(stub).WrapPendingBalance(pb, sBal, doc[4].(string), doc[5].(string))
	if err != nil {
		return shim.Error("failed to wrap")
	}

	data, err := json.Marshal(log)
	if err != nil {
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}
