// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"time"
)

// Balance _
type Balance struct {
	DOCTYPEID   string     `json:"@balance"`
	Amount      Amount     `json:"amount"`
	CreatedTime *time.Time `json:"created_time,omitempty"`
	UpdatedTime *time.Time `json:"updated_time,omitempty"`
}

// // BalanceLogRelType _
// type BalanceLogRelType int8

// const (
// 	BalanceLogRelTypeTransfer BalanceLogRelType = iota
// )

// // BalanceLogType _
// type BalanceLogType int8

// const (
// 	// BalanceLogTypeMint _
// 	BalanceLogTypeMint BalanceLogType = iota
// 	// BalanceLogTypeBurn _
// 	BalanceLogTypeBurn
// 	// BalanceLogTypeDeposit _
// 	BalanceLogTypeDeposit
// 	BalanceLogTypeWithdraw

// 	BalanceLogTypeWithdrawFromContract
// )

// type BalanceLogRel struct {
// 	Type BalanceLogType
// }

// // type TransferRM

// // BalanceLog _
// type BalanceLog struct {
// 	DOCTYPEID   string     `json:"@balance_log"`
// 	Diff       Amount     `json:"diff"`
// 	Amount      Amount     `json:"amount"`
// 	Memo        string     `json:"memo"`
// 	CreatedTime *time.Time `json:"created_time,omitempty"`
// }
