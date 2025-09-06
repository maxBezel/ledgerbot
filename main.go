package main

import (
	"context"
	"flag"
	"log"

	api "github.com/OvyFlash/telegram-bot-api"
	"github.com/maxBezel/ledgerbot/model"
	sql "github.com/maxBezel/ledgerbot/storage"
)

type Server struct {
	bot     *api.BotAPI
	storage *sql.Storage
}

func (s *Server) HandleUpdates(update api.Update) {
	if update.Message == nil {
		return
	}
	command := update.Message.Command()

	switch command {
	case "start":
		s.start(update.Message)
	case "new":
		s.newAccount(context.Background(), update.Message)
	default:
		return
	}

}

func (s *Server) newAccount(ctx context.Context, msg *api.Message) {
	accName := msg.CommandArguments()
	if accName == "" {
		s.bot.Send(api.NewMessage(msg.Chat.ID, "No account name given. Usage: /new accountName"))
		return
	}

	// TODO: check if account already exists

	acc := model.NewAccount(accName, msg.Chat.ID)
	if err := s.storage.AddAccount(ctx, acc); err != nil {
		log.Printf("unable to add new account %v", err)
		return
	}

	s.bot.Send(api.NewMessage(msg.Chat.ID, "created new account "+accName))
}

func (s *Server) start(msg *api.Message) {
	if _, err := s.bot.Send(api.NewMessage(msg.Chat.ID, startMsg)); err != nil {
		log.Printf("unable to send message, %v", err)
	}
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

	server := Server{
		bot:     bot,
		storage: storage,
	}

	config := api.NewUpdate(0)
	updates := bot.GetUpdatesChan(config)

	for update := range updates {
		server.HandleUpdates(update)
	}
}
