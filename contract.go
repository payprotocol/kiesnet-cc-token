// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"

	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
)

// Contract represents the contract
type Contract struct {
	DOCTYPEID     string       `json:"@contract"`
	Creator       string       `json:"creator"`
	SignersCount  int          `json:"signers_count"`
	ApprovedCount int          `json:"approved_count"`
	Document      string       `json:"document"`
	CreatedTime   *txtime.Time `json:"created_time,omitempty"`
	UpdatedTime   *txtime.Time `json:"updated_time,omitempty"`
	ExpiryTime    *txtime.Time `json:"expiry_time,omitempty"`
	ExecutedTime  *txtime.Time `json:"executed_time,omitempty"`
	CanceledTime  *txtime.Time `json:"canceled_time,omitempty"`
	FinishedTime  *txtime.Time `json:"finished_time,omitempty"`
	Sign          *Sign        `json:"sign"`
}

// GetID implements Identifiable
func (c *Contract) GetID() string {
	return c.DOCTYPEID
}

// AssertSignable _
func (c *Contract) AssertSignable(t *txtime.Time) error {
	if c.ExecutedTime != nil {
		return errors.New("already executed")
	}
	if c.CanceledTime != nil {
		return errors.New("already canceled")
	}
	if c.ExpiryTime != nil && t != nil && t.Cmp(c.ExpiryTime) >= 0 {
		return errors.New("already expired")
	}
	if c.Sign.ApprovedTime != nil {
		return errors.New("already approved")
	}
	if c.Sign.DisapprovedTime != nil {
		return errors.New("already dispproved")
	}
	return nil
}

// MarshalPayload _
func (c *Contract) MarshalPayload() ([]byte, error) {
	return json.Marshal(c)
}
