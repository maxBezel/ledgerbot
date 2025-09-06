package model

import (
	"strings"
	"time"
)

type Account struct {
	Id        int
	Name      string
	ChatId    int64
	Balance   float64
	CreatedAt time.Time
}

func NewAccount(name string, chatID int64) *Account {
	return &Account{
		Name:      strings.TrimSpace(name),
		ChatId:    chatID,
		Balance:   0,
		CreatedAt: time.Now().UTC(),
	}
}
