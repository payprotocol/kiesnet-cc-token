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
	DOCTYPEID   string       `json:"@chunk"`  // id - CHK_Nano
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
	DOCTYPEID   string       `json:"@merge_result"` // id - MERG_VBRacct_End-chunk-id
	Account     string       `json:"account"`
	Start       *Chunk       `json:"start"`
	End         *Chunk       `json:"end"`
	Amount      Amount       `json:"amount"`
	RID         string       `json:"toid"`
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
}

// NewMergeResultType _
func NewMergeResultType(id, toid string, owner Identifiable, start, end *Chunk, cTime *txtime.Time, amount Amount) *MergeResult {
	return &MergeResult{
		DOCTYPEID:   id,
		Account:     owner.GetID(),
		Start:       start,
		End:         end,
		Amount:      amount,
		RID:         toid,
		CreatedTime: cTime,
	}
}
