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
	DOCTYPEID   string       `json:"@chunk"` // id
	Amount      Amount       `json:"amount"` //can be positive(pay) or negative(refund)
	RID         string       `json:"rid"`    //related id. user who pays to the merchant or receives refund from the merchant.
	PKey        string       `json:"pkey"`   //parent key. this value exists only when the chunk type is refund(negative amount)
	CreatedTime *txtime.Time `json:"created_time,omitempty"`
}

// PruneLog _
type PruneLog struct {
	DOCTYPEID        string       `json:"@prune_log"`
	PruneFromAddress string       `json:"from_id"`        //prune start chunk
	PruneToAddress   string       `json:"to_id"`          //prune end chunk
	ReceiverAddress  string       `json:"r_id"`           //master account
	NextChunkKey     string       `json:"next_chunk_key"` //the key to indicate the remaining chunks in the given time period
	Sum              int64        `json:"sum"`            //sum result from the pruning
	CreatedTime      *txtime.Time `json:"created_time,omitempty"`
}

// NewPayChunkType _
func NewPayChunkType(id string, amount Amount, rid, pkey string, ts *txtime.Time) *PayChunk {
	return &PayChunk{
		DOCTYPEID:   id,
		Amount:      amount,
		RID:         rid,
		PKey:        pkey,
		CreatedTime: ts,
	}
}

// NewPruneLog _
func NewPruneLog(id, fromAddress, toAddress, receiverAddress, nextChunkKey string, sum int64) *PruneLog {
	return &PruneLog{
		DOCTYPEID:        id,
		PruneFromAddress: fromAddress,
		PruneToAddress:   toAddress,
		ReceiverAddress:  receiverAddress,
		NextChunkKey:     nextChunkKey,
		Sum:              sum,
	}
}
