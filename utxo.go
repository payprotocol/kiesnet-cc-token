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
	DOCTYPEID   string       `json:"@chunk"` // id
	Amount      Amount       `json:"amount"`
	RID         string       `json:"rid"`
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
}

// NewChunkType _
func NewChunkType(id string, owner Identifiable, amount Amount, cTime *txtime.Time) *Chunk {
	return &Chunk{
		DOCTYPEID:   id,
		Amount:      amount,
		CreatedTime: cTime,
	}
}

// ChunkSum _
type ChunkSum struct {
	Sum  *Amount `json:"sum"`
	End  string  `json:"end_key"`
	Next string  `json:"next_key"`
}
