package model

import (
	"strconv"
	"strings"
	"time"
)

type Account struct {
	Id        int
	Name      string
	ChatId    int64
	Balance   float64
	CreatedAt string
}

func NewAccount(name string, chatID int64) *Account {
	return &Account{
		Name:      strings.TrimSpace(name),
		ChatId:    chatID,
		Balance:   0,
		CreatedAt: strconv.FormatInt(time.Now().UTC().Unix(), 10),
	}
}
