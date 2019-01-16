package main

import (
	"github.com/key-inside/kiesnet-ccpkg/contract"
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
	DOCTYPEID   string       `json:"@pay_chunk"` // id
	Type        PayChunkType `json:"type"`
	Account     string       `json:"account"` // account ID (address)
	RID         string       `json:"rid"`     // relative ID - account or contract
	Amount      Amount       `json:"amount"`
	Memo        string       `json:"memo"`
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
}

// NewPayChunkType _
func NewPayChunkType(id string, owner Identifiable, rel Identifiable, amount Amount, memo string, cTime *txtime.Time) *PayChunk {
	ptype := PayChunkTypeAccount
	if _, ok := rel.(*contract.Contract); ok {
		ptype = PayChunkTypeContract
	}
	return &PayChunk{
		DOCTYPEID:   id,
		Type:        ptype,
		Account:     owner.GetID(),
		RID:         rel.GetID(),
		Amount:      amount,
		Memo:        memo,
		CreatedTime: cTime,
	}
}
