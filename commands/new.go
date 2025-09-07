package commands

import (
	"context"

	api "github.com/OvyFlash/telegram-bot-api"
	msgs "github.com/maxBezel/ledgerbot/internal/messages"
	"github.com/maxBezel/ledgerbot/model"
)

func New() Command {
	return Command{
		Name:        "new",
		Description: "Создать новый аккаунт",
		Hidden: false,
		Handle: func(ctx context.Context, d Deps, msg *api.Message) error {
			accName := msg.CommandArguments()
			chatID := msg.Chat.ID
			if accName == "" {
				_, _ = d.Bot.Send(api.NewMessage(msg.Chat.ID, msgs.T(msgs.NoAccountName)))
				return nil
			}

			exists, err := d.Storage.Exists(ctx, chatID, accName)
			if err != nil {
				return err
			}
			if exists {
				_, _ = d.Bot.Send(api.NewMessage(chatID, msgs.T(msgs.AccAlreadyExist)))
				return nil
			}

			acc := model.NewAccount(accName, msg.Chat.ID)
			if err := d.Storage.AddAccount(ctx, acc); err != nil {
				return err
			}

			reply := msgs.T(msgs.AccountCreated, accName)
			_, _ = d.Bot.Send(api.NewMessage(msg.Chat.ID, reply))
			return nil
		},
	}
}

