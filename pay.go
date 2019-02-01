// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// Pay _
type Pay struct {
	DOCTYPEID   string       `json:"@pay"`                   //address
	PayID       string       `json:"pay_id"`                 //used for the external client to pass the pay id for refund
	Amount      Amount       `json:"amount"`                 //can be positive(pay) or negative(refund)
	TotalRefund Amount       `json:"total_refund,omitempty"` //Total refund value
	RID         string       `json:"rid"`                    //related id. user who pays to the merchant or receives refund from the merchant.
	ParentKey   string       `json:"parent_key,omitempty"`   //parent key. this value exists only when the pay type is refund(negative amount)
	Memo        string       `json:"memo"`
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
}

// NewPay _
func NewPay(id, payid string, amount Amount, rid, pkey, memo string, ts *txtime.Time) *Pay {
	return &Pay{
		DOCTYPEID:   id,
		PayID:       payid,
		Amount:      amount,
		RID:         rid,
		ParentKey:   pkey,
		Memo:        memo,
		CreatedTime: ts,
	}
}

// PaySum _
type PaySum struct {
	Sum     *Amount `json:"sum"`
	Start   string  `json:"start_key"`
	End     string  `json:"end_key"`
	HasMore bool    `json:"has_more"`
}
