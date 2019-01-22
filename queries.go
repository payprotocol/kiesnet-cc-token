// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"fmt"

	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

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

// QueryBalanceLogsByIDAndTimes _
const QueryBalanceLogsByIDAndTimes = `{
	"selector": {
		"@balance_log": "%s",
		"$and": [
            {
                "created_time": {
                    "$gte": "%s"
                }
            },
            {
                "created_time": {
                    "$lt": "%s"
                }
            }
        ]
	},
	"sort": [{"@balance_log": "desc"}, {"created_time": "desc"}],
	"use_index": ["balance", "logs"]
}`

// CreateQueryBalanceLogsByIDAndTimes _
func CreateQueryBalanceLogsByIDAndTimes(id string, stime, etime *txtime.Time) string {
	if nil == stime {
		stime = txtime.Unix(0, 0)
	}
	if nil == etime {
		etime = txtime.Unix(253402300799, 999999999) // 9999-12-31 23:59:59.999999999
	}
	return fmt.Sprintf(QueryBalanceLogsByIDAndTimes, id, stime.String(), etime.String())
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

//QueryUtxoChunks _
const QueryUtxoChunks = `{
	"selector":{		
		"@chunk": "%s",
		"$and":[
			{
				"created_time":{
					"$gt": "%s"
				}
			},{
				"created_time":{
					"$lte": "%s"
				}

			}
		] 
	},
	"fields":[
		"@chunk",
		"amount",
		"created_time"
	]
}`

//LatestPruneLog _
const LatestPruneLog = `{
	"selector": {
	   "@prune_log": "%s"
	},
	"sort": [
	   {
		  "created_time": "desc"
	   }
	],
	"use_index": [
	   "prune",
	   "log"
	],
	"limit": 1
 }`

//RefundChunks _
const RefundChunks = `{
	"selector": {
	   "@chunk": "%s",
	   "pkey": "%s"
	}
 }`

// CreateQueryUtxoChunks _
func CreateQueryUtxoChunks(id string, stime, etime *txtime.Time) string {
	return fmt.Sprintf(QueryUtxoChunks, id, stime, etime)
}

// CreateQueryLatestPruneLog _
func CreateQueryLatestPruneLog(id string) string {
	return fmt.Sprintf(LatestPruneLog, id)
}

// CreateQueryRefundChunks _
func CreateQueryRefundChunks(id, pkey string) string {
	return fmt.Sprintf(RefundChunks, id, pkey)
}
