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
func (ab *AccountStub) CreateKey(addr string) string {
	return "ACC_" + addr
}

// CreateAccount _
func (ab *AccountStub) CreateAccount(kid string) (*Account, error) {
	ts, err := txtime.GetTime(ab.stub)
	if err != nil {
		return nil, err
	}

	addr := NewAddress(ab.token, AccountTypePersonal, kid)
	account, err := ab.GetAccount(addr)
	if err != nil {
		if _, ok := err.(NotExistedAccountError); !ok {
			return nil, errors.Wrap(err, "failed to create an account")
		}

		// create personal account
		account, _ := NewAccount(addr) // err will not occur
		account.CreatedTime = ts
		account.UpdatedTime = ts
		if err = ab.PutAccount(account); err != nil {
			return nil, errors.Wrap(err, "failed to create an account")
		}

		// create account-holder relationship
		rel := NewAccountHolder(kid, account)
		rel.CreatedTime = ts
		if err = ab.PutAccountHolder(rel); err != nil {
			return nil, errors.Wrap(err, "failed to create an account-holder")
		}

		return account, nil
	}

	if account.GetID() != kid { // CRITICAL: hash collision
		logger.Criticalf("account address collision: %s, %s", kid, addr.String())
		return nil, errors.New("hash collision: need to change username from CA")
	}
	return nil, ExistedAccountError{addr.String()}
}

// CreateJointAccount _
func (ab *AccountStub) CreateJointAccount(holders stringset.Set) (*JointAccount, error) {
	ts, err := txtime.GetTime(ab.stub)
	if err != nil {
		return nil, err
	}

	// random id
	h := make([]byte, 32)
	sha3.ShakeSum256(h, []byte(ab.stub.GetTxID()))
	id := hex.EncodeToString(h)

	addr := NewAddress(ab.token, AccountTypeJoint, id)
	if _, err = ab.GetAccount(addr); err != nil {
		if _, ok := err.(NotExistedAccountError); !ok {
			return nil, errors.Wrap(err, "failed to create an account")
		}

		// create joint account
		account, _ := NewJointAccount(addr, holders) // err will not occur
		account.CreatedTime = ts
		account.UpdatedTime = ts
		if err = ab.PutAccount(account); err != nil {
			return nil, errors.Wrap(err, "failed to create an account")
		}

		// cretae account-holder relationship
		for kid := range holders {
			rel := NewAccountHolder(kid, account)
			rel.CreatedTime = ts
			if err = ab.PutAccountHolder(rel); err != nil {
				return nil, errors.Wrap(err, "failed to create an account-holder")
			}
		}

		return account, nil
	}

	// hash collision (retry later)
	return nil, errors.New("failed to create a random address")
}

// GetAccount retrieves the account by an address
// An address is hash string, so, it can be collided.
// To retrieve the account of a specific KID, use GetPersonalAccount or GetQueryPersonalAccount.
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
	data, err := ab.GetQueryPersonalAccount(kid)
	if err != nil {
		return nil, err
	}
	// data is not nil
	account := &Account{}
	if err = json.Unmarshal(data, account); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal the account")
	}
	return account, nil
}

// GetQueryPersonalAccount retrieves the main(personal) account by a token code and an ID(KID)
func (ab *AccountStub) GetQueryPersonalAccount(kid string) ([]byte, error) {
	query := CreateQueryPersonalAccountByIDAndTokenCode(kid, ab.token)
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

// GetQueryHolderAccounts _
func (ab *AccountStub) GetQueryHolderAccounts(kid, bookmark string) (*QueryResult, error) {
	var query string
	if len(ab.token) > 0 {
		query = CreateQueryAccountHoldersByIDAndTokenCode(kid, ab.token)
	} else {
		query = CreateQueryAccountHoldersByID(kid)
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
	if err = ab.stub.PutState(ab.CreateKey(account.GetAddress()), data); err != nil {
		return errors.Wrap(err, "failed to put the account state")
	}
	return nil
}

// CreateAccountHolderKey _
func (ab *AccountStub) CreateAccountHolderKey(rel *AccountHolder) string {
	return "ACC_HLD_" + rel.DOCTYPEID + "_" + rel.Account.Address
}

// PutAccountHolder _
func (ab *AccountStub) PutAccountHolder(rel *AccountHolder) error {
	data, err := json.Marshal(rel)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the account-holder")
	}
	if err = ab.stub.PutState(ab.CreateAccountHolderKey(rel), data); err != nil {
		return errors.Wrap(err, "failed to put the account-holder state")
	}
	return nil
}
