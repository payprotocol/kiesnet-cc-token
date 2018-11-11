// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import "fmt"

// QueryPersonalAccountByIDAndTokenCode _
const QueryPersonalAccountByIDAndTokenCode = `{
	"selector": {
		"@account": "%s",
		"token": "%s",
		"type": 1
	},
	"limit": 1,
	"use_index": ["account", "personal"]
}`

// CreateQueryPersonalAccountByIDAndTokenCode _
func CreateQueryPersonalAccountByIDAndTokenCode(id, tokenCode string) string {
	return fmt.Sprintf(QueryPersonalAccountByIDAndTokenCode, id, tokenCode)
}

// QueryAccountHoldersByID _
const QueryAccountHoldersByID = `{
	"selector": {
		"@account_holder": "%s"
	},
	"sort": ["@account_holder", "account.token", "account.type"],
	"use_index": ["account", "account-holder"]
}`

// CreateQueryAccountHoldersByID _
func CreateQueryAccountHoldersByID(id string) string {
	return fmt.Sprintf(QueryAccountHoldersByID, id)
}

// QueryAccountHoldersByIDAndTokenCode _
const QueryAccountHoldersByIDAndTokenCode = `{
	"selector": {
		"@account_holder": "%s",
		"account.token": "%s"
	},
	"sort": ["@account_holder", "account.token", "account.type"],
	"use_index": ["account", "account-holder"]
}`

// CreateQueryAccountHoldersByIDAndTokenCode _
func CreateQueryAccountHoldersByIDAndTokenCode(id, tokenCode string) string {
	return fmt.Sprintf(QueryAccountHoldersByIDAndTokenCode, id, tokenCode)
}
