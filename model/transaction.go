package model

import "time"

type Transaction struct {
	Id        int
	AccountId int
	Amount    float64
	RawText   string
	Note      string
	CreatedAt time.Time
	CreatedBy int64
}
