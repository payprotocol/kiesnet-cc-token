// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"fmt"

	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// ResponsibleError is the interface used to distinguish responsible errors
type ResponsibleError interface {
	IsReponsible() bool
}

// ResponsibleErrorImpl _
type ResponsibleErrorImpl struct{}

// IsReponsible _
func (e ResponsibleErrorImpl) IsReponsible() bool {
	return true
}

// NotIssuedTokenError _
type NotIssuedTokenError struct {
	ResponsibleErrorImpl
	code string
}

// Error implements error interface
func (e NotIssuedTokenError) Error() string {
	return fmt.Sprintf("the token '%s' is not issued", e.code)
}

// InvalidAccessError _
type InvalidAccessError struct {
	ResponsibleErrorImpl
}

// Error implements error interface
func (e InvalidAccessError) Error() string {
	return "invalid access"
}

// SupplyError _
type SupplyError struct {
	ResponsibleErrorImpl
	reason string
}

// Error implements error interface
func (e SupplyError) Error() string {
	return e.reason
}

// InvalidAccountAddrError _
type InvalidAccountAddrError struct {
	ResponsibleErrorImpl
	reason string
}

// Error implements error interface
func (e InvalidAccountAddrError) Error() string {
	if len(e.reason) > 0 {
		return fmt.Sprintf("invalid account address: %s", e.reason)
	}
	return "invalid account address"
}

// ExistedAccountError _
type ExistedAccountError struct {
	ResponsibleErrorImpl
	addr string
}

// Error implements error interface
func (e ExistedAccountError) Error() string {
	return fmt.Sprintf("the account '%s' is already exists", e.addr)
}

// NotExistedAccountError _
type NotExistedAccountError struct {
	ResponsibleErrorImpl
	addr string
}

// Error implements error interface
func (e NotExistedAccountError) Error() string {
	if len(e.addr) > 0 {
		return fmt.Sprintf("the account '%s' is not exists", e.addr)
	}
	return "the account is not exists"
}

// ExistUtxcoChunkError _
type ExistUtxcoChunkError struct {
	ResponsibleErrorImpl
	key string
}

// Error implements error interface
func (e ExistUtxcoChunkError) Error() string {
	return fmt.Sprintf("the chunk '%s' is already issued", e.key)
}

// NotExistUtxoChunksError error _
type NotExistUtxoChunksError struct {
	ResponsibleErrorImpl
	stime *txtime.Time
	etime *txtime.Time
}

// Error implements error interface
func (e NotExistUtxoChunksError) Error() string {
	return fmt.Sprintf("no chunks between %s and %s", e.stime.UTC(), e.etime.UTC())
}
