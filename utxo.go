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

// MergeHistory _
type MergeHistory struct {
	DOCTYPEID      string       `json:"@merge_history"`
	FromAddress    string       `json:"from_key"`
	ToAddress      string       `json:"to_key"`
	MergedChunkKey string       `json:"merged_chunk_key"`
	NextChunkKey   string       `json:"next_chunk_key"`
	Sum            int64        `json:"sum"`
	CreatedTime    *txtime.Time `json:"created_time,omitempty"`
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

// NewMergeHistory _
func NewMergeHistory(id, fromAddress, toAddress, mergedChunkKey, nextChunkKey string, sum int64) *MergeHistory {
	return &MergeHistory{
		DOCTYPEID:      id,
		FromAddress:    fromAddress,
		ToAddress:      toAddress,
		MergedChunkKey: mergedChunkKey,
		NextChunkKey:   nextChunkKey,
		Sum:            sum,
	}
}
