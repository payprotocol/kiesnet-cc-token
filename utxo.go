package main

import (
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// ChunkType _
type ChunkType int8

const (
	// ChunkTypeAccount _
	ChunkTypeAccount ChunkType = iota
	// ChunkTypeContract _
	ChunkTypeContract
)

// Chunk _
type Chunk struct {
	DOCTYPEID   string       `json:"@chunk"`  // id
	Account     string       `json:"account"` // account ID (address)
	RID         string       `json:"rid"`
	Amount      Amount       `json:"amount"`
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
}

// NewChunkType _
func NewChunkType(id string, owner Identifiable, rid Identifiable, amount Amount, cTime *txtime.Time) *Chunk {
	if nil == rid {
		return &Chunk{
			DOCTYPEID:   id,
			Account:     owner.GetID(),
			RID:         "",
			Amount:      amount,
			CreatedTime: cTime,
		}
	}
	return &Chunk{
		DOCTYPEID:   id,
		Account:     owner.GetID(),
		RID:         rid.GetID(),
		Amount:      amount,
		CreatedTime: cTime,
	}
}

// MergeResult _
type MergeResult struct {
	DOCTYPEID   string       `json:"@merge_result"`
	Start       string       `json:"start_key"`
	End         string       `json:"end_key"`
	Result      string       `json:"result_chunk"`
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
}

// NewMergeResultType _
func NewMergeResultType(id, mergedChunk, start, end string, cTime *txtime.Time) *MergeResult {
	return &MergeResult{
		DOCTYPEID:   id,
		Start:       start,
		End:         end,
		Result:      mergedChunk,
		CreatedTime: cTime,
	}
}
