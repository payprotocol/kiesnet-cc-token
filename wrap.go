package main

import (
	"errors"
	"strings"

	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

type Wrap struct {
	DOCTYPEID   string       `json:"@wrap"`  //address
	Code        string       `json:"code"`   //token code 3 capital letters like WPCI, PCI if unwrap must be PCI..
	Amount      Amount       `json:"amount"` //must be positive
	RID         string       `json:"rid"`    //related id. user account id who wrap or unwrap token
	ExtID       string       `json:"ext_id"` //external address
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
}

// NewWrap _
func NewWrap(id, code string, amount Amount, rID, extID string, ts *txtime.Time) *Wrap {
	return &Wrap{
		DOCTYPEID:   id,
		Code:        code,
		Amount:      amount,
		RID:         rID,
		ExtID:       extID,
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

// ValidateExtTokenCode _
func ValidateExtTokenCode(code string) (string, error) {
	code = strings.ToUpper(code)
	switch code {
	case
		"WPCI":
		return code, nil
	}
	return "", errors.New("this token is not available to wrap. check token code")
}
