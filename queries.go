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

// QueryContractsByID _
const QueryContractsByID = `{
	"selector": {
		"@contract": "%s"
	},
	"use_index": ["contract", "id"]
}`

// CreateQueryContractsByID _
func CreateQueryContractsByID(id string) string {
	return fmt.Sprintf(QueryContractsByID, id)
}

// QueryContractsBySigner _
const QueryContractsBySigner = `{
	"selector": {
		"@contract": {
			"$exists": true
		},
		"sign.signer": "%s"
	},
	"sort": [{"sign.signer": "desc"}, {"created_time": "desc"}],
	"use_index": ["contract", "created-time"]
}`

// CreateQueryContractsBySigner _
func CreateQueryContractsBySigner(kid string) string {
	return fmt.Sprintf(QueryContractsBySigner, kid)
}

// QueryFinishedContractsBySigner _
const QueryFinishedContractsBySigner = `{
	"selector": {
		"@contract": {
			"$exists": true
		},
		"sign.signer": "%s",
		"finished_time": {
			"$lte": "%s"
		}
	},
	"sort": [{"sign.signer": "desc"}, {"finished_time": "desc"}],
	"use_index": ["contract", "finished-time"]
}`

// CreateQueryFinishedContractsBySigner _
func CreateQueryFinishedContractsBySigner(kid string, ts *txtime.Time) string {
	return fmt.Sprintf(QueryFinishedContractsBySigner, kid, ts.String())
}

// QueryUnfinishedContractsBySigner _
const QueryUnfinishedContractsBySigner = `{
	"selector": {
		"@contract": {
			"$exists": true
		},
		"sign.signer": "%s",
		"finished_time": {
			"$gt": "%s"
		}
	},
	"sort": ["sign.signer", "finished_time"],
	"use_index": ["contract", "finished-time"]
}`

// CreateQueryUnfinishedContractsBySigner _
func CreateQueryUnfinishedContractsBySigner(kid string, ts *txtime.Time) string {
	return fmt.Sprintf(QueryUnfinishedContractsBySigner, kid, ts.String())
}

// QueryApprovedContractsBySigner - unfinished, approved
const QueryApprovedContractsBySigner = `{
	"selector": {
		"$and": [
			{
				"@contract": {
					"$exists": true
				}
			},
			{
				"sign.approved_time": {
					"$exists": true
				}
			},
			{
				"executed_time": {
					"$exists": false
				}
			},
			{
				"canceled_time": {
					"$exists": false
				}
			}
		],
		"sign.signer": "%s",
		"expiry_time": {
			"$gt": "%s"
		}
	},
	"sort": ["sign.signer", "expiry_time"],
	"use_index": ["contract", "approved-expiry-time"]
}`

// CreateQueryApprovedContractsBySigner _
func CreateQueryApprovedContractsBySigner(kid string, ts *txtime.Time) string {
	return fmt.Sprintf(QueryApprovedContractsBySigner, kid, ts.String())
}

// QueryUnsignedContractsBySigner - unfinished, unsigned
const QueryUnsignedContractsBySigner = `{
	"selector": {
		"$and": [
			{
				"@contract": {
					"$exists": true
				}
			},
			{
				"sign.approved_time": {
					"$exists": false
				}
			},
			{
				"sign.disapproved_time": {
					"$exists": false
				}
			},
			{
				"executed_time": {
					"$exists": false
				}
			},
			{
				"canceled_time": {
					"$exists": false
				}
			}
		],
		"sign.signer": "%s",
		"expiry_time": {
			"$gt": "%s"
		}
	},
	"sort": ["sign.signer", "expiry_time"],
	"use_index": ["contract", "unsigned-expiry-time"]
}`

// CreateQueryUnsignedContractsBySigner _
func CreateQueryUnsignedContractsBySigner(kid string, ts *txtime.Time) string {
	return fmt.Sprintf(QueryUnsignedContractsBySigner, kid, ts.String())
}
