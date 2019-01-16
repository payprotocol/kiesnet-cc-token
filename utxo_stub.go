package main

import (
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

// GetAllUtxoChunks _
// func (ub *UtxoStub) GetAllUtxoChunks(id string, stime, etime *txtime.Time) error {
// 	bookmark := ""
// 	var result *QueryResult
// 	for result, err := ub.GetQueryUtxoChunks(id, bookmark, stime, etime); result.Meta.FetchedRecordsCount == 1000; {
// 		if nil != err {
// 			return err
// 		}
// 		result
// 	}
// 	if nil != err {
// 		return err
// 	}

// }

// GetQueryUtxoChunks _
func (ub *UtxoStub) GetQueryUtxoChunks(id, bookmark string, stime, etime *txtime.Time) (*QueryResult, error) {
	query := CreateQueryPayChunks(id, stime, etime)
	fmt.Println(query)
	iter, meta, err := ub.stub.GetQueryResultWithPagination(query, int32(UtxoChunksFetchSize), bookmark)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	return NewQueryResult(meta, iter)
}
