package main

import (
	"context"
	"flag"
	"log"

	api "github.com/OvyFlash/telegram-bot-api"
	"github.com/maxBezel/ledgerbot/commands"
	sql "github.com/maxBezel/ledgerbot/storage"
)

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

	deps := commands.Deps{Bot: bot, Storage: storage}
	reg := commands.NewRegistry(deps)

	reg.Register(commands.Start())
	reg.Register(commands.New())
	reg.Register(commands.Del())
	reg.Register(commands.Transaction())
	reg.Register(commands.Get())

	if _, err := bot.Request(api.NewSetMyCommands(reg.BotCommands()...)); err != nil {
		log.Fatal(err)
	}

	config := api.NewUpdate(0)
	updates := bot.GetUpdatesChan(config)
	ctx := context.Background()

	for u := range updates {

		if u.CallbackQuery != nil {
			commands.HandleCallback(ctx, deps, u.CallbackQuery)
			continue
		}
		if u.Message != nil {
			reg.Handle(ctx, u.Message)
		}
	}
}
