package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
)

// UtxoChunksPruneSize _
const UtxoChunksPruneSize = 500

// UtxoChunksFetchSize _
const UtxoChunksFetchSize = 200

// UtxoStub _
type UtxoStub struct {
	stub shim.ChaincodeStubInterface
}

// NewUtxoStub _
func NewUtxoStub(stub shim.ChaincodeStubInterface) *UtxoStub {
	return &UtxoStub{stub}
}

// CreateChunkKey _
func (ub *UtxoStub) CreateChunkKey(id string, nanosecond int64) string {
	if id == "" {
		return ""
	}
	return fmt.Sprintf("CHNK_%s_%d", id, nanosecond)
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
func (ub *UtxoStub) Pay(sender, merchant *Balance, amount Amount, memo, pkey string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(ub.stub)
	if nil != err {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}
	key := ub.CreateChunkKey(merchant.GetID(), ts.UnixNano())
	if c, err := ub.GetChunk(key); c != nil || err != nil {
		if nil != err {
			return nil, errors.Wrap(err, "failed to validate new chunk")
		}
		return nil, errors.New("duplicated chunk found")
	}

	chunk := NewChunkType(merchant.GetID(), amount, sender.GetID(), pkey, ts)
	if err = ub.PutChunk(chunk); nil != err {
		return nil, errors.Wrap(err, "failed to put new chunk")
	}

	if amount.Sign() < 0 {
		amount.Neg()
	}
	sender.Amount.Add(&amount)
	sender.UpdatedTime = ts
	if err = NewBalanceStub(ub.stub).PutBalance(sender); nil != err {
		return nil, errors.Wrap(err, "failed to update sender balance")
	}
	sbl := NewBalanceTransferLog(sender, merchant, amount, memo)
	sbl.CreatedTime = ts
	if err = NewBalanceStub(ub.stub).PutBalanceLog(sbl); err != nil {
		return nil, errors.Wrap(err, "failed to update sender balance log")
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
	query := CreateQueryUtxoPruneChunks(id, stime, etime)
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
		return nil, errors.New(fmt.Sprintf("No chunks between %s and %s", stime, etime))
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
		if cnt == UtxoChunksPruneSize+1 {
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
// Returns sum of past refund amounts in positive amount
func (ub *UtxoStub) GetTotalRefundAmount(id, pkey string) (*Amount, error) {
	query := CreateQueryRefundChunks(id, pkey)
	iter, err := ub.stub.GetQueryResult(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get query results")
	}
	amount, err := NewAmount("0")
	if err != nil {

		return nil, errors.Wrap(err, "failed to parse to number")
	}
	defer iter.Close()

	for iter.HasNext() {
		kv, err := iter.Next()
		chunk := &Chunk{}
		err = json.Unmarshal(kv.Value, &chunk)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse json to struct")
		}
		amount = amount.Add(chunk.Amount.Neg())
	}

	return amount, nil
}

// CheckDuplicatedChunk _
func (ub *UtxoStub) CheckDuplicatedChunk(key string) bool {
	chunk, err := ub.GetChunk(key)
	if err == nil && chunk != nil {
		return true
	}
	return false
}

// GetUtxoChunksByTime _
func (ub *UtxoStub) GetUtxoChunksByTime(id, bookmark string, stime, etime *txtime.Time, fetchSize int) (*QueryResult, error) {
	if fetchSize < 1 {
		fetchSize = UtxoChunksFetchSize
	}
	if fetchSize > UtxoChunksFetchSize {
		fetchSize = UtxoChunksFetchSize
	}
	query := ""
	if stime != nil || etime != nil {
		query = CreateQueryUtxoChunksByIDAndTime(id, stime, etime)
	} else {
		query = CreateQueryUtxoChunksByID(id)
	}
	iter, meta, err := ub.stub.GetQueryResultWithPagination(query, int32(fetchSize), bookmark)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	return NewQueryResult(meta, iter)
}

// PayPendingBalance _
func (ub *UtxoStub) PayPendingBalance(pb *PendingBalance, receiver *Balance, pendingTime *txtime.Time, pkey string) error {
	ts, err := txtime.GetTime(ub.stub)
	if err != nil {
		return errors.Wrap(err, "failed to get the timestamp")
	}

	sender := &Balance{DOCTYPEID: pb.Account} // proxy
	bb := NewBalanceStub(ub.stub)
	if pendingTime != nil && pendingTime.Cmp(ts) > 0 { // time lock
		pb := NewPendingBalance(ub.stub.GetTxID(), receiver, sender, pb.Amount, pb.Memo, pendingTime)
		pb.CreatedTime = ts
		if err = bb.PutPendingBalance(pb); err != nil {
			return err
		}
	} else {
		if _, err := ub.Pay(sender, receiver, pb.Amount, pb.Memo, pkey); nil != err {
			return err
		}
	}

	// remove pending balance
	if err = bb.stub.DelState(bb.CreatePendingKey(pb.DOCTYPEID)); err != nil {
		return errors.Wrap(err, "failed to delete the pending balance")
	}

	return nil

}
