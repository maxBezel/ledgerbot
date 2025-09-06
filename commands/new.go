package commands

import (
	"context"

	api "github.com/OvyFlash/telegram-bot-api"
	"github.com/maxBezel/ledgerbot/model"
)

func New() Command {
	return Command{
		Name:        "new",
		Description: "Создать новый аккаунт",
		Handle: func(ctx context.Context, d Deps, msg *api.Message) error {
			accName := msg.CommandArguments()
			if accName == "" {
				_, _ = d.Bot.Send(api.NewMessage(msg.Chat.ID, "No account name given. Usage: /new accountName"))
				return nil
			}

			acc := model.NewAccount(accName, msg.Chat.ID)
			if err := d.Storage.AddAccount(ctx, acc); err != nil {
				return err
			}

			_, _ = d.Bot.Send(api.NewMessage(msg.Chat.ID, "created new account " + accName))
			return nil
		},
	}
}