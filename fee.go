// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import "github.com/key-inside/kiesnet-ccpkg/txtime"

// Fee is a transfer/pay fee utxo which will be pruned to genesis account
type Fee struct {
	DOCTYPEID   string       `json:"@fee"`    // genesis account address
	FeeID       string       `json:"fee_id"`  // unique sequential identifier (timestamp + txid)
	Account     string       `json:"account"` // account address who payed fee
	Amount      Amount       `json:"amount"`
	CreatedTime *txtime.Time `json:"created_time"`
}

var _feePolicies = map[string]*FeePolicy{}

// FeePolicy _
type FeePolicy struct {
	TargetAddress string
	Rates         map[string]FeeRate
}

// FeeRate _
type FeeRate struct {
	Rate      float32
	MaxAmount int64 // 0 is unlimit
}
