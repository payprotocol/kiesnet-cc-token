package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
)

// UtxoChunksFetchSize _
const UtxoChunksFetchSize = 500

// UtxoStub _
type UtxoStub struct {
	stub shim.ChaincodeStubInterface
}

// ChunkQueryResult _
type ChunkQueryResult struct {
	ID          string `json:"@chunk"`
	Amount      string `json:"amount"`
	CreatedTime string `json:"created_time"`
}

// NewUtxoStub _
func NewUtxoStub(stub shim.ChaincodeStubInterface) *UtxoStub {
	return &UtxoStub{stub}
}

// CreateChunkKey _
func (ub *UtxoStub) CreateChunkKey(id string, seq int64) string {
	if id == "" {
		return ""
	}
	return fmt.Sprintf("CHNK_%s_%d", id, seq)
}

//CreatePruneLogKey _
func (ub *UtxoStub) CreatePruneLogKey(id string, seq int64) string {
	return fmt.Sprintf("PLOG_%s_%d", id, seq)
}

//GetChunk _
func (ub *UtxoStub) GetChunk(id string) (*Chunk, error) {
	data, err := ub.stub.GetState(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the chunk state")
	}
	if data != nil {
		chunk := &Chunk{}
		if err = json.Unmarshal(data, chunk); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal the chunk")
		}
		return chunk, nil
	}
	return nil, errors.New("the chunk doesn't exist")
}

// Pay _
func (ub *UtxoStub) Pay(sender, receiver *Balance, amount Amount, memo, pkey string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(ub.stub)
	if nil != err {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	bb := NewBalanceStub(ub.stub)
	//if negative amount, then create the new negative chunk to the sender and add balance to the receiver.
	if amount.Sign() < 0 {
		chunk := NewChunkType(sender.GetID(), amount, receiver.GetID(), pkey, ts)
		if err = ub.PutChunk(chunk); nil != err {
			return nil, err
		}
		//deposit to the receiver's account
		amount.Neg()
		receiver.Amount.Add(&amount)
		receiver.UpdatedTime = ts
		if err = bb.PutBalance(receiver); err != nil {
			return nil, err
		}

		sbl := NewBalanceTransferLog(sender, receiver, amount, memo)
		sbl.CreatedTime = ts
		if err = bb.PutBalanceLog(sbl); err != nil {
			return nil, err
		}
		return sbl, nil

	}
	//if positive amount, then create the new positive chunk to the receiver and subtract balance to the sender.
	chunk := NewChunkType(receiver.GetID(), amount, sender.GetID(), pkey, ts)
	if err = ub.PutChunk(chunk); nil != err {
		return nil, err
	}

	// withdraw from the sender's account
	amount.Neg()
	sender.Amount.Add(&amount)
	sender.UpdatedTime = ts

	if err = bb.PutBalance(sender); err != nil {
		return nil, err
	}

	//create the sender's balance log. This sender's balance log is returned as response.
	sbl := NewBalanceTransferLog(sender, receiver, amount, memo)
	sbl.CreatedTime = ts
	if err = bb.PutBalanceLog(sbl); err != nil {
		return nil, err
	}

	return sbl, nil
}

// PutChunk _
func (ub *UtxoStub) PutChunk(chunk *Chunk) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the balance")
	}
	if err = ub.stub.PutState(ub.CreateChunkKey(chunk.DOCTYPEID, chunk.CreatedTime.UnixNano()), data); err != nil {
		return errors.Wrap(err, "failed to put the balance state")
	}
	return nil
}

// GetChunkSumByTime _{end sum next}
func (ub *UtxoStub) GetChunkSumByTime(id string, stime, etime *txtime.Time) (*ChunkSum, error) {
	query := CreateQueryUtxoChunks(id, stime, etime)
	iter, err := ub.stub.GetQueryResult(query)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var s int64
	cs := &ChunkSum{}
	c := &Chunk{}
	cnt := 0 //record counter
	if !iter.HasNext() {
		return nil, NotExistUtxoChunksError{stime: stime, etime: etime}
	}

	for iter.HasNext() {
		cnt++
		kv, err := iter.Next()
		if nil != err {
			return nil, err
		}

		err = json.Unmarshal(kv.Value, c)
		if err != nil {
			return nil, err
		}
		//get the next chunk key ( +1 chunk after the threshhold)
		if cnt == UtxoChunksFetchSize+1 {
			cs.Next = kv.GetKey()
			break
		}
		s += c.Amount.Int64()
		cs.End = kv.GetKey()
	}

	sum, err := NewAmount(strconv.FormatInt(s, 10))
	cs.Sum = sum

	return cs, nil

}

// GetTotalRefundAmount _
func (ub *UtxoStub) GetTotalRefundAmount(id, pkey string) (*Amount, error) {
	query := CreateQueryRefundChunks(id, pkey)
	iter, err := ub.stub.GetQueryResult(query)
	if err != nil {
		return nil, err
	}

	amount, err := NewAmount("0")
	if err != nil {
		return nil, err
	}

	defer iter.Close()

	for iter.HasNext() {
		kv, err := iter.Next()
		chunk := &Chunk{}
		err = json.Unmarshal(kv.Value, &chunk)
		if err != nil {
			return nil, err
		}

		amount = amount.Add(chunk.Amount.Neg())
	}

	return amount, nil
}
