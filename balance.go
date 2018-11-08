// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"math/big"
	"time"
)

// Balance _
type Balance struct {
	DOCTYPEID   string     `json:"@balance"` // account id
	Amount      big.Int    `json:"amount,string"`
	CreatedTime *time.Time `json:"created_time,omitempty"`
	UpdatedTime *time.Time `json:"updated_time,omitempty"`
	// Contract    *Contract  `json:"contract,omitempty"`
}

// // Contract _
// type Contract struct {
// 	ID            string     `josn:"id"`
// 	WithdrawAfter *time.Time `json:"withdraw_after,omitempty"`
// }
