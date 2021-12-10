package main

import (
	"errors"
	"regexp"
	"strings"

	"golang.org/x/crypto/sha3"
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

// NormalizeExtAddress _
func NormalizeExtAddress(addr string) (string, error) {
	err := validate(addr)
	if err != nil {
		return "", err
	}
	return normalize(addr), nil
}

var _addressFormat = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
var _lowercaseHex = regexp.MustCompile(`^[0-9a-f]{40}$`)
var _uppercaseHex = regexp.MustCompile(`^[0-9A-F]{40}$`)

// validate _
func validate(addr string) error {
	if !_addressFormat.MatchString(addr) {
		return errors.New("address format error")
	}
	unprefix := addr[2:]
	if _lowercaseHex.MatchString(unprefix) || _uppercaseHex.MatchString(unprefix) {
		return nil
	}
	if mixedcase(unprefix) != unprefix { // checksum
		return errors.New("address checksum error")
	}
	return nil
}

// checksum (EIP-55)
func mixedcase(unprefix string) string {
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

// normalize _
func normalize(addr string) string {
	return strings.ToLower(addr)
}
