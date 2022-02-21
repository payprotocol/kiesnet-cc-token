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

// params[0] : wrap key (wrap tx id)
// params[1] : fee (big int string) must bigger than or equal to 0
// params[2] : external tx id (if it is nil, it is 'impossible wrap')
func wrapComplete(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	// param check
	if len(params) < 1 {
		return shim.Error("incorrect number of parameters. expecting 1+")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	// wrap key (wrap tx id)
	wrapKey := params[0]

	var fee *Amount
	if len(params) > 1 {
		// check fee format even when it can be ignored (preventing abused arguments)
		fee, err = NewAmount(params[1])
		if err != nil {
			return shim.Error(err.Error())
		}
		if fee.Sign() < 0 {
			return shim.Error("invalid fee. must be greater than or equal to 0")
		}
	}

	// if 'external tx id' is not exist, it is 'impossible wrap' and fee will be ignored
	extTxID := ""
	if len(params) > 2 {
		extTxID, err = NormalizeExtTxID(params[2])
		if err != nil {
			return shim.Error("invalid ext tx id")
		}
	} else {
		fee = ZeroAmount()
	}

	wb := NewWrapStub(stub)
	wrap, err := wb.GetWrap(wrapKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	if wrap.CompleteTxID != "" {
		return shim.Error(DuplicateWrapCompleteError{}.Error())
	}

	code, _ := ParseCode(wrap.Address)
	token, err := NewTokenStub(stub).GetToken(code)
	if err != nil {
		return responseError(err, "failed to get the token")
	}
	wAddr, err := token.GetWrapAddress(wrap.ExtCode)

	// wrapper(wrap account)
	ab := NewAccountStub(stub, code)
	wrapper, err := ab.GetAccount(wAddr)
	if err != nil {
		return shim.Error("failed to get the wrap account")
	}
	if !wrapper.HasHolder(kid) {
		return shim.Error("invoker is not wrapper")
	}
	// can wrapper be suspended?
	if wrapper.IsSuspended() {
		return shim.Error("the wrap account is suspended")
	}

	bb := NewBalanceStub(stub)
	wBal, err := bb.GetBalance(wrapper.GetID())
	if err != nil {
		return shim.Error("failed to get the balance of the wrap account")
	}

	// amount := wrap.Amount.Copy().Neg()
	log, err := wb.WrapComplete(wrap, wBal, *fee, extTxID)
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
// params[3] : external tx id
// params[4] : amount (big int string) must bigger than 0
func unwrap(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	// param check
	if len(params) < 5 {
		return shim.Error("incorrect number of parameters. expecting 5")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	// external txid
	extTxID, err := NormalizeExtTxID(params[3])
	if err != nil {
		return shim.Error("invalid ext tx id")
	}

	wb := NewWrapStub(stub)

	// check unwrap duplication (move to stub?)
	data, err := wb.stub.GetState(wb.CreateUnwrapKey(extTxID))
	if err != nil {
		return responseError(err, "failed to get unwrap state")
	}
	if data != nil {
		return shim.Error(DuplicateUnwrapCompleteError{}.Error())
	}
	unwrap := &Unwrap{
		DOCTYPEID:    extTxID,
		CompleteTxID: wb.stub.GetTxID(),
	}
	if err = wb.PutUnwrap(unwrap); err != nil {
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
	if rAddr == nil { // param[0] was token code (impossible unwrap)
		rAddr = wAddr
	}

	ab := NewAccountStub(stub, code)

	// wrapper(wrap account)
	wrapper, err := ab.GetAccount(wAddr)
	if err != nil {
		return shim.Error("failed to get the wrap account")
	}
	if !wrapper.HasHolder(kid) {
		return shim.Error("invoker is not wrapper")
	}
	// can wrapper be suspended?
	if wrapper.IsSuspended() {
		return shim.Error("the wrap account is suspended")
	}

	// receiver
	if rAddr == nil { // param[0] was token code (impossible unwrap)
		rAddr = wAddr
	}
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
		return shim.Error("failed to get the balance of the wrap account")
	}

	var log *BalanceLog
	if !wAddr.Equal(rAddr) {
		// normal unwrap
		// wrap acocunt balance check
		if wBal.Amount.Cmp(amount) < 0 {
			return shim.Error("not enough balance")
		}
		log, err = wb.Unwrap(wBal, rBal, *amount, extCode, extID, extTxID)
		if err != nil {
			return responseError(err, "failed to unwrap")
		}
	} else {
		// unwrap error handle
		log, err = wb.UnwrapImpossible(wBal, extCode, extID, extTxID)
		if err != nil {
			return responseError(err, "failed to unwrap")
		}
	}

	data, err = json.Marshal(log)
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

	wrap, err := NewWrapStub(stub).WrapPendingBalance(pb, sBal, doc[4].(string), doc[5].(string))
	if err != nil {
		return shim.Error("failed to wrap")
	}

	memo := ""
	if len(doc) > 6 {
		memo = doc[6].(string)
	}
	log := (struct {
		DOCTYPEID string         `json:"@balance_log"` // address
		Type      BalanceLogType `json:"type"`
		RID       string         `json:"rid"` // EOA
		Diff      Amount         `json:"diff"`
		ExtCode   string         `json:"ext_code,omitempty"`
		Memo      string         `json:"memo,omitempty"`
	}{
		DOCTYPEID: wrap.Address,
		Type:      BalanceLogTypeWrap,
		RID:       wrap.ExtID,
		Diff:      wrap.Amount,
		ExtCode:   wrap.ExtCode,
		Memo:      memo,
	}) // hide balance amount

	data, err := json.Marshal(log)
	if err != nil {
		return shim.Error("failed to marshal the log")
	}

	return shim.Success(data)
}
