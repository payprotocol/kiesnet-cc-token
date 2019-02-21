// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

var _feePolicies = map[string]*FeePolicy{}

// FeePolicy _
type FeePolicy struct {
	TargetAddress string
	Rates         map[string]FeeRate
}

// FeeRate _
type FeeRate struct {
	Rate      float32
	MaxAmount int64 // 0 is unlimit
}
