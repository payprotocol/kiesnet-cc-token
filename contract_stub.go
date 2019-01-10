// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
)

// ContractsFetchSize _
const ContractsFetchSize = 20

// ContractStub _
type ContractStub struct {
	stub shim.ChaincodeStubInterface
}

// NewContractStub _
func NewContractStub(stub shim.ChaincodeStubInterface) *ContractStub {
	return &ContractStub{stub}
}

// CreateKey _
func (cb *ContractStub) CreateKey(id, signer string) string {
	return fmt.Sprintf("CTR_%s_%s", id, signer)
}

// CreateHash _
func (cb *ContractStub) CreateHash(text string) string {
	h := make([]byte, 32)
	sha3.ShakeSum256(h, []byte(text))
	return hex.EncodeToString(h)
}

// CreateContracts _
func (cb *ContractStub) CreateContracts(creator, document string, signers *stringset.Set, expiry int64) (*Contract, error) {
	ts, err := txtime.GetTime(cb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	scount := signers.Size()
	var expTime *txtime.Time
	if expiry >= 600 { // minimum 10 minutes
		expTime = txtime.New(ts.Add(time.Second * time.Duration(expiry)))
	} else { // default 15 days
		expTime = txtime.New(ts.AddDate(0, 0, 15))
	}

	id := cb.CreateHash(creator + cb.stub.GetTxID())
	// check id collision
	query := CreateQueryContractsByID(id)
	iter, err := cb.stub.GetQueryResult(query)
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	if iter.HasNext() {
		return nil, errors.New("contract ID collided")
	}

	var _contract *Contract // creator's contract (for return)

	for signer := range signers.Map() {
		sign := &Sign{
			Signer: signer,
		}
		contract := &Contract{
			DOCTYPEID:     id,
			Creator:       creator,
			SignersCount:  scount,
			ApprovedCount: 1, // creator has approved
			Document:      document,
			CreatedTime:   ts,
			UpdatedTime:   ts,
			ExpiryTime:    expTime,
			FinishedTime:  expTime,
			Sign:          sign,
		}
		if creator == signer {
			sign.ApprovedTime = ts
			_contract = contract
		}
		if err = cb.PutContract(contract); err != nil {
			return nil, err
		}
	}

	return _contract, nil
}

// GetContract _
func (cb *ContractStub) GetContract(id, signer string) (*Contract, error) {
	data, err := cb.stub.GetState(cb.CreateKey(id, signer))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the contract state")
	}
	if data != nil {
		contract := &Contract{}
		if err = json.Unmarshal(data, contract); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal the contract")
		}
		return contract, nil
	}
	return nil, NotExistedContractError{id: id, signer: signer}
}

// PutContract _
func (cb *ContractStub) PutContract(contract *Contract) error {
	data, err := json.Marshal(contract)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the contract")
	}
	key := cb.CreateKey(contract.DOCTYPEID, contract.Sign.Signer)
	if err = cb.stub.PutState(key, data); err != nil {
		return errors.Wrap(err, "failed to put the contract state")
	}
	return nil
}

// ApproveContract _
func (cb *ContractStub) ApproveContract(contract *Contract) (*Contract, error) {
	ts, err := txtime.GetTime(cb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	if err = contract.AssertSignable(ts); err != nil {
		return nil, err
	}

	contract.Sign.ApprovedTime = ts
	contract.UpdatedTime = ts
	contract.ApprovedCount++
	if contract.SignersCount == contract.ApprovedCount {
		contract.ExecutedTime = ts
		contract.FinishedTime = ts
	}

	// update all other signers
	if err = cb.UpdateContracts(contract); err != nil {
		return nil, err
	}

	return contract, nil
}

// CancelContract _
func (cb *ContractStub) CancelContract(contract *Contract) (*Contract, error) {
	ts, err := txtime.GetTime(cb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	contract.CanceledTime = ts
	contract.FinishedTime = ts
	contract.UpdatedTime = ts

	// update all other signers
	if err = cb.UpdateContracts(contract); err != nil {
		return nil, err
	}

	return contract, nil
}

// DisapproveContract _
func (cb *ContractStub) DisapproveContract(contract *Contract) (*Contract, error) {
	ts, err := txtime.GetTime(cb.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	if err = contract.AssertSignable(ts); err != nil {
		return nil, err
	}

	contract.Sign.DisapprovedTime = ts
	contract.UpdatedTime = ts
	contract.CanceledTime = ts
	contract.FinishedTime = ts

	// update all other signers
	if err = cb.UpdateContracts(contract); err != nil {
		return nil, err
	}

	return contract, nil
}

// UpdateContracts updates contracts with values of updater
func (cb *ContractStub) UpdateContracts(updater *Contract) error {
	query := CreateQueryContractsByID(updater.DOCTYPEID)
	iter, err := cb.stub.GetQueryResult(query)
	if err != nil {
		return errors.Wrap(err, "failed to query contracts")
	}
	defer iter.Close()

	_copy := *updater // copy
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return errors.Wrap(err, "failed to get a contract")
		}
		updatee := &Contract{}
		if err = json.Unmarshal(kv.Value, updatee); err != nil {
			return errors.Wrap(err, "failed to unmarshal the contract")
		}
		if updatee.Sign.Signer != updater.Sign.Signer {
			_copy.Sign = updatee.Sign // switch signer
		} else {
			_copy.Sign = updater.Sign
		}
		if err = cb.PutContract(&_copy); err != nil {
			return errors.Wrap(err, "failed to update a contract")
		}
	}

	return nil
}

// GetQueryContracts _
// option - 1 of [finished, unfinished, approved, unsigned, all]
func (cb *ContractStub) GetQueryContracts(kid, opt, bookmark string) (*QueryResult, error) {
	ts, err := txtime.GetTime(cb.stub)
	if nil != err {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	query := ""

	switch opt {
	case "finished": // finished|finished_time|desc
		query = CreateQueryFinishedContractsBySigner(kid, ts)
	case "unfinished": // unfinished|expiry_time|asc
		query = CreateQueryUnfinishedContractsBySigner(kid, ts)
	case "approved": // unfinished|approved|expiry_time|asc
		query = CreateQueryApprovedContractsBySigner(kid, ts)
	case "unsigned": // unfinished|unsigned|expiry_time|asc
		query = CreateQueryUnsignedContractsBySigner(kid, ts)
	default: // all|created_time|desc
		query = CreateQueryContractsBySigner(kid)
	}

	iter, meta, err := cb.stub.GetQueryResultWithPagination(query, ContractsFetchSize, bookmark)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	return NewQueryResult(meta, iter)
}
