package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
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

// NewUtxoStub _
func NewUtxoStub(stub shim.ChaincodeStubInterface) *UtxoStub {
	return &UtxoStub{stub}
}

//GetAllUtxoChunks _
//Get all UTXO Chunks of the current user. Testing purpose. In real world, must return certain number of the chunks in the given time frame.
func (ub *UtxoStub) GetAllUtxoChunks(id string, stime, etime *txtime.Time) error {
	query := CreateQueryPayChunks(id, stime, etime)
	bookmark := ""
	//var result *QueryResult
	//var err error
	//buf := bytes.NewBufferString("")

	iter, meta, err := ub.stub.GetQueryResultWithPagination(query, int32(UtxoChunksFetchSize), bookmark)
	if err != nil {
		return err
	}

	var responseArray []string

	for iter.HasNext() {
		queryResponse, err := iter.Next()
		if err != nil {
			fmt.Println(err.Error())
		}

		var buffer bytes.Buffer
		buffer.WriteString(string(queryResponse.Value))
		fmt.Println(buffer.String())
		responseArray = append(responseArray, buffer.String())
	}

	fmt.Println("################# getting all utxo chunks result")
	fmt.Println(iter)
	fmt.Println("book mark:", meta.Bookmark)
	fmt.Println("meta data.fetch records count:", meta.FetchedRecordsCount)
	defer iter.Close()

	//b, _ := result.MarshalJSON()
	//fmt.Println(string(b))
	return nil

}

// GetQueryUtxoChunk _
func (ub *UtxoStub) GetQueryUtxoChunk(id string, stime, etime *txtime.Time) (*PruneQueryResult, error) {
	var sum int64
	var result = PruneQueryResult{}
	bs := NewBalanceStub(ub.stub)
	//create query

	query := CreateQueryPayChunks(id, stime, etime)

	//fmt.Println("######### query:", query)

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
		//fmt.Println("######### kv Value", string(kv.Value))
		fmt.Println("######### Current Chunk's Amount : ", cqResult.Amount)

		//get the next chunk key ( +1 chunk after the threshhold)
		if recCount == UtxoChunksFetchSize+1 {
			result.NextChunkKey = cqResult.ID
			break
		}

		if recCount == 1 {
			result.FromKey = bs.CreateChunkKey(cqResult.ID)
			//fmt.Println("######### First Chunk's ID : ", result.FromAddress)
		}

		val, err := strconv.ParseInt(cqResult.Amount, 10, 64)
		if err != nil {
			return nil, err
		}

		sum += val
		result.ToKey = bs.CreateChunkKey(cqResult.ID)

	}

	result.Sum = sum
	result.MergeCount = recCount

	return &result, nil

}
