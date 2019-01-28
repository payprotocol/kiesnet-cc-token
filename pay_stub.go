// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/key-inside/kiesnet-ccpkg/txtime"
	"github.com/pkg/errors"
)

// UtxoPaysPruneSize _
const UtxoPaysPruneSize = 500

// UtxoPaysFetchSize _
const UtxoPaysFetchSize = 20 // ???: 20, 아래서  max용으로 사용하고 있음.

// UtxoStub _
type UtxoStub struct {
	stub shim.ChaincodeStubInterface
}

// NewUtxoStub _
func NewUtxoStub(stub shim.ChaincodeStubInterface) *UtxoStub {
	return &UtxoStub{stub}
}

// CreatePayKey _
func (ub *UtxoStub) CreatePayKey(id string, unixnano int64) string {
	if id == "" {
		return ""
	}
	return fmt.Sprintf("PAY_%s_%d", id, unixnano)
}

// CreatePayKeyByTime _
func (ub *UtxoStub) CreatePayKeyByTime(id string, ts *txtime.Time) string {
	unixNano := ts.UnixNano()
	return ub.CreatePayKey(id, unixNano)
}

//GetPay _
func (ub *UtxoStub) GetPay(key string) (*Pay, error) {
	data, err := ub.stub.GetState(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the pay state")
	}
	if data != nil {
		pay := &Pay{}
		if err = json.Unmarshal(data, pay); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal the pay")
		}
		return pay, nil
	}
	return nil, NotExistedPayError{key: key}
}

// PutPay _
func (ub *UtxoStub) PutPay(pay *Pay) error {
	data, err := json.Marshal(pay)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the balance")
	}
	if err = ub.stub.PutState(ub.CreatePayKey(pay.DOCTYPEID, pay.CreatedTime.UnixNano()), data); err != nil {
		return errors.Wrap(err, "failed to put the balance state")
	}
	return nil
}

