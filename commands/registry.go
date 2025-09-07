package commands

import (
	"context"
	"log"

	api "github.com/OvyFlash/telegram-bot-api"
	"github.com/maxBezel/ledgerbot/model"
	sqlite "github.com/maxBezel/ledgerbot/storage"
)

type Bot interface {
	Send(c api.Chattable) (api.Message, error)
}
type Storage interface {
	AddAccount(ctx context.Context, acc *model.Account) error
	RemoveAccount(ctx context.Context, chatID int64, name string) error
	GetAll(ctx context.Context, chatID int64) ([]string, error)
	ApplyDeltaAndLog(ctx context.Context, chatId int64, name string, delta float64, txs *model.Transaction) (newBalance float64, txnID int64, err error)
	Exists(ctx context.Context, chatID int64, name string) (bool, error)
	GetAccountID(ctx context.Context, chatID int64, name string) (int, error)
	RevertTransaction(ctx context.Context, txsId int64) (error)
	ListAccountBalances(ctx context.Context, chatID int64) ([]sqlite.AccountBalance, error)
	WriteTransactionsCsv(ctx context.Context, chatId int64, filename string) error
	GetCurrentBalance(ctx context.Context, accountID int) (float64, error)
}

type Deps struct {
	Bot     Bot
	Storage Storage
}

type Handler func(ctx context.Context, d Deps, msg *api.Message) error

type Command struct {
	Name        string
	Description string
	Hidden 		  bool
	Handle      Handler
}

type Registry struct {
	deps Deps
	m    map[string]Command
}

func NewRegistry(deps Deps) *Registry {
	return &Registry{deps: deps, m: make(map[string]Command)}
}

func (r *Registry) Register(cmd Command) { r.m[cmd.Name] = cmd }

func (r *Registry) Handle(ctx context.Context, msg *api.Message) bool {
	if msg == nil {
		return false
	}
	name := msg.Command()

	if c, ok := r.m[name]; ok {
		_ = c.Handle(ctx, r.deps, msg)
		return true
	} else {
		if t, ok := r.m["transaction"]; ok {
			err := t.Handle(ctx, r.deps, msg)
			if err != nil {
				log.Printf(err.Error())
			}
			
			return true
		}
	}
	return false
}

func (r *Registry) BotCommands() []api.BotCommand {
	out := make([]api.BotCommand, 0, len(r.m))
	for _, c := range r.m {
		if c.Hidden {
			continue
		}
		out = append(out, api.BotCommand{Command: c.Name, Description: c.Description})
	}
	return out
}
