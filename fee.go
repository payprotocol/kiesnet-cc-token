// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

var _feeRates = map[string]map[string]FeeRate{
	"pci": {
		"transfer": FeeRate{Rate: 0.1, MaxFee: 10000},
		"pay":      FeeRate{Rate: -0.2},
	},
}

// FeeRate _
type FeeRate struct {
	Rate   float32
	MaxFee int64 // 0 is unlimit
}

// GetFeeRate _
func GetFeeRate(tokenCode string, funcName string) FeeRate {
	return _feeRates["pci"][funcName]
}
