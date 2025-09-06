package main

import (
	"context"
	"flag"
	"log"

	api "github.com/OvyFlash/telegram-bot-api"
	sql "github.com/maxBezel/ledgerbot/storage"
)

type Handler func(bot *api.BotAPI, chatId int64)

func startHandler(bot *api.BotAPI, chatId int64) {
	msg := api.NewMessage(chatId, startMsg)
	if _, err := bot.Send(msg); err != nil {
		log.Fatal("unable to send message")
	}
}

func addAccountHandler(bot *api.BotAPI, chatId int64) {
	msg := api.NewMessage(chatId, addAccountMsg)
	if _, err := bot.Send(msg); err != nil {
		log.Fatal("unable to send message")
	}
}

var routes = map[string]Handler{
	"start": startHandler,
	"new":   addAccountHandler,
}

const sqlitePath = "data/data.db"

func main() {
	// token
	token := flag.String("token", "", "token provided by @BotFather")
	flag.Parse()
	if *token == "" {
		log.Fatal("no token given")
	}

	// sql
	storage, err := sql.New(sqlitePath)
	if err != nil {
		log.Fatal("failed to connect to db")
	}

	storage.Init(context.TODO())

	// bot
	bot, err := api.NewBotAPI(*token)
	if err != nil {
		log.Fatal("failed to create bot API")
	}

	config := api.NewUpdate(0)
	updates := bot.GetUpdatesChan(config)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		command := update.Message.Command()

		if command == "" {
			continue
		}

		handler, ok := routes[command]
		if !ok {
			continue
		}

		handler(bot, update.Message.Chat.ID)
	}
}
