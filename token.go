// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"math/big"
	"regexp"
	"strings"
	"time"

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
	DOCTYPEID        string     `json:"@token"`
	Code             string     `json:"code" validate:"required,min=3,max=6,alphanum"`
	Decimal          int        `json:"decimal"`
	MaxSupply        big.Int    `json:"max_supply,string"`
	Supply           big.Int    `json:"supply,string"`
	GenesisAccountID string     `json:"genesis_account_id"`
	CreatedTime      *time.Time `json:"created_time,omitempty"`
	UpdatedTime      *time.Time `json:"updated_time,omitempty"`
}
