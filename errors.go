// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import "fmt"

// NotIssuedTokenError _
type NotIssuedTokenError struct {
	code string
}

// Error implements error interface
func (e NotIssuedTokenError) Error() string {
	return fmt.Sprintf("the token '%s' is not issued", e.code)
}

// InvalidAccessError _
type InvalidAccessError struct{}

// Error implements error interface
func (e InvalidAccessError) Error() string {
	return "invalid access"
}

// InvalidAccountAddrError _
type InvalidAccountAddrError struct {
	msg string
}

// Error implements error interface
func (e InvalidAccountAddrError) Error() string {
	if len(e.msg) > 0 {
		return fmt.Sprintf("invalid account address: %s", e.msg)
	}
	return "invalid account address"
}

// ExistedAccountError _
type ExistedAccountError struct {
	addr string
}

// Error implements error interface
func (e ExistedAccountError) Error() string {
	return fmt.Sprintf("the account '%s' is already exists", e.addr)
}

// NotExistedAccountError _
type NotExistedAccountError struct {
	addr string
}

// Error implements error interface
func (e NotExistedAccountError) Error() string {
	if len(e.addr) > 0 {
		return fmt.Sprintf("the account '%s' is not exists", e.addr)
	}
	return "the account is not exists"
}
