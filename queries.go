// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import "fmt"

// QueryBalanceLogsByID _
const QueryBalanceLogsByID = `{
	"selector": {
		"@balance_log": "%s"
	},
	"sort": [{"@balance_log": "desc"}, {"created_time": "desc"}],
	"use_index": ["balance", "logs"]
}`

// CreateQueryBalanceLogsByID _
func CreateQueryBalanceLogsByID(id string) string {
	return fmt.Sprintf(QueryBalanceLogsByID, id)
}

// QueryHoldersByID _
const QueryHoldersByID = `{
	"selector": {
		"@holder": "%s"
	},
	"sort": ["@holder", "token", "type"],
	"use_index": ["account", "holder"]
}`

// CreateQueryHoldersByID _
func CreateQueryHoldersByID(id string) string {
	return fmt.Sprintf(QueryHoldersByID, id)
}

// QueryHoldersByIDAndTokenCode _
const QueryHoldersByIDAndTokenCode = `{
	"selector": {
		"@holder": "%s",
		"token": "%s"
	},
	"sort": ["@holder", "token", "type"],
	"use_index": ["account", "holder"]
}`

// CreateQueryHoldersByIDAndTokenCode _
func CreateQueryHoldersByIDAndTokenCode(id, tokenCode string) string {
	return fmt.Sprintf(QueryHoldersByIDAndTokenCode, id, tokenCode)
}

// QueryPendingBalancesByAddress _
const QueryPendingBalancesByAddress = `{
	"selector": {
		"@pending_balance": {
			"$exists": true
		},
		"account": "%s"
	},
	"sort": [%s],
	"use_index": "pending-balance"
}`

// CreateQueryPendingBalancesByAddress _
func CreateQueryPendingBalancesByAddress(addr, sort string) string {
	var _sort string
	if "created_time" == sort {
		_sort = `{"account":"desc"},{"created_time":"desc"}`
	} else { // pending_time
		_sort = `"account","pending_time"`
	}
	return fmt.Sprintf(QueryPendingBalancesByAddress, addr, _sort)
}
