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

// PaysPruneSize _
const PaysPruneSize = 900

// PaysFetchSize _
const PaysFetchSize = 20

// PayStub _
type PayStub struct {
	stub shim.ChaincodeStubInterface
}

// NewPayStub _
func NewPayStub(stub shim.ChaincodeStubInterface) *PayStub {
	return &PayStub{stub}
}

// CreatePayKey _
func (pb *PayStub) CreatePayKey(id string) string {
	if id == "" {
		return ""
	}
	return fmt.Sprintf("PAY_%s", id)
}

// CreatePayKeyByTime _
func (pb *PayStub) CreatePayKeyByTime(ts *txtime.Time, txid string) string {
	return pb.CreatePayKey(fmt.Sprintf("%d%s", ts.UnixNano(), txid))
}

//GetPay _
func (pb *PayStub) GetPay(key string) (*Pay, error) {
	data, err := pb.stub.GetState(key)
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
func (pb *PayStub) PutPay(pay *Pay) error {
	data, err := json.Marshal(pay)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the balance")
	}
	if err = pb.stub.PutState(pb.CreatePayKey(pay.PayID), data); err != nil {
		return errors.Wrap(err, "failed to put the balance state")
	}
	return nil
}

// PutParentPay _
func (pb *PayStub) PutParentPay(key string, pay *Pay) error {
	data, err := json.Marshal(pay)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the balance")
	}
	if err = pb.stub.PutState(key, data); err != nil {
		return errors.Wrap(err, "failed to put the balance state")
	}
	return nil
}

// Pay _
func (pb *PayStub) Pay(sender *Balance, receiver string, amount Amount, memo string) (*PayResult, error) {
	ts, err := txtime.GetTime(pb.stub)
	if nil != err {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}
	payid := fmt.Sprintf("%d%s", ts.UnixNano(), pb.stub.GetTxID())
	pay := NewPay(receiver, payid, amount, sender.GetID(), "", memo, ts)
	if err = pb.PutPay(pay); nil != err {
		return nil, errors.Wrap(err, "failed to put new pay")
	}

	amount.Neg()
	sender.Amount.Add(&amount)
	sender.UpdatedTime = ts
	if err = NewBalanceStub(pb.stub).PutBalance(sender); nil != err {
		return nil, errors.Wrap(err, "failed to update sender balance")
	}

	var sbl *BalanceLog
	sbl = NewBalanceWithPayLog(sender, pay)
	sbl.CreatedTime = ts
	if err = NewBalanceStub(pb.stub).PutBalanceLog(sbl); err != nil {
		return nil, errors.Wrap(err, "failed to update sender balance log")
	}

	return NewPayResult(pay, sbl), nil
}

// Refund _
func (pb *PayStub) Refund(sender, receiver *Balance, amount Amount, memo string, parentPay *Pay) (*BalanceLog, error) {
	ts, err := txtime.GetTime(pb.stub)
	if nil != err {
		return nil, errors.Wrap(err, "failed to get the timestamp")
	}

	payid := fmt.Sprintf("%d%s", ts.UnixNano(), pb.stub.GetTxID())
	pay := NewPay(sender.GetID(), payid, amount, receiver.GetID(), pb.CreatePayKeyByTime(parentPay.CreatedTime, pb.stub.GetTxID()), memo, ts)

	if err = pb.PutPay(pay); nil != err {
		return nil, errors.Wrap(err, "failed to put new pay")
	}

	receiver.Amount.Add(amount.Copy().Neg())
	receiver.UpdatedTime = ts

	if err = NewBalanceStub(pb.stub).PutBalance(receiver); nil != err {
		return nil, errors.Wrap(err, "failed to update receiver balance")
	}

	//update the total refund amount to the parent pay
	parentPay.TotalRefund = *parentPay.TotalRefund.Add(amount.Copy().Neg())

	err = pb.PutParentPay(pb.CreatePayKey(parentPay.PayID), parentPay)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update parent pay")
	}

	var rbl *BalanceLog
	rbl = NewBalanceWithRefundLog(receiver, pay)
	rbl.CreatedTime = ts

	if err = NewBalanceStub(pb.stub).PutBalanceLog(rbl); err != nil {
		return nil, errors.Wrap(err, "failed to update receiver's balance log")
	}

	return rbl, nil
}

// GetPaySumByTime _{end sum next}
func (pb *PayStub) GetPaySumByTime(id string, stime, etime *txtime.Time) (*PaySum, error) {
	query := CreateQueryPrunePays(id, stime, etime)
	iter, err := pb.stub.GetQueryResult(query)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var s int64
	cs := &PaySum{HasMore: false}
	c := &Pay{}
	cnt := 0 //record counter
	if !iter.HasNext() {
		return nil, errors.New(fmt.Sprintf("no pays between %s and %s", stime, etime))
	}

	for iter.HasNext() {
		cnt++
		kv, err := iter.Next()
		if nil != err {
			return nil, err
		}
		if 1 == cnt {
			cs.Start = kv.Key
		}
		err = json.Unmarshal(kv.Value, c)
		if err != nil {
			return nil, err
		}
		//get the next pay key ( +1 pay after the threshhold)
		if cnt == PaysPruneSize+1 {
			cs.HasMore = true
			break
		}
		s += c.Amount.Int64()
		cs.End = kv.Key
	}

	sum, err := NewAmount(strconv.FormatInt(s, 10))
	cs.Sum = sum

	return cs, nil

}

// GetPaysByTime _
func (pb *PayStub) GetPaysByTime(id, bookmark string, stime, etime *txtime.Time, fetchSize int) (*QueryResult, error) {
	if fetchSize < 1 {
		fetchSize = PaysFetchSize
	}
	if fetchSize > 200 {
		fetchSize = PaysFetchSize
	}
	query := ""
	if stime != nil || etime != nil {
		query = CreateQueryPaysByIDAndTime(id, stime, etime)
	} else {
		query = CreateQueryPaysByID(id)
	}
	iter, meta, err := pb.stub.GetQueryResultWithPagination(query, int32(fetchSize), bookmark)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	return NewQueryResult(meta, iter)
}

// PayPendingBalance _
func (pb *PayStub) PayPendingBalance(pbalance *PendingBalance, merchant, memo string) error {
	ts, err := txtime.GetTime(pb.stub)
	if nil != err {
		return err
	}
	payid := fmt.Sprintf("%d%s", ts.UnixNano(), pb.stub.GetTxID())

	// Put pay
	pay := NewPay(merchant, payid, pbalance.Amount, pbalance.Account, "", memo, ts)
	if err = pb.PutPay(pay); nil != err {
		return errors.Wrap(err, "failed to put new pay")
	}

	// remove pending balance
	if err := pb.stub.DelState(NewBalanceStub(pb.stub).CreatePendingKey(pbalance.DOCTYPEID)); err != nil {
		return errors.Wrap(err, "failed to delete the pending balance")
	}
	return nil
}
