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

// Chunk _
type Chunk struct {
	DOCTYPEID   string       `json:"@chunk"`         // chunk id
	Owner       string       `json:"owner"`          // owner id
	Amount      Amount       `json:"amount"`         //can be positive(pay) or negative(refund)
	RID         string       `json:"rid"`            //related id. user who pays to the merchant or receives refund from the merchant.
	PKey        string       `json:"pkey,omitempty"` //parent key. this value exists only when the chunk type is refund(negative amount)
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
}

// NewChunkType _
func NewChunkType(key, owner string, amount Amount, rid, pkey string, ts *txtime.Time) *Chunk {
	return &Chunk{
		DOCTYPEID:   key,
		Owner:       owner,
		Amount:      amount,
		RID:         rid,
		PKey:        pkey,
		CreatedTime: ts,
	}
}

// ChunkSum _
type ChunkSum struct {
	Sum  *Amount `json:"sum"`
	End  string  `json:"end_key"`
	Next string  `json:"next_key"`
}
