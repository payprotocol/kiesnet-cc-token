// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// ???: 전체적인 naming 문제

// Pay _
type Pay struct {
	DOCTYPEID   string       `json:"@pay"`                   // pay id
	Account     string       `json:"account"`                // owner id
	Amount      Amount       `json:"amount"`                 //can be positive(pay) or negative(refund)
	TotalRefund Amount       `json:"total_refund,omitempty"` // Total refund value
	RID         string       `json:"rid"`                    //related id. user who pays to the merchant or receives refund from the merchant.
	ParentKey   string       `json:"parent_key,omitempty"`   //parent key. this value exists only when the pay type is refund(negative amount)
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
}

// NewPayType _
func NewPay(key, account string, amount Amount, rid, pkey string, ts *txtime.Time) *Pay {
	return &Pay{
		DOCTYPEID:   key,
		Owner:       owner,
		Amount:      amount,
		RID:         rid,
		PKey:        pkey,
		CreatedTime: ts,
	}
}

// PaySum _
type PaySum struct {
	Sum  *Amount `json:"sum"`
	End  string  `json:"end_key"`
	Next string  `json:"next_key"`
}
