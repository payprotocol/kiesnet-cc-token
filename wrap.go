package main

import (
	"errors"
	"regexp"
	"strings"

	"golang.org/x/crypto/sha3"
)

// WrapPolicy _
type WrapPolicy struct {
	WrapAddress string `json:"wrap_address"`
	ExtChain    string `json:"ext_chain"`
}

// NewWrapBridge _
func NewWrapBridge(data map[string]interface{}) (map[string]*WrapPolicy, error) {
	wb := make(map[string]*WrapPolicy)
	for extCode, rawPolicy := range data {
		values := strings.Split(rawPolicy.(string), ";")
		if len(values) == 2 {
			wb[strings.ToUpper(extCode)] = &WrapPolicy{
				WrapAddress: values[0],
				ExtChain:    values[1],
			}
		} else {
			return nil, errors.New("failed to parse wrap bridge")
		}
	}
	return wb, nil
}

// Wrap _
type Wrap struct {
	DOCTYPEID    string `json:"@wrap"` // tx_id
	Address      string `json:"address"`
	Amount       Amount `json:"amount"`
	ExtCode      string `json:"ext_code"`                 // external token code
	ExtID        string `json:"ext_id"`                   // EOA
	CompleteTxID string `json:"complete_tx_id,omitempty"` // tx hash (internal or external)
}

// Unwrap _
type Unwrap struct {
	DOCTYPEID    string `json:"@unwrap"`                  // external tx_id
	CompleteTxID string `json:"complete_tx_id,omitempty"` // internal tx hash
}

// NormalizeExtAddress _
func NormalizeExtAddress(addr string) (string, error) {
	err := validateEOA(addr)
	if err != nil {
		return "", err
	}
	return strings.ToLower(addr), nil
}

var isValidEOAFormat = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`).MatchString

// validateEOA _
func validateEOA(addr string) error {
	if !isValidEOAFormat(addr) {
		return errors.New("address format error")
	}
	unprefix := addr[2:]
	if strings.ToLower(unprefix) == unprefix || strings.ToUpper(unprefix) == unprefix { // all lowercase or all uppercase: no checksum
		return nil
	}
	if eip55Checksum(unprefix) != unprefix {
		return errors.New("address checksum error")
	}
	return nil
}

func eip55Checksum(unprefix string) string {
	unchecked := strings.ToLower(unprefix)
	sha := sha3.NewLegacyKeccak256()
	sha.Write([]byte(unchecked))
	hash := sha.Sum(nil)
	mixed := []byte(unchecked)
	for i := 0; i < len(mixed); i++ {
		hashByte := hash[i/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}
		if mixed[i] > '9' && hashByte > 7 {
			mixed[i] -= 32
		}
	}
	return string(mixed)
}

// NormalizeExtTxID _
func NormalizeExtTxID(txid string) (string, error) {
	txid = strings.ToLower(txid)
	err := validateExtTxID(txid)
	if err != nil {
		return "", err
	}
	return txid, nil
}

var isValidExtTxIDFormat = regexp.MustCompile(`^0x[0-9a-f]{64}$`).MatchString

func validateExtTxID(txid string) error {
	if !isValidExtTxIDFormat(txid) {
		return errors.New("txid format error")
	}
	return nil
}
