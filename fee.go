// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import "github.com/key-inside/kiesnet-ccpkg/txtime"

// Fee is a transfer/pay fee utxo which will be pruned to genesis account
type Fee struct {
	DOCTYPEID   string       `json:"@fee,required"` // token code
	FeeID       string       `json:"fee_id"`        // unique sequential identifier (timestamp + txid)
	Account     string       `json:"account"`       // account address who payed fee
	Amount      Amount       `json:"amount"`
	CreatedTime *txtime.Time `json:"created_time"`
}

var _feePolicies = map[string]*FeePolicy{}

// FeePolicy _
type FeePolicy struct {
	TargetAddress string             `json:"target_address"`
	Rates         map[string]FeeRate `json:"rates"`
}

// FeeRate _
type FeeRate struct {
	Rate      string `json:"rate"`       // numeric string of positive decimal fraction
	MaxAmount int64  `json:"max_amount"` // 0 is unlimit
}

// FeeSum stands for amount&state of accumulated fee from Start to End
type FeeSum struct {
	Sum     *Amount `json:"sum"`
	Count   int     `json:"count"`
	Start   string  `json:"start_id"`
	End     string  `json:"end_id"`
	HasMore bool    `json:"has_more"`
}
