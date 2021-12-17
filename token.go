// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"regexp"
	"strings"

	"github.com/key-inside/kiesnet-ccpkg/contract"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
)

var _validateTokenCode = regexp.MustCompile(`^[A-Z0-9]{3,6}$`).MatchString

// ValidateTokenCode validates a code and returns an uppercased code
func ValidateTokenCode(code string) (string, error) {
	code = strings.ToUpper(code)
	if !_validateTokenCode(code) {
		return "", errors.New("token code must be 3~6 length alphanum")
	}
	return code, nil
}

// Token _
type Token struct {
	DOCTYPEID       string                 `json:"@token"` // Code, validate:"required,min=3,max=6,alphanum"
	Decimal         int                    `json:"decimal"`
	MaxSupply       Amount                 `json:"max_supply"`
	Supply          Amount                 `json:"supply"`
	LastPrunedFeeID string                 `json:"last_pruned_fee_id,omitempty"`
	GenesisAccount  string                 `json:"genesis_account"`
	FeePolicy       *FeePolicy             `json:"fee_policy,omitempty"`  // FeePolicy is nil if and only if knt fee is never yet imported. Once knt is initiated/upgraded with fee, it wil always exists.
	WrapBridge      map[string]*WrapPolicy `json:"wrap_bridge,omitempty"` // map of extCode and wrap policy(wrap address, ext chain, fee)
	CreatedTime     *txtime.Time           `json:"created_time,omitempty"`
	UpdatedTime     *txtime.Time           `json:"updated_time,omitempty"`
}

func (t *Token) getWrapPolicy(extCode string) (*WrapPolicy, error) {
	if t.WrapBridge == nil {
		return nil, errors.New("wrap_bridge is not installed")
	}
	policy, ok := t.WrapBridge[strings.ToUpper(extCode)]
	if !ok {
		return nil, errors.New("wrap_bridge has no policy of given ext_code")
	}
	return policy, nil
}

// GetWrapAddress returns wrap address of given extCode.
func (t *Token) GetWrapAddress(extCode string) (*Address, error) {
	policy, err := t.getWrapPolicy(extCode)
	if err != nil {
		return nil, err
	}
	return ParseAddress(policy.WrapAddress)
}

// GetWrapFee returns wrap fee amount of given extCode.
func (t *Token) GetWrapFee(extCode string) (*Amount, error) {
	policy, err := t.getWrapPolicy(extCode)
	if err != nil {
		return nil, err
	}
	return NewAmount(policy.Fee)
}

// TokenResult is response payload of token/burn and token/mint.
type TokenResult struct {
	Token      *Token             `json:"token,omitempty"`
	BalanceLog *BalanceLog        `json:"balance_log,omitempty"`
	Contract   *contract.Contract `json:"contract,omitempty"`
}
