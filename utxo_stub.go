package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
)

// UtxoChunksFetchSize _
const UtxoChunksFetchSize = 2

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
// 	query := CreateQueryPayChunks(id, stime, etime)
// 	bookmark := ""
// 	var result *QueryResult
// 	var err error
// 	buf := bytes.NewBufferString("")
// 	for {
// 		iter, meta, err := ub.stub.GetQueryResultWithPagination(query, int32(UtxoChunksFetchSize), bookmark)
// 		if err != nil {
// 			return err
// 		}
// 		defer iter.Close()
// 		bookmark = meta.Bookmark
// 		if meta.FetchedRecordsCount < UtxoChunksFetchSize {
// 			break
// 		}

// 	}

// 	b, _ := result.MarshalJSON()
// 	fmt.Println(string(b))
// 	return nil

// }

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
