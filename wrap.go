package main

import (
	"errors"
	"strings"
)

type Wrap struct{}

type WrapBridge struct {
	WPCIAddress string `json:"wpci_address"`
}

// NewWrapResult _
func NewWrapInfo(addr string) *WrapBridge {
	return &WrapBridge{
		WPCIAddress: addr,
	}
}

// ParseWrapAddress _
func ParseWrapAddress(code string, token Token) (*Address, error) {
	code = strings.ToUpper(code)
	switch code {
	case
		"WPCI":
		return ParseAddress(token.WrapBridge.WPCIAddress)
	}
	return nil, errors.New("there is no wrap address. check token code")
}
