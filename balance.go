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
	DOCTYPEID   string         `json:"@balance_log"` // address
	Type        BalanceLogType `json:"type"`
	RID         string         `json:"rid"` // relative ID
	Diff        Amount         `json:"diff"`
	Amount      Amount         `json:"amount"`
	Memo        string         `json:"memo"`
	CreatedTime *time.Time     `json:"created_time,omitempty"`
}

// NewBalanceSupplyLog _
func NewBalanceSupplyLog(bal *Balance, diff Amount) *BalanceLog {
	if diff.Sign() < 0 { // burn
		return &BalanceLog{
			DOCTYPEID: bal.DOCTYPEID,
			Type:      BalanceLogTypeBurn,
			Diff:      diff,
			Amount:    bal.Amount,
		}
	} // else  mint
	return &BalanceLog{
		DOCTYPEID: bal.DOCTYPEID,
		Type:      BalanceLogTypeMint,
		Diff:      diff,
		Amount:    bal.Amount,
	}
}

// NewBalanceTransferLog _
func NewBalanceTransferLog(sender, receiver *Balance, diff Amount, memo string) *BalanceLog {
	if diff.Sign() < 0 { // sender log
		return &BalanceLog{
			DOCTYPEID: sender.DOCTYPEID,
			Type:      BalanceLogTypeSend,
			RID:       receiver.DOCTYPEID,
			Diff:      diff,
			Amount:    sender.Amount,
			Memo:      memo,
		}
	} // else receiver log
	return &BalanceLog{
		DOCTYPEID: receiver.DOCTYPEID,
		Type:      BalanceLogTypeReceive,
		RID:       sender.DOCTYPEID,
		Diff:      diff,
		Amount:    receiver.Amount,
		Memo:      memo,
	}
}

// NewBalanceWithdrawLog _
func NewBalanceWithdrawLog(bal *Balance, pb *PendingBalance) *BalanceLog {
	return &BalanceLog{
		DOCTYPEID: bal.DOCTYPEID,
		Type:      BalanceLogTypeWithdraw,
		RID:       pb.RID,
		Diff:      pb.Amount,
		Amount:    bal.Amount,
		Memo:      pb.Memo,
	}
}

// PendingBalance _
type PendingBalance struct {
	DOCTYPEID   string     `json:"@pending_balance"` // id
	Account     string     `json:"account"`          // address
	RID         string     `json:"rid"`              // relative ID
	Amount      Amount     `json:"amount"`
	Memo        string     `json:"memo"`
	CreatedTime *time.Time `json:"created_time,omitempty"`
	PendingTime *time.Time `json:"pending_time,omitempty"`
}
