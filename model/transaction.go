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
	Note      string
	CreatedAt string
	CreatedBy int64
}

func NewTransaction(AccountId int, Amount float64, Note string, CreatedBy int64) *Transaction {
	return &Transaction{
		AccountId: AccountId,
		Amount: Amount,
		Note: strings.TrimSpace(Note),
		CreatedBy: CreatedBy,
		CreatedAt: strconv.FormatInt(time.Now().UTC().Unix(), 10),
	}
}
