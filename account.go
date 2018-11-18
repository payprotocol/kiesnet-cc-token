// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"time"

	"github.com/key-inside/kiesnet-ccpkg/stringset"
	"github.com/pkg/errors"
)

// AccountInterface _
type AccountInterface interface {
	GetAddress() string
	GetID() string
	GetToken() string
	GetType() AccountType
	IsSuspended() bool
}

// AccountType _
type AccountType byte

const (
	// AccountTypeUnknown _
	AccountTypeUnknown AccountType = iota
	// AccountTypePersonal _
	AccountTypePersonal
	// AccountTypeJoint _
	AccountTypeJoint
)

// Account _
type Account struct {
	DOCTYPEID     string      `json:"@account"`
	Address       string      `json:"address"`
	Token         string      `json:"token"`
	Type          AccountType `json:"type"`
	CreatedTime   *time.Time  `json:"created_time,omitempty"`
	UpdatedTime   *time.Time  `json:"updated_time,omitempty"`
	SuspendedTime *time.Time  `json:"suspended_time,omitempty"`
}

// GetAddress implements AccountInterface
func (a *Account) GetAddress() string {
	return a.Address
}

// GetID implements AccountInterface
func (a *Account) GetID() string {
	return a.DOCTYPEID
}

// GetToken implements AccountInterface
func (a *Account) GetToken() string {
	return a.Token
}

// GetType implements AccountInterface
func (a *Account) GetType() AccountType {
	return a.Type
}

// IsSuspended implements AccountInterface
func (a *Account) IsSuspended() bool {
	return a.SuspendedTime != nil
}

// NewAccount _
func NewAccount(addr *Address) (*Account, error) {
	if addr.Type != AccountTypePersonal {
		return nil, errors.New("invalid account type address")
	}
	return &Account{
		DOCTYPEID: addr.RawID,
		Address:   addr.String(),
		Token:     addr.Code,
		Type:      addr.Type,
	}, nil
}

// JointAccount _
type JointAccount struct {
	Account
	Holders stringset.Set `json:"holders"`
}

// NewJointAccount _
func NewJointAccount(addr *Address, holders stringset.Set) (*JointAccount, error) {
	if addr.Type != AccountTypeJoint {
		return nil, errors.New("invalid account type address")
	}
	return &JointAccount{
		Account: Account{
			DOCTYPEID: addr.RawID,
			Address:   addr.String(),
			Token:     addr.Code,
			Type:      addr.Type,
		},
		Holders: holders,
	}, nil
}

// AccountRM (Account Relation Meta)
type AccountRM struct {
	Address string      `json:"address"`
	Token   string      `json:"token"`
	Type    AccountType `json:"type"`
}

// AccountHolder represents an account-holder relationship (many-to-many)
type AccountHolder struct {
	DOCTYPEID   string     `json:"@account_holder"`
	Account     AccountRM  `json:"account"`
	CreatedTime *time.Time `json:"created_time,omitempty"`
}

// NewAccountHolder _
func NewAccountHolder(kid string, account AccountInterface) *AccountHolder {
	return &AccountHolder{
		DOCTYPEID: kid,
		Account: AccountRM{
			account.GetAddress(),
			account.GetToken(),
			account.GetType(),
		},
	}
}

// IsHeldBy _
// func (a *JointAccount) IsHeldBy(id string) bool {
// 	return a.Holders[id]
// }

// func (a *Account) IsHeldBy(kid string) bool {
// 	return a.DOCTYPEID == kid
// }
