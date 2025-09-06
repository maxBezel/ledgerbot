package model

import (
	"strconv"
	"strings"
	"time"
)

type Transaction struct {
	Id        int
	AccountId int
	Amount    float64
	Expression string
	Note      string
	CreatedAt string
	CreatedBy int64
}

func NewTransaction(accountId int, amount float64, note string, expression string, createdBy int64) *Transaction {
	return &Transaction{
		AccountId: accountId,
		Amount: amount,
		Expression: expression,
		Note: strings.TrimSpace(note),
		CreatedBy: createdBy,
		CreatedAt: strconv.FormatInt(time.Now().UTC().Unix(), 10),
	}
}
