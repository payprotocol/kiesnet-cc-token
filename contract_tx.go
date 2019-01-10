// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/key-inside/kiesnet-ccpkg/kid"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// CtrFunc _
type CtrFunc func(stub shim.ChaincodeStubInterface, cid string, doc []interface{}) peer.Response

// routes is the map of contract functions
var ctrRoutes = map[string][]CtrFunc{
	"account/create":        []CtrFunc{contractVoid, executeAccountCreate},
	"account/holder/add":    []CtrFunc{contractVoid, executeAccountHolderAdd},
	"account/holder/remove": []CtrFunc{contractVoid, executeAccountHolderRemove},
	"token/burn":            []CtrFunc{contractVoid, executeTokenBurn},
	"token/create":          []CtrFunc{contractVoid, executeTokenCreate},
	"token/mint":            []CtrFunc{contractVoid, executeTokenMint},
	"transfer":              []CtrFunc{cancelTransfer, executeTransfer},
}

// params[0] : contract ID
func contractApprove(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 1 {
		return shim.Error("incorrect number of parameters. expecting 1")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	id := params[0]

	cb := NewContractStub(stub)
	contract, err := cb.GetContract(id, kid)
	if err != nil {
		return responseError(err, "failed to approve the contract")
	}
	contract, err = cb.ApproveContract(contract)
	if err != nil {
		return responseError(err, "failed to approve the contract")
	}

	if contract.ExecutedTime != nil {
		// execute contract
		return contractCallback(stub, 1, contract)
		// if err = invokeExecuteContract(stub, contract); err != nil {
		// 	return shim.Error("failed to execute the contract: " + err.Error())
		// }
	}

	return response(contract)
}

// params[0] : contract ID
func contractCancel(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 1 {
		return shim.Error("incorrect number of parameters. expecting 1")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	id := params[0]

	ts, err := txtime.GetTime(stub)
	if err != nil {
		return responseError(err, "failed to cancel the contract")
	}

	cb := NewContractStub(stub)
	contract, err := cb.GetContract(id, kid)
	if err != nil {
		return shim.Error(err.Error())
	}
	// validate
	if contract.FinishedTime != nil && ts.Cmp(contract.FinishedTime) >= 0 { // ts >= finished_time => expired
		return shim.Error("already finished contract")
	}

	if contract, err = cb.CancelContract(contract); err != nil {
		return responseError(err, "failed to cancel the contract")
	}

	return response(contract)
}

// params[0] : contract ID
func contractDisapprove(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 1 {
		return shim.Error("incorrect number of parameters. expecting 1")
	}

	// authentication
	kid, err := kid.GetID(stub, true)
	if err != nil {
		return shim.Error(err.Error())
	}

	id := params[0]

	cb := NewContractStub(stub)
	contract, err := cb.GetContract(id, kid)
	if err != nil {
		return shim.Error(err.Error())
	}
	contract, err = cb.DisapproveContract(contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	// cancel contract
	return contractCallback(stub, 0, contract)
	// if err = contractCallback(stub, 0, contract); err != nil {
	// 	return shim.Error("failed to cancel the contract: " + err.Error())
	// }

	// return response(contract)
}

// params[0] : contract ID
func contractGet(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) != 1 {
		return shim.Error("incorrect number of parameters. expecting 1")
	}

	// authentication
	kid, err := kid.GetID(stub, false)
	if err != nil {
		return shim.Error(err.Error())
	}

	id := params[0]

	cb := NewContractStub(stub)
	contract, err := cb.GetContract(id, kid)
	if err != nil {
		return responseError(err, "failed to get the contract")
	}

	return response(contract)
}

// params[0] : option
// params[1] : bookmark
func contractList(stub shim.ChaincodeStubInterface, params []string) peer.Response {
	if len(params) < 1 {
		return shim.Error("incorrect number of parameters. expecting 1+")
	}

	// authentication
	kid, err := kid.GetID(stub, false)
	if err != nil {
		return shim.Error(err.Error())
	}

	option := params[0]
	bookmark := ""
	if len(params) > 1 {
		bookmark = params[1]
	}

	cb := NewContractStub(stub)
	res, err := cb.GetQueryContracts(kid, option, bookmark)
	if nil != err {
		return responseError(err, "failed to get contracts list")
	}

	return response(res)
}

// helpers

// func invokeCallback(stub shim.ChaincodeStubInterface, ccid string, args [][]byte) error {
// 	res := stub.InvokeChaincode(ccid, args, "")
// 	if res.GetStatus() == 200 {
// 		return nil
// 	}
// 	return errors.New(res.GetMessage())
// }

// func invokeExecuteContract(stub shim.ChaincodeStubInterface, contract *Contract) error {
// 	args := [][]byte{[]byte("contract/execute"), []byte(contract.DOCTYPEID), []byte(contract.Document)}
// 	return invokeCallback(stub, contract.CCID, args)
// }

// func invokeCancelContract(stub shim.ChaincodeStubInterface, contract *Contract) error {
// 	args := [][]byte{[]byte("contract/cancel"), []byte(contract.DOCTYPEID), []byte(contract.Document)}
// 	return invokeCallback(stub, contract.CCID, args)
// }

// fnIdx : 0 = cancel, 1 = execute
// params[0] : contract ID
// params[1] : contract document
func contractCallback(stub shim.ChaincodeStubInterface, fnIdx int, contract *Contract) peer.Response {
	cid := contract.DOCTYPEID
	doc := []interface{}{}
	err := json.Unmarshal([]byte(contract.Document), &doc)
	if err != nil {
		return responseError(err, "failed to unmarshal the contract document")
	}
	dtype := doc[0].(string)
	if ctrFn := ctrRoutes[dtype][fnIdx]; ctrFn != nil {
		return ctrFn(stub, cid, doc)
	}
	return shim.Error("unknown contract: '" + dtype + "'")
}

// func contractCancel(stub shim.ChaincodeStubInterface, params []string) peer.Response {
// 	return contractCallback(stub, 0, params)
// }

// func contractExecute(stub shim.ChaincodeStubInterface, params []string) peer.Response {
// 	return contractCallback(stub, 1, params)
// }

// callback has nothing to do
func contractVoid(stub shim.ChaincodeStubInterface, cid string, doc []interface{}) peer.Response {
	return shim.Success(nil)
}

// helpers

func contractCreate(stub shim.ChaincodeStubInterface, kid string, doc []interface{}, signers *stringset.Set) peer.Response {
	if nil == signers {
		return shim.Error("not enough signers")
	}
	signers.Add(kid)

	if signers.Size() < 2 {
		return shim.Error("not enough signers")
	} else if signers.Size() > 128 {
		return shim.Error("too many signers")
	}

	docb, err := json.Marshal(doc)
	if err != nil {
		return responseError(err, "failed to marshal the contract document")
	}
	document := string(docb)
	expiry := int64(0)

	cb := NewContractStub(stub)
	contract, err := cb.CreateContracts(kid, document, signers, expiry)
	if err != nil {
		return responseError(err, "failed to create the contract")
	}

	return response(contract)
}
