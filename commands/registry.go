package commands

import (
	"context"
	"log"
	"strconv"
	"strings"

	api "github.com/OvyFlash/telegram-bot-api"
	"github.com/maxBezel/ledgerbot/model"
)

type Bot interface {
	Send(c api.Chattable) (api.Message, error)
}
type Storage interface {
	AddAccount(ctx context.Context, acc *model.Account) error
	AddTransaction(ctx context.Context, txs *model.Transaction) (int64, error)
	RemoveAccount(ctx context.Context, chatID int64, name string) error
	GetAll(ctx context.Context, chatID int64) ([]string, error)
	AdjustBalance(ctx context.Context, chatId int64, name string, delta float64) (float64, error)
	Exists(ctx context.Context, chatID int64, name string) (bool, error)
	GetAccountID(ctx context.Context, chatID int64, name string) (int, error)
	RevertTransaction(ctx context.Context, txsId int) (error)
}

type Deps struct {
	Bot     Bot
	Storage Storage
}

type Handler func(ctx context.Context, d Deps, msg *api.Message) error

type Command struct {
	Name        string
	Description string
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
	}

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
		out = append(out, api.BotCommand{Command: c.Name, Description: c.Description})
	}
	return out
}

func HandleCallback(ctx context.Context, d Deps, cq *api.CallbackQuery) {
    data := cq.Data
    if strings.HasPrefix(data, "undo:") {
        txID, err := strconv.Atoi(strings.TrimPrefix(data, "undo:"))
        if err == nil {
            if err := d.Storage.RevertTransaction(ctx, txID); err != nil {
                _ = answerCB(d.Bot, cq, "Failed to undo: "+err.Error(), true)
                return
            }
            _ = answerCB(d.Bot, cq, "Transaction reverted", false)

            edit := api.NewEditMessageText(cq.Message.Chat.ID, cq.Message.MessageID,
                cq.Message.Text+"\n\nReverted ✅")
            _, _ = d.Bot.Send(edit)
            return
        }
    }
    _ = answerCB(d.Bot, cq, "Unknown action", true)
}

func answerCB(bot Bot, cq *api.CallbackQuery, text string, alert bool) error {
    cb := api.NewCallback(cq.ID, text)
    if alert {
        cb.ShowAlert = true
    }

    _, err := bot.Send(cb)
    return err
}