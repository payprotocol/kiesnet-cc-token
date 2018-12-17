// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"github.com/key-inside/kiesnet-ccpkg/contract"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// Balance _
type Balance struct {
	DOCTYPEID   string       `json:"@balance"` // address
	Amount      Amount       `json:"amount"`
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
	UpdatedTime *txtime.Time `json:"updated_time,omitempty"`
}

// GetID implements Identifiable
func (b *Balance) GetID() string {
	return b.DOCTYPEID
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
	CreatedTime *txtime.Time   `json:"created_time,omitempty"`
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

// NewBalanceDepositLog _
func NewBalanceDepositLog(bal *Balance, pb *PendingBalance) *BalanceLog {
	diff := pb.Amount.Copy().Neg()
	return &BalanceLog{
		DOCTYPEID: bal.DOCTYPEID,
		Type:      BalanceLogTypeDeposit,
		RID:       pb.RID,
		Diff:      *diff,
		Amount:    bal.Amount,
		Memo:      pb.Memo,
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

// PendingBalanceType _
type PendingBalanceType int8

const (
	// PendingBalanceTypeAccount _
	PendingBalanceTypeAccount PendingBalanceType = iota
	// PendingBalanceTypeContract _
	PendingBalanceTypeContract
)

// PendingBalance _
type PendingBalance struct {
	DOCTYPEID   string             `json:"@pending_balance"` // id
	Type        PendingBalanceType `json:"type"`
	Account     string             `json:"account"` // account ID (address)
	RID         string             `json:"rid"`     // relative ID - account or contract
	Amount      Amount             `json:"amount"`
	Memo        string             `json:"memo"`
	CreatedTime *txtime.Time       `json:"created_time,omitempty"`
	PendingTime *txtime.Time       `json:"pending_time,omitempty"`
}

// NewPendingBalance _
func NewPendingBalance(id string, owner Identifiable, rel Identifiable, amount Amount, memo string, pTime *txtime.Time) *PendingBalance {
	ptype := PendingBalanceTypeAccount
	if _, ok := rel.(*contract.Contract); ok {
		ptype = PendingBalanceTypeContract
	}
	return &PendingBalance{
		DOCTYPEID:   id,
		Type:        ptype,
		Account:     owner.GetID(),
		RID:         rel.GetID(),
		Amount:      amount,
		Memo:        memo,
		PendingTime: pTime,
	}
}
