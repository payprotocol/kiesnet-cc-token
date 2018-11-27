// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
)

// Address _
type Address struct {
	Code string
	Type AccountType
	Hash []byte
}

// NewAddress _
func NewAddress(tokenCode string, typeCode AccountType, id string) *Address {
	addr := &Address{}
	addr.Code = tokenCode
	addr.Type = typeCode

	idh, err := hex.DecodeString(id)
	if err != nil || len(idh) != 20 { // not kid
		idh = make([]byte, 20)
		sha3.ShakeSum256(idh, []byte(id))
	}

	// add checksum to hash
	buf := bytes.NewBuffer(idh)
	buf.Write(addr.Checksum(idh))
	addr.Hash = buf.Bytes()

	return addr
}

// ParseAddress parses address string and validates it
func ParseAddress(addr string) (*Address, error) {
	addr = strings.ToUpper(addr)
	l := len(addr)
	if l < 50 {
		return nil, InvalidAccountAddrError{"length"}
	}
	i := l - 50 // start index of hex

	idh, err := hex.DecodeString(addr[i:])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode address")
	}

	_addr := &Address{}
	_addr.Code = addr[0:i]
	_addr.Type = AccountType(idh[0])
	_addr.Hash = idh[1:]

	if err = _addr.Validate(); err != nil {
		return nil, err
	}
	return _addr, nil
}

// ID _
func (addr *Address) ID() string {
	return hex.EncodeToString(addr.Hash[:20])
}

// Checksum _
func (addr *Address) Checksum(hash []byte) []byte {
	buf := bytes.NewBuffer([]byte(addr.Code))
	buf.WriteByte(byte(addr.Type))
	buf.Write(hash)
	h := make([]byte, 4)
	sha3.ShakeSum256(h, buf.Bytes())
	return h
}

// Equal _
func (addr *Address) Equal(a *Address) bool {
	return addr.Code == a.Code && addr.Type == a.Type && bytes.Equal(addr.Hash, a.Hash)
}

// String _
func (addr *Address) String() string {
	// token code + [50 bytes upper-case hex]
	return fmt.Sprintf("%s%02X%X", addr.Code, byte(addr.Type), addr.Hash)
}

// Validate _
func (addr *Address) Validate() error {
	// token code
	if _, err := ValidateTokenCode(addr.Code); err != nil {
		return InvalidAccountAddrError{"token code"}
	}
	// account type
	if addr.Type <= AccountTypeUnknown || addr.Type > AccountTypeJoint {
		return InvalidAccountAddrError{"account type"}
	}
	// checksum
	checksum := addr.Checksum(addr.Hash[:20])
	if bytes.HasSuffix(addr.Hash, checksum) {
		return nil // valid
	}
	return InvalidAccountAddrError{"checksum"}
}
