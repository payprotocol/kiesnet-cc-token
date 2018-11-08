// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/hex"
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/stringset"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
)

// AccountsFetchSize _
const AccountsFetchSize = 20

// AccountStub _
type AccountStub struct {
	stub shim.ChaincodeStubInterface
}

// NewAccountStub _
func NewAccountStub(stub shim.ChaincodeStubInterface) *AccountStub {
	ab := &AccountStub{}
	ab.stub = stub
	return ab
}

// CreateKey _
func (ab *AccountStub) CreateKey(addr string) string {
	return "ACC_" + addr
}

// CreateAccount _
func (ab *AccountStub) CreateAccount(tokenCode, kid string) (*Account, error) {
	ts, err := txtime.GetTime(ab.stub)
	if err != nil {
		return nil, err
	}

	addr := NewAddress(tokenCode, AccountTypePersonal, kid)
	address := addr.String()
	account, err := ab.GetAccount(addr)
	if err != nil {
		if _, ok := err.(NotExistedAccountError); !ok {
			return nil, errors.Wrap(err, "failed to create an account")
		}
		account := NewAccount(tokenCode, kid)
		account.Address = address
		account.CreatedTime = ts
		account.UpdatedTime = ts
		if err = ab.PutAccount(account); err != nil {
			return nil, errors.Wrap(err, "failed to create an account")
		}
		return account, nil
	}

	if account.GetID() != kid { // CRITICAL : hash collision
		logger.Criticalf("account address collision: %s, %s", kid, address)
		return nil, errors.New("hash collision: need to change username from CA")
	}
	return nil, ExistedAccountError{address}
}

// CreateJointAccount _
func (ab *AccountStub) CreateJointAccount(tokenCode string, holders stringset.Set) (*JointAccount, error) {
	ts, err := txtime.GetTime(ab.stub)
	if err != nil {
		return nil, err
	}

	// random id
	h := make([]byte, 32)
	sha3.ShakeSum256(h, []byte(ab.stub.GetTxID()))
	id := hex.EncodeToString(h)

	addr := NewAddress(tokenCode, AccountTypeJoint, id)
	_, err = ab.GetAccount(addr)
	if err != nil {
		if _, ok := err.(NotExistedAccountError); !ok {
			return nil, errors.Wrap(err, "failed to create an account")
		}
		account := NewJointAccount(tokenCode, id, holders)
		account.Address = addr.String()
		account.CreatedTime = ts
		account.UpdatedTime = ts
		if err = ab.PutAccount(account); err != nil {
			return nil, errors.Wrap(err, "failed to create an account")
		}
		return account, nil
	}

	// hash collision (retry later)
	return nil, errors.New("failed to create a random address")
}

// GetAccount retrieves the account by an address
// An address is hash string, so, it can be collided.
// To retrieve the account of a specific KID, use GetQueryAccount.
func (ab *AccountStub) GetAccount(addr *Address) (AccountInterface, error) {
	address := addr.String()
	data, err := ab.stub.GetState(ab.CreateKey(address))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the account state")
	}

	if data != nil {
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
	return nil, NotExistedAccountError{address}
}

// GetQueryMainAccount retrieves the main(personal) account by a token code and an ID(KID)
func (ab *AccountStub) GetQueryMainAccount(tokenCode, id string) ([]byte, error) {
	query := CreateQueryMainAccountByIDAndTokenCode(tokenCode, id)
	iter, err := ab.stub.GetQueryResult(query)
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	if iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, err
		}
		return kv.Value, nil
	}
	return nil, NotExistedAccountError{}
}

// GetQueryAccountsResult _
func (ab *AccountStub) GetQueryAccountsResult(tokenCode string, id string, bookmark string) (*QueryResult, error) {
	var query string
	if len(tokenCode) > 0 {
		query = CreateQueryAccountsByIDAndTokenCode(tokenCode, id)
	} else {
		query = CreateQueryAccountsByID(id)
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

	err = ab.stub.PutState(ab.CreateKey(account.GetAddress()), data)
	if err != nil {
		return errors.Wrap(err, "failed to put the account state")
	}
	return nil
}
