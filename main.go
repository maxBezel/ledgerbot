package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

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
	case "del":
		s.removeAccount(context.Background(), update.Message)
	case "list":
		s.listAccounts(context.Background(), update.Message)
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

func (s *Server) removeAccount(ctx context.Context, msg *api.Message) {
	accName := msg.CommandArguments()
	chatId := msg.Chat.ID
	if accName == "" {
		s.bot.Send(api.NewMessage(chatId, "No account name given. Usage: /del accountName"))
		return
	}

	exists, err := s.storage.Exists(ctx, chatId, accName)
	if err != nil {
		log.Printf("Unable to check if account exists %v", err)
		return
	}

	if !exists {
		s.bot.Send(api.NewMessage(chatId, "Requested account does not exist"))
		return
	}

	if err := s.storage.RemoveAccount(ctx, chatId, accName); err != nil {
		fmt.Printf("Unable to remove account %v", err)
		return
	}

	s.bot.Send(api.NewMessage(chatId, "account successfully removed"))
}

func (s *Server) listAccounts(ctx context.Context, msg *api.Message) {
	chatId := msg.Chat.ID
	accNames, err := s.storage.GetAll(ctx, chatId)
	if err != nil {
		log.Printf("Was unable to list accounts %v", err)
		return
	}

	if len(accNames) == 0 {
		s.bot.Send(api.NewMessage(chatId, "You dont have any accounts yet"))
		return
	}

	var b strings.Builder
	b.WriteString("Your accounts:\n")
	for i, name := range accNames {
		fmt.Fprintf(&b, "%d) %s\n", i+1, name)
	}

	allAccMsg := api.NewMessage(chatId, b.String())
	if _, err := s.bot.Send(allAccMsg); err != nil {
		log.Printf("send accounts list: %v", err)
	}
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

	commands := []api.BotCommand{
		{Command: "start", Description: "Начать диалог с ботом"},
		{Command: "list",  Description: "Вывести список аккаунтов"},
		{Command: "new",  Description: "Создать новый аккаунт"},
		{Command: "del",  Description: "Удалить существующий аккаунт"},
	}

	cfg := api.NewSetMyCommands(commands...)
	if _, err := bot.Request(cfg); err != nil {
		log.Fatal(err)
	}

	def, _ := bot.GetMyCommands()
	log.Printf("default=%v", def)

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
