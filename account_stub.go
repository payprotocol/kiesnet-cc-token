// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
)

// AccountsFetchSize _
const AccountsFetchSize = 20

// AccountStub _
type AccountStub struct {
	stub  shim.ChaincodeStubInterface
	token string
}

// NewAccountStub _
func NewAccountStub(stub shim.ChaincodeStubInterface, tokenCode string) *AccountStub {
	return &AccountStub{
		stub:  stub,
		token: tokenCode,
	}
}

// CreateKey _
func (ab *AccountStub) CreateKey(id string) string {
	return "ACC_" + id
}

// CreateAccount _
func (ab *AccountStub) CreateAccount(kid string) (*Account, error) {
	ts, err := txtime.GetTime(ab.stub)
	if err != nil {
		return nil, err
	}

	addr := NewAddress(ab.token, AccountTypePersonal, kid)
	_, err = ab.GetAccount(addr)
	if err != nil {
		if _, ok := err.(NotExistedAccountError); !ok {
			return nil, errors.Wrap(err, "failed to create an account")
		}

		// create personal account
		account := &Account{
			DOCTYPEID:   addr.String(),
			Token:       ab.token,
			Type:        AccountTypePersonal,
			CreatedTime: ts,
			UpdatedTime: ts,
		}
		if err = ab.PutAccount(account); err != nil {
			return nil, errors.Wrap(err, "failed to create an account")
		}

		// create account-holder relationship
		holder := NewHolder(kid, account)
		holder.CreatedTime = ts
		if err = ab.PutHolder(holder); err != nil {
			return nil, errors.Wrap(err, "failed to create a holder")
		}

		return account, nil
	}

	return nil, ExistedAccountError{addr.String()}
}

// CreateJointAccount _
func (ab *AccountStub) CreateJointAccount(holders stringset.Set) (*JointAccount, error) {
	ts, err := txtime.GetTime(ab.stub)
	if err != nil {
		return nil, err
	}

	addr := NewAddress(ab.token, AccountTypeJoint, ab.stub.GetTxID()) // random address
	if _, err = ab.GetAccount(addr); err != nil {
		if _, ok := err.(NotExistedAccountError); !ok {
			return nil, errors.Wrap(err, "failed to create an account")
		}

		// create joint account
		account := &JointAccount{
			Account: Account{
				DOCTYPEID:   addr.String(),
				Token:       ab.token,
				Type:        AccountTypeJoint,
				CreatedTime: ts,
				UpdatedTime: ts,
			},
			Holders: holders,
		}
		if err = ab.PutAccount(account); err != nil {
			return nil, errors.Wrap(err, "failed to create an account")
		}

		// cretae account-holder relationship
		for kid := range holders {
			holder := NewHolder(kid, account)
			holder.CreatedTime = ts
			if err = ab.PutHolder(holder); err != nil {
				return nil, errors.Wrap(err, "failed to create a holder")
			}
		}

		return account, nil
	}

	// hash collision (retry later)
	return nil, errors.New("failed to create a random address")
}

// GetAccount retrieves the account by an address
func (ab *AccountStub) GetAccount(addr *Address) (AccountInterface, error) {
	data, err := ab.GetAccountState(addr)
	if err != nil {
		return nil, err
	}
	// data is not nil
	var account AccountInterface
	switch addr.Type {
	case AccountTypePersonal:
		account = &Account{}
	case AccountTypeJoint:
		account = &JointAccount{}
	default: // never here (addr has been validated)
		return nil, InvalidAccountAddrError{}
	}
	if err = json.Unmarshal(data, account); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal the account")
	}
	return account, nil
}

// GetAccountState _
func (ab *AccountStub) GetAccountState(addr *Address) ([]byte, error) {
	address := addr.String()
	data, err := ab.stub.GetState(ab.CreateKey(address))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the account state")
	}
	if data != nil {
		return data, nil
	}
	return nil, NotExistedAccountError{address}
}

// GetPersonalAccount retrieves the main(personal) account by a token code and an ID(KID)
func (ab *AccountStub) GetPersonalAccount(kid string) (AccountInterface, error) {
	addr := NewAddress(ab.token, AccountTypePersonal, kid)
	return ab.GetAccount(addr)
}

// GetQueryHolderAccounts _
func (ab *AccountStub) GetQueryHolderAccounts(kid, bookmark string) (*QueryResult, error) {
	var query string
	if len(ab.token) > 0 {
		query = CreateQueryHoldersByIDAndTokenCode(kid, ab.token)
	} else {
		query = CreateQueryHoldersByID(kid)
	}
	iter, meta, err := ab.stub.GetQueryResultWithPagination(query, AccountsFetchSize, bookmark)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	return NewQueryResult(meta, iter)
}

// GetSignableID _
func (ab *AccountStub) GetSignableID(addr string) (string, error) {
	_addr, err := ParseAddress(addr)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse the account address: %s", addr)
	}
	if _addr.Code != ab.token {
		return "", errors.Errorf("invalid co-holder's address (mismatched token): %s", addr)
	}
	if _addr.Type != AccountTypePersonal {
		return "", errors.Errorf("invalid co-holder's address (must be personal account address): %s", addr)
	}
	account, err := ab.GetAccount(_addr)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get the account: %s", addr)
	}
	return account.GetID(), nil
}

// PutAccount _
func (ab *AccountStub) PutAccount(account AccountInterface) error {
	data, err := json.Marshal(account)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the account")
	}
	if err = ab.stub.PutState(ab.CreateKey(account.GetID()), data); err != nil {
		return errors.Wrap(err, "failed to put the account state")
	}
	return nil
}

// CreateHolderKey _
func (ab *AccountStub) CreateHolderKey(id, addr string) string {
	return fmt.Sprintf("HLD_%s_%s", id, addr)
}

// PutHolder _
func (ab *AccountStub) PutHolder(holder *Holder) error {
	data, err := json.Marshal(holder)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the holder")
	}
	if err = ab.stub.PutState(ab.CreateHolderKey(holder.DOCTYPEID, holder.Address), data); err != nil {
		return errors.Wrap(err, "failed to put the holder state")
	}
	return nil
}
