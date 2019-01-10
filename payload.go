// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

// Payload _
type Payload interface {
	// MarshalPayload returns bytes array for response payload
	MarshalPayload() ([]byte, error)
}
