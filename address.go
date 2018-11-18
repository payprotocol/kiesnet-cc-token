// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
)

// Address _
type Address struct {
	Code  string
	Type  AccountType
	Hash  []byte
	RawID string
}

// NewAddress _
func NewAddress(tokenCode string, typeCode AccountType, id string) *Address {
	addr := &Address{}
	addr.Code = tokenCode
	addr.Type = typeCode
	addr.Hash = addr.CreateHash(id)
	addr.RawID = id
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

// Checksum _
func (addr *Address) Checksum(hash []byte) []byte {
	buf := bytes.NewBuffer([]byte(addr.Code))
	buf.WriteByte(byte(addr.Type))
	buf.Write(hash)
	h := make([]byte, 32)
	sha3.ShakeSum256(h, buf.Bytes())
	return h[:4]
}

// CreateHash _
func (addr *Address) CreateHash(id string) []byte {
	rmd := ripemd160.New()
	rmd.Reset()
	rmd.Write([]byte(addr.Code))
	rmd.Write([]byte{byte(addr.Type)})
	rmd.Write([]byte(id))
	rh := rmd.Sum(nil)

	buf := bytes.NewBuffer(rh)
	buf.Write(addr.Checksum(rh))
	return buf.Bytes() // 28 bytes
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
