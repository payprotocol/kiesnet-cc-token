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
const UtxoChunksFetchSize = 5

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

// GetUtxoChunksResult _
type GetUtxoChunksResult struct {
	FromKey      string
	ToKey        string
	Sum          int64
	MergeCount   int
	NextChunkKey string
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

//CreateMergeHistoryKey _
func (ub *UtxoStub) CreateMergeHistoryKey(id string, seq int64) string {
	return fmt.Sprintf("MGHR_%s_%d", id, seq)
}

//GetChunk _
func (ub *UtxoStub) GetChunk(id string) (*PayChunk, error) {
	data, err := ub.stub.GetState(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the chunk state")
	}
	if data != nil {
		chunk := &PayChunk{}
		if err = json.Unmarshal(data, chunk); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal the chunk")
		}
		return chunk, nil
	}
	return nil, errors.New("the chunk doesn't exist")
}

// Pay _
func (ub *UtxoStub) Pay(sender, receiver *Balance, amount Amount, memo string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(ub.stub)
	if nil != err {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}
	// 리시버 청크에 붙여주기
	chunk := NewPayChunkType(receiver.GetID(), receiver, amount, ts)
	if err = ub.PutChunk(chunk); nil != err {
		return nil, err
	}

	// withdraw from the sender's account
	amount.Neg()
	sender.Amount.Add(&amount)
	sender.UpdatedTime = ts

	bb := NewBalanceStub(ub.stub)
	if err = bb.PutBalance(sender); err != nil {
		return nil, err
	}

	//create the sender's balance log. This sender's balance log is returned as response.
	sbl := NewBalanceTransferLog(sender, receiver, amount, memo)
	sbl.CreatedTime = ts
	if err = bb.PutBalanceLog(sbl); err != nil {
		return nil, err
	}

	/*
		//TODO: do we need to create the receiver's utxo/pay log, too??
	*/

	return sbl, nil
}

// PutChunk _
func (ub *UtxoStub) PutChunk(chunk *PayChunk) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the balance")
	}
	if err = ub.stub.PutState(ub.CreateChunkKey(chunk.DOCTYPEID, chunk.CreatedTime.UnixNano()), data); err != nil {
		return errors.Wrap(err, "failed to put the balance state")
	}
	return nil
}

// PutMergeHistory _
func (ub *UtxoStub) PutMergeHistory(mergeHistory *MergeHistory) error {
	data, err := json.Marshal(mergeHistory)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the merge history")
	}
	if err = ub.stub.PutState(ub.CreateMergeHistoryKey(mergeHistory.DOCTYPEID, mergeHistory.CreatedTime.UnixNano()), data); err != nil {
		return errors.Wrap(err, "failed to put the merge history state")
	}
	return nil
}

// GetUtxoChunksByTime _
func (ub *UtxoStub) GetUtxoChunksByTime(id string, stime, etime *txtime.Time) (*GetUtxoChunksResult, error) {
	var sum int64
	var result = GetUtxoChunksResult{}

	//create query
	query := CreateQueryUtxoChunks(id, stime, etime)

	fmt.Println("######### query:", query)

	iter, err := ub.stub.GetQueryResult(query)
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	recCount := 0 //record counter

	for iter.HasNext() {
		recCount++
		cqResult := ChunkQueryResult{}
		kv, _ := iter.Next()

		err = json.Unmarshal(kv.Value, &cqResult)
		if err != nil {
			return nil, err
		}

		fmt.Println("######### Current Chunk's Amount : ", cqResult.Amount)

		//get the next chunk key ( +1 chunk after the threshhold)
		if recCount == UtxoChunksFetchSize+1 {
			result.NextChunkKey = kv.GetKey()
			break
		}

		if recCount == 1 {
			result.FromKey = kv.GetKey()
			//fmt.Println("######### First Chunk's ID : ", result.FromAddress)
		}

		val, err := strconv.ParseInt(cqResult.Amount, 10, 64)
		if err != nil {
			return nil, err
		}

		sum += val
		result.ToKey = kv.GetKey()

	}

	result.Sum = sum
	result.MergeCount = recCount

	return &result, nil

}

// GetLatestMergeHistory _
// id : MergeHistory owner's id.
func (ub *UtxoStub) GetLatestMergeHistory(id string) (*MergeHistory, error) {

	query := CreateQueryLatestMergeHistory(id)
	iter, err := ub.stub.GetQueryResult(query)

	if err != nil {
		fmt.Println("##### im here 1")
		return nil, err
	}

	//result not found by the query. That means there is no merge history under this ID yet.
	if !iter.HasNext() {
		fmt.Println("##### im here 2")
		return nil, nil
	}

	defer iter.Close()

	kv, err := iter.Next()
	mergeHistory := MergeHistory{}
	err = json.Unmarshal(kv.Value, &mergeHistory)
	if err != nil {
		fmt.Println("##### im here 3")
		return nil, err
	}

	return &mergeHistory, nil

}
