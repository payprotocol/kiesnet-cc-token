package main

import (
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// PayChunkType _
type PayChunkType int8

const (
	// PayChunkTypeAccount _
	PayChunkTypeAccount PayChunkType = iota
	// PayChunkTypeContract _
	PayChunkTypeContract
)

// PayChunk _
type PayChunk struct {
	DOCTYPEID   string       `json:"@chunk"`  // id
	Account     string       `json:"account"` // account ID (address)
	Amount      Amount       `json:"amount"`
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
}

// NewPayChunkType _
func NewPayChunkType(id string, owner Identifiable, amount Amount, cTime *txtime.Time) *PayChunk {
	return &PayChunk{
		DOCTYPEID:   id,
		Account:     owner.GetID(),
		Amount:      amount,
		CreatedTime: cTime,
	}
}
