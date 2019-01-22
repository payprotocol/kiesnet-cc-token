package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
)

// UtxoChunksFetchSize _
const UtxoChunksFetchSize = 500

// UtxoStub _
type UtxoStub struct {
	stub shim.ChaincodeStubInterface
}

// NewUtxoStub _
func NewUtxoStub(stub shim.ChaincodeStubInterface) *UtxoStub {
	return &UtxoStub{stub}
}

// CreateKey _
func (ub *UtxoStub) CreateKey(id string) (string, error) {
	ts, err := ub.stub.GetTxTimestamp()
	if nil != err {
		return "", err
	}
	postfix := strconv.FormatInt(ts.GetSeconds(), 10)
	return fmt.Sprintf("CHNK_%s_%s", id, postfix), nil
}

// PutChunk _
func (ub *UtxoStub) PutChunk(chunk *Chunk) (string, error) {
	key := chunk.DOCTYPEID
	data, err := json.Marshal(chunk)
	if nil != err {
		return "", err
	}
	if err = ub.stub.PutState(key, data); nil != err {
		return "", err
	}
	return key, nil
}

// GetMergeResultKey _
func (ub *UtxoStub) GetMergeResultKey(id, endkey string) string {
	return fmt.Sprintf("MERG_%s_%s", id, endkey)
	// key, err := ub.stub.CreateCompositeKey("MERG", []string{id, endkey})
	// if nil != err {
	// 	return "", err
	// }
	// return key, nil
}

// PutMergeResult _
func (ub *UtxoStub) PutMergeResult(mr *MergeResult) error {
	key := mr.DOCTYPEID
	data, err := json.Marshal(mr)
	if nil != err {
		return err
	}
	if err = ub.stub.PutState(key, data); nil != err {
		return err
	}
	return nil
}

// GetSumOfUtxoChunksByRange _
func (ub *UtxoStub) GetSumOfUtxoChunksByRange(owner Identifiable, toID string, stime, etime *txtime.Time) (*Amount, *Chunk, *Chunk, error) {
	query := CreateQueryChunks(owner.GetID(), stime, etime)
	fmt.Println(query)
	iter, err := ub.stub.GetQueryResult(query)
	if nil != err {
		return nil, nil, nil, err
	}
	defer iter.Close()
	if !iter.HasNext() {
		return nil, nil, nil, NotExistUtxoChunksError{stime: stime, etime: etime}
	}
	cnt := 1
	var s int64
	schnk := &Chunk{}
	echnk := &Chunk{}
	var kv *queryresult.KV
	for iter.HasNext() {
		c := &Chunk{}
		kv, err = iter.Next()
		if nil != err {
			return nil, nil, nil, err
		}

		err = json.Unmarshal(kv.Value, c)
		if nil != err {
			return nil, nil, nil, err
		}
		s += c.Amount.Int64()
		if 1 == cnt {
			schnk = c
		}
		cnt++
		echnk = c
		if UtxoChunksFetchSize == cnt {
			break
		}
	}
	sum, err := NewAmount(strconv.FormatInt(s, 10))
	if nil != err {
		return nil, nil, nil, err
	}
	return sum, schnk, echnk, nil
}

// GetMergeResultChunk _
func (ub *UtxoStub) GetMergeResultChunk(key string) (*MergeResult, error) {
	data, err := ub.stub.GetState(key)
	if nil != err {
		return nil, err
	}
	result := &MergeResult{}
	err = json.Unmarshal(data, result)
	if nil != err {
		return nil, err
	}
	return result, nil
}

// GetChunk _
func (ub *UtxoStub) GetChunk(key string) (*Chunk, error) {
	// TODO : key validation
	data, err := ub.stub.GetState(key)
	if nil != err {
		return nil, err
	}
	chunk := &Chunk{}
	err = json.Unmarshal(data, chunk)
	if nil != err {
		return nil, err
	}
	return chunk, nil
}

// MergeRangeValidator _
func (ub *UtxoStub) MergeRangeValidator(id string, stime, etime *txtime.Time) (bool, error) {
	query := CreateQueryMergeResultByDate(id, stime, etime)
	fmt.Println(query)
	iter, err := ub.stub.GetQueryResult(query)
	if nil != err {
		return false, err
	}
	defer iter.Close()
	if iter.HasNext() {
		return false, nil
	}
	return true, nil

}

// GetLastestMergeResultByID _
func (ub *UtxoStub) GetLastestMergeResultByID(owner Identifiable) (*MergeResult, error) {
	query := CreateQueryMergeResultsByAccount(owner.GetID())
	fmt.Println(query)
	iter, err := ub.stub.GetQueryResult(query)
	if nil != err {
		return nil, err
	}
	defer iter.Close()

	mr := &MergeResult{}
	if !iter.HasNext() {
		return nil, nil
	}
	if iter.HasNext() {
		kv, err := iter.Next()
		if nil != err {
			return nil, err
		}
		err = json.Unmarshal(kv.Value, mr)
		if nil != err {
			return nil, err
		}
	}
	return mr, nil
}

// Pay _
func (ub *UtxoStub) Pay(sender, receiver *Balance, amount Amount, memo string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(ub.stub)
	if nil != err {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}
	// 리시버 청크에 붙여주기
	bb := NewBalanceStub(ub.stub)
	key, err := ub.CreateKey(receiver.GetID())
	if nil != err {
		return nil, err
	}
	data, err := ub.stub.GetState(key)
	if nil != err {
		return nil, err
	}
	if nil != data {
		return nil, ExistUtxcoChunkError{key: key}
	}
	chunk := NewChunkType(key, receiver, sender, amount, ts)
	if _, err = ub.PutChunk(chunk); nil != err {
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

	/*
		//TODO: do we need to create the receiver's utxo/pay log, too??
	*/

	return sbl, nil
}
