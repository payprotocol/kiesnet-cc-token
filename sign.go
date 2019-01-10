// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import "github.com/key-inside/kiesnet-ccpkg/txtime"

// Sign represents signer's action. (approve or disapprove)
type Sign struct {
	Signer          string       `json:"signer"`
	ApprovedTime    *txtime.Time `json:"approved_time,omitempty"`
	DisapprovedTime *txtime.Time `json:"disapproved_time,omitempty"`
}
