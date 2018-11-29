// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
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
	DOCTYPEID      string     `json:"@token"` // Code, validate:"required,min=3,max=6,alphanum"
	Decimal        int        `json:"decimal"`
	MaxSupply      Amount     `json:"max_supply"`
	Supply         Amount     `json:"supply"`
	GenesisAccount string     `json:"genesis_account"`
	CreatedTime    *time.Time `json:"created_time,omitempty"`
	UpdatedTime    *time.Time `json:"updated_time,omitempty"`
}

// TokenMeta is meta information struct defined by each token instance chaincode.
type TokenMeta struct {
	_map map[string]interface{}
}

// QueryTokenMeta retrieves TokenMeta of each token instance chaincode.
func QueryTokenMeta(stub shim.ChaincodeStubInterface, chaincodeName string) (*TokenMeta, error) {
	args := [][]byte{[]byte("meta")}
	res := stub.InvokeChaincode(chaincodeName, args, "")
	if res.GetStatus() == 200 {
		m := make(map[string]interface{})
		err := json.Unmarshal(res.GetPayload(), &m)
		if err != nil {
			return nil, err
		}
		tokenMeta := &TokenMeta{_map: m}
		return tokenMeta, nil
	}
	return nil, errors.New(res.GetMessage())
}
