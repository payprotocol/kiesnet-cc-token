package main

import (
	"bytes"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// UtxoChunksFetchSize _
const UtxoChunksFetchSize = 1000

// UtxoStub _
type UtxoStub struct {
	stub shim.ChaincodeStubInterface
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

// GetQueryUtxoChunks _
func (ub *UtxoStub) GetQueryUtxoChunks(id, bookmark string, stime, etime *txtime.Time) (*QueryResult, error) {
	query := CreateQueryPayChunks(id, stime, etime)
	iter, meta, err := ub.stub.GetQueryResultWithPagination(query, int32(UtxoChunksFetchSize), bookmark)
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	return NewQueryResult(meta, iter)
}
