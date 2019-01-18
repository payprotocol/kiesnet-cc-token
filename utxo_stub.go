package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
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
func (ub *UtxoStub) CreateKey(id string) string {
	return "CHNK_" + id
}

// PutChunk _
func (ub *UtxoStub) PutChunk(chunk *Chunk) (string, error) {
	key := ub.CreateKey(ub.stub.GetTxID())
	data, err := json.Marshal(chunk)
	if nil != err {
		return "", err
	}
	if err = ub.stub.PutState(key, data); nil != err {
		return "", err
	}
	return key, nil
}

// CreateMergeResultKey _
func (ub *UtxoStub) CreateMergeResultKey(id, mergedKey string) string {
	return fmt.Sprintf("MERG_%s_%s", id, mergedKey)
}

// GetSumOfUtxoChunksByRange _
func (ub *UtxoStub) GetSumOfUtxoChunksByRange(id string, stime, etime *txtime.Time) (*Amount, string, string, error) {

	query := CreateQueryChunks(id, stime, etime)
	iter, err := ub.stub.GetQueryResult(query)
	if nil != err {
		return nil, "", "", err
	}
	defer iter.Close()

	cnt := 1
	var s int64
	sk := ""
	ek := ""
	var kv *queryresult.KV
	for iter.HasNext() {
		chunk := &Chunk{}
		kv, err = iter.Next()
		if nil != err {
			return nil, "", "", err
		}

		err = json.Unmarshal(kv.Value, chunk)
		if nil != err {
			return nil, "", "", err
		}
		s += chunk.Amount.Int64()
		if 1 == cnt {
			sk = kv.Key
		}
		cnt++
		if UtxoChunksFetchSize == cnt {
			break
		}
	}
	ek = kv.Key
	sum, err := NewAmount(strconv.FormatInt(s, 10))
	if nil != err {
		return nil, "", "", err
	}
	return sum, sk, ek, nil
}

// GetMergeResultChunk _
func (ub *UtxoStub) GetMergeResultChunk(key string) (*MergeResult, error) {
	// TODO : key validation
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
func (ub *UtxoStub) MergeRangeValidator(id string, stime *txtime.Time) (bool, error) {
	query := CreateQueryMergeResultByStartDate(id, stime)
	iter, err := ub.stub.GetQueryResult(query)
	if nil != err {
		return false, err
	}
	defer iter.Close()
	if !iter.HasNext() {
		return false, nil
	}
	return true, nil

}
