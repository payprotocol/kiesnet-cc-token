// Copyright Key Inside Co., Ltd. 2018 All Rights Reserved.

package main

import "fmt"

// TODO: indexing

// QueryMainAccountByIDAndTokenCode _
/*
{
	"selector": {
		"@account": "%s",
		"token": "%s",
		"type": 1
	},
	"limit": 1,
	"use_index": "TODO"
}
*/
const QueryMainAccountByIDAndTokenCode = `{"selector":{"@account":"%s","token":"%s","type":1},"limit":1}`

// CreateQueryMainAccountByIDAndTokenCode _
func CreateQueryMainAccountByIDAndTokenCode(tokenCode, id string) string {
	return fmt.Sprintf(QueryMainAccountByIDAndTokenCode, id, tokenCode)
}

// QueryAccountsByID _
/*
{
	"selector": {
		"$or": [
			{
				"@account": "%s",
				"type": 1
			},
			{
				"type": 2,
				"holders": {
					"$elemMatch": {
						"$eq": "%s"
					}
				}
			}
		]
	},
	"use_index": "TODO"
}
*/
const QueryAccountsByID = `{"selector":{"$or":[{"@account":"%s","type":1},{"type":2,"holders":{"$elemMatch":{"$eq":"%s"}}}]}}`

// CreateQueryAccountsByID _
func CreateQueryAccountsByID(id string) string {
	return fmt.Sprintf(QueryAccountsByID, id, id)
}

// QueryAccountsByIDAndTokenCode _
/*
{
	"selector": {
		"$or": [
			{
				"@account": "%s",
				"token": "%s",
				"type": 1
			},
			{
				"token": "%s",
				"type": 2,
				"holders": {
					"$elemMatch": {
						"$eq": "%s"
					}
				}
			}
		]
	},
	"use_index": "TODO"
}
*/
const QueryAccountsByIDAndTokenCode = `{"selector":{"$or":[{"@account":"%s","token":"%s","type":1},{"token":"%s","type":2,"holders":{"$elemMatch":{"$eq":"%s"}}}]}}`

// CreateQueryAccountsByIDAndTokenCode _
func CreateQueryAccountsByIDAndTokenCode(tokenCode, id string) string {
	return fmt.Sprintf(QueryAccountsByIDAndTokenCode, id, tokenCode, tokenCode, id)
}
