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

//CreatePruneLogKey _
func (ub *UtxoStub) CreatePruneLogKey(id string, seq int64) string {
	return fmt.Sprintf("PLOG_%s_%d", id, seq)
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
func (ub *UtxoStub) Pay(sender, receiver *Balance, amount Amount, memo, pkey string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(ub.stub)
	if nil != err {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	bb := NewBalanceStub(ub.stub)
	fmt.Println("#### 1")
	//if negative amount, then create the new negative chunk to the sender and add balance to the receiver.
	if amount.Sign() < 0 {
		fmt.Println("#### 2")
		chunk := NewPayChunkType(sender.GetID(), amount, receiver.GetID(), pkey, ts)
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
	fmt.Println("#### 3")
	//if positive amount, then create the new positive chunk to the receiver and subtract balance to the sender.
	chunk := NewPayChunkType(receiver.GetID(), amount, sender.GetID(), pkey, ts)
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

// Prune _
func (ub *UtxoStub) Prune(id string, receiver *Balance, amount Amount, qResult *GetUtxoChunksResult) (*PruneLog, error) {
	ts, err := txtime.GetTime(ub.stub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	bb := NewBalanceStub(ub.stub)
	receiver.Amount.Add(&amount)
	receiver.UpdatedTime = ts
	if err = bb.PutBalance(receiver); err != nil {
		return nil, err
	}

	pLog := NewPruneLog(id, qResult.FromKey, qResult.ToKey, receiver.DOCTYPEID, qResult.NextChunkKey, qResult.Sum)
	pLog.CreatedTime = ts

	if err := ub.PutPruneLog(pLog); err != nil {
		return nil, err
	}

	return pLog, nil
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

// PutPruneLog _
func (ub *UtxoStub) PutPruneLog(pruneLog *PruneLog) error {
	data, err := json.Marshal(pruneLog)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the prune log")
	}
	if err = ub.stub.PutState(ub.CreatePruneLogKey(pruneLog.DOCTYPEID, pruneLog.CreatedTime.UnixNano()), data); err != nil {
		return errors.Wrap(err, "failed to put the prune log state")
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
		chunk := &PayChunk{}
		err = json.Unmarshal(kv.Value, &chunk)
		if err != nil {
			return nil, err
		}

		amount = amount.Add(chunk.Amount.Neg())
	}

	return amount, nil
}

// GetLatestPruneLog _
// id : prune log  owner's id.
func (ub *UtxoStub) GetLatestPruneLog(id string) (*PruneLog, error) {

	query := CreateQueryLatestPruneLog(id)
	iter, err := ub.stub.GetQueryResult(query)

	if err != nil {
		return nil, err
	}

	//result not found by the query. That means there is no prune log under this ID yet.
	if !iter.HasNext() {
		return nil, nil
	}

	defer iter.Close()

	kv, err := iter.Next()
	pruneLog := PruneLog{}
	err = json.Unmarshal(kv.Value, &pruneLog)
	if err != nil {
		return nil, err
	}

	return &pruneLog, nil

}

// getPruneStartTime _
// Prune start time is always retrieved from prune log regardless of the next chunk key presence.
// Next chunk key is used just to indicate there are remaining chunks to merge in the given time period.
func getPruneStartTime(ub *UtxoStub, id string) (*txtime.Time, error) {
	dsTime := "2019-01-01T12:00:00.000000000Z"
	var stime *txtime.Time
	mh, err := ub.GetLatestPruneLog(id)
	if err != nil {
		return nil, err
	} else if mh == nil { //There is no prune log yet.
		stime, err = txtime.Parse(dsTime)
		fmt.Println("######getPruneStartTime debug 1")
		if err != nil {
			fmt.Println("######getPruneStartTime debug 2")
			return nil, err
		}
	} else { //PruneLog exists
		nChunk, err := ub.GetChunk(mh.PruneToAddress)
		if err != nil {
			return nil, err
		}
		stime = nChunk.CreatedTime
	}

	fmt.Println("############ Merge search start time: ", stime)
	return stime, nil
}