// Pay _
func (ub *UtxoStub) Pay(sender, receiver *Balance, amount Amount, memo, pkey string) (*BalanceLog, error) {
	ts, err := txtime.GetTime(ub.stub)
	if nil != err {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	key := ub.CreatePayKey(receiver.GetID(), ts.UnixNano())
	if p, err := ub.GetPay(key); nil != err {
		if _, ok := err.(NotExistedPayError); !ok {
			return nil, errors.Wrap(err, "failed to get a pay by key")
		}
		if nil != p {
			return nil, errors.New("duplicate pay exists")
		}
	}
	pay := NewPay(receiver.GetID(), amount, sender.GetID(), pkey, memo, ts)
	if err = ub.PutPay(pay); nil != err {
		return nil, errors.Wrap(err, "failed to put new pay")
	}

	amount.Neg()
	sender.Amount.Add(&amount)
	sender.UpdatedTime = ts
	if err = NewBalanceStub(ub.stub).PutBalance(sender); nil != err {
		return nil, errors.Wrap(err, "failed to update sender balance")
	}

	var sbl *BalanceLog
	sbl = NewBalanceWithPayLog(sender, pay)
	sbl.CreatedTime = ts
	if err = NewBalanceStub(ub.stub).PutBalanceLog(sbl); err != nil {
		return nil, errors.Wrap(err, "failed to update sender balance log")
	}

	return sbl, nil
}

// Refund _
func (ub *UtxoStub) Refund(sender, receiver *Balance, amount Amount, memo string, parentPay *Pay) (*BalanceLog, error) {
	ts, err := txtime.GetTime(ub.stub)
	if nil != err {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	key := ub.CreatePayKey(receiver.GetID(), ts.UnixNano())
	if p, err := ub.GetPay(key); nil != err {
		if _, ok := err.(NotExistedPayError); !ok {
			return nil, errors.Wrap(err, "failed to get a pay by key")
		}
		if nil != p {
			return nil, errors.New("duplicate pay exists")
		}
	}

	pay := NewPay(sender.GetID(), amount, receiver.GetID(), ub.CreatePayKeyByTime(parentPay.DOCTYPEID, parentPay.CreatedTime), memo, ts)

	if err = ub.PutPay(pay); nil != err {
		return nil, errors.Wrap(err, "failed to put new pay")
	}

	receiver.Amount.Add(amount.Copy().Neg())
	receiver.UpdatedTime = ts

	if err = NewBalanceStub(ub.stub).PutBalance(receiver); nil != err {
		return nil, errors.Wrap(err, "failed to update receiver balance")
	}

	//update the total refund amount to the parent utxo
	parentPay.TotalRefund = *parentPay.TotalRefund.Add(amount.Copy().Neg())

	err = ub.PutPay(parentPay)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update parent pay utxo")
	}

	var rbl *BalanceLog
	rbl = NewBalanceWithRefundLog(receiver, pay)
	rbl.CreatedTime = ts

	if err = NewBalanceStub(ub.stub).PutBalanceLog(rbl); err != nil {
		return nil, errors.Wrap(err, "failed to update receiver's balance log")
	}

	return rbl, nil
}

// GetPaySumByTime _{end sum next}
func (ub *UtxoStub) GetPaySumByTime(id string, stime, etime *txtime.Time) (*PaySum, error) {
	query := CreateQueryUtxoPrunePays(id, stime, etime)
	iter, err := ub.stub.GetQueryResult(query)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var s int64
	cs := &PaySum{HasMore: false}
	c := &Pay{}
	cnt := 0 //record counter
	if !iter.HasNext() {
		return nil, errors.New(fmt.Sprintf("No pays between %s and %s", stime, etime))
	}

	for iter.HasNext() {
		if 0 == cnt {
			cs.Start = c.DOCTYPEID
		}
		cnt++
		kv, err := iter.Next()
		if nil != err {
			return nil, err
		}

		err = json.Unmarshal(kv.Value, c)
		if err != nil {
			return nil, err
		}
		//get the next pay key ( +1 pay after the threshhold)
		if cnt == UtxoPaysPruneSize+1 {
			cs.HasMore = true
			break
		}
		s += c.Amount.Int64()
		cs.End = c.DOCTYPEID
	}

	sum, err := NewAmount(strconv.FormatInt(s, 10))
	cs.Sum = sum

	return cs, nil

}

// GetUtxoPaysByTime _
func (ub *UtxoStub) GetUtxoPaysByTime(id, bookmark string, stime, etime *txtime.Time, fetchSize int) (*QueryResult, error) {
	if fetchSize < 1 {
		fetchSize = UtxoPaysFetchSize
	}
	if fetchSize > 200 {
		fetchSize = UtxoPaysFetchSize
	}
	query := ""
	if stime != nil || etime != nil {
		query = CreateQueryUtxoPaysByIDAndTime(id, stime, etime)
	} else {
		query = CreateQueryUtxoPaysByID(id)
	}
	iter, meta, err := ub.stub.GetQueryResultWithPagination(query, int32(fetchSize), bookmark)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	return NewQueryResult(meta, iter)
}

// PayPendingBalance _
func (ub *UtxoStub) PayPendingBalance(pb *PendingBalance, merchant, memo string) error {
	ts, err := txtime.GetTime(ub.stub)
	if nil != err {
		return err
	}

	key := ub.CreatePayKey(merchant, ts.UnixNano())
	if p, err := ub.GetPay(key); nil != err {
		if _, ok := err.(NotExistedPayError); !ok {
			return errors.Wrap(err, "failed to get a pay by key")
		}
		if nil != p {
			return errors.New("duplicate pay exists")
		}
	}
	// Put pay
	pay := NewPay(merchant, pb.Amount, pb.Account, "", memo, ts)
	if err = ub.PutPay(pay); nil != err {
		return errors.Wrap(err, "failed to put new pay")
	}

	// remove pending balance
	if err := ub.stub.DelState(NewBalanceStub(ub.stub).CreatePendingKey(pb.DOCTYPEID)); err != nil {
		return errors.Wrap(err, "failed to delete the pending balance")
	}
	return nil
}
