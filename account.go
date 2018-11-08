// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"time"

	"github.com/key-inside/kiesnet-ccpkg/stringset"
)

// AccountInterface _
type AccountInterface interface {
	GetAddress() string
	GetID() string
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
func (a Account) GetAddress() string {
	return a.Address
}

// GetID implements AccountInterface
func (a Account) GetID() string {
	return a.DOCTYPEID
}

// NewAccount _
func NewAccount(tokenCode, kid string) *Account {
	account := &Account{}
	account.DOCTYPEID = kid
	account.Token = tokenCode
	account.Type = AccountTypePersonal
	return account
}

// JointAccount _
type JointAccount struct {
	Account
	Holders stringset.Set `json:"holders"`
}

// NewJointAccount _
func NewJointAccount(tokenCode, id string, holders stringset.Set) *JointAccount {
	account := &JointAccount{}
	account.DOCTYPEID = id
	account.Token = tokenCode
	account.Type = AccountTypeJoint
	account.Holders = holders
	return account
}

// IsHeldBy _
// func (a *JointAccount) IsHeldBy(id string) bool {
// 	return a.Holders[id]
// }

// func (a *Account) IsHeldBy(kid string) bool {
// 	return a.DOCTYPEID == kid
// }
