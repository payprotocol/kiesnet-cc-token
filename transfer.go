// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import (
	"time"
)

// TransferLog _
type TransferLog struct {
	DOCTYPEID   string     `json:"@transfer_log"`
	Code        string     `json:"code"`
	Amount      string     `json:"amount"`
	Memo        string     `json:"memo"`
	CreatedTime *time.Time `json:"created_time,omitempty"`
}
