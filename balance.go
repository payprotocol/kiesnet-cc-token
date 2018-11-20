// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import "time"

// Balance _
type Balance struct {
	DOCTYPEID   string     `json:"@balance"` // address
	Amount      Amount     `json:"amount"`
	CreatedTime *time.Time `json:"created_time,omitempty"`
	UpdatedTime *time.Time `json:"updated_time,omitempty"`
}

// BalanceLogType _
type BalanceLogType int8

const (
	// BalanceLogTypeMint _
	BalanceLogTypeMint BalanceLogType = iota
	// BalanceLogTypeBurn _
	BalanceLogTypeBurn
	// BalanceLogTypeSend _
	BalanceLogTypeSend
	// BalanceLogTypeReceive _
	BalanceLogTypeReceive
	// BalanceLogTypeDeposit deposit balance to contract
	BalanceLogTypeDeposit
	// BalanceLogTypeWithdraw withdraw balance from contract
	BalanceLogTypeWithdraw
)

// BalanceLog _
type BalanceLog struct {
	DOCTYPEID   string         `json:"@balance_log"`
	Type        BalanceLogType `json:"type"`
	RID         string         `json:"rid"` // relative ID
	Diff        Amount         `json:"diff"`
	Amount      Amount         `json:"amount"`
	Memo        string         `json:"memo"`
	CreatedTime *time.Time     `json:"created_time,omitempty"`
}

// PendingBalance _
type PendingBalance struct {
	DOCTYPEID   string     `json:"@pending_balance"`
	RID         string     `json:"rid"`
	Amount      Amount     `json:"amount"`
	Memo        string     `json:"memo"`
	CreatedTime *time.Time `json:"created_time,omitempty"`
	PendingTime *time.Time `json:"pending_time,omitempty"`
}
