// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"math/big"

	"github.com/pkg/errors"
)

// Amount _
type Amount struct {
	big.Int
}

// NewAmount _
func NewAmount(val string) (*Amount, error) {
	a := &Amount{}
	if len(val) > 0 {
		if _, ok := a.SetString(val, 10); !ok {
			return nil, errors.New("invalid amount value: must be integer")
		}
	}
	return a, nil
}

// Add override
func (a *Amount) Add(x *Amount) *Amount {
	a.Int.Add(&a.Int, &x.Int)
	return a
}

// Cmp override
func (a *Amount) Cmp(x *Amount) int {
	return a.Int.Cmp(&x.Int)
}

// Neg override
func (a *Amount) Neg() *Amount {
	a.Int.Neg(&a.Int)
	return a
}

// MarshalJSON override
func (a *Amount) MarshalJSON() ([]byte, error) {
	// TODO: why ????
	return []byte(a.String()), nil
}

// // UnmarshalJSON override
// func (a *Amount) UnmarshalJSON(text []byte) error {
// 	return a.Int.UnmarshalJSON(text)
// }
