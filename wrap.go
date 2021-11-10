package main

import (
	"errors"
	"strings"

	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

type WrapType int8

const (
	TypeWrap WrapType = iota
	TypeUnWrap
)

type Wrap struct {
	DOCTYPEID   string       `json:"@wrap"`               //address
	Type        WrapType     `json:"type"`                //type
	Amount      Amount       `json:"amount"`              //must be positive
	Fee         Amount       `json:"fee,omitempty"`       //must be positive
	RID         string       `json:"rid"`                 //related id. user account id who wrap or unwrap token
	ExtID       string       `json:"ext_id"`              //external address
	ExtTxID     string       `json:"ext_tx_id,omitempty"` //used for unwrap
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
}

// NewWrap _
func NewWrap(id string, amount, fee Amount, rID, extID string, ts *txtime.Time) *Wrap {
	return &Wrap{
		DOCTYPEID:   id,
		Type:        TypeWrap,
		Amount:      amount,
		Fee:         fee,
		RID:         rID,
		ExtID:       extID,
		CreatedTime: ts,
	}
}

func NewUnWrap(id string, amount Amount, rID, extID, extTxID string, ts *txtime.Time) *Wrap {
	return &Wrap{
		DOCTYPEID:   id,
		Type:        TypeWrap,
		Amount:      amount,
		RID:         rID,
		ExtID:       extID,
		ExtTxID:     extTxID,
		CreatedTime: ts,
	}
}

type WrapInfo struct {
	WPCIAddress string `json:"wpci_address"`
}

// NewWrapResult _
func NewWrapInfo(addr string) *WrapInfo {
	return &WrapInfo{
		WPCIAddress: addr,
	}
}

// ParseWrapAddress _
func ParseWrapAddress(code string, token Token) (*Address, error) {
	code = strings.ToUpper(code)
	switch code {
	case
		"WPCI":
		return ParseAddress(token.WrapInfo.WPCIAddress)
	}
	return nil, errors.New("there is no wrap address. check token code")
}

// WrapResult _
type WrapResult struct {
	Wrap       *Wrap       `json:"wrap"`
	BalanceLog *BalanceLog `json:"balance_log"`
}

// NewWrapResult _
func NewWrapResult(wrap *Wrap, log *BalanceLog) *WrapResult {
	return &WrapResult{
		Wrap:       wrap,
		BalanceLog: log,
	}
}
