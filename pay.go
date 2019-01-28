// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// ???: 전체적인 naming 문제

// PayPayType _
type PayPayType int8

// ???: useless
const (
	// PayPayTypeAccount _
	PayPayTypeAccount PayPayType = iota
	// PayPayTypeContract _
	PayPayTypeContract
)

// Pay _
type Pay struct {
	DOCTYPEID   string       `json:"@pay"`           // pay id
	Owner       string       `json:"owner"`          // owner id
	Amount      Amount       `json:"amount"`         //can be positive(pay) or negative(refund)
	RID         string       `json:"rid"`            //related id. user who pays to the merchant or receives refund from the merchant.
	PKey        string       `json:"pkey,omitempty"` //parent key. this value exists only when the pay type is refund(negative amount)
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
}

// NewPayType _
func NewPayType(key, owner string, amount Amount, rid, pkey string, ts *txtime.Time) *Pay {
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
