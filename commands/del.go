package commands

import (
	"context"

	api "github.com/OvyFlash/telegram-bot-api"
)

func Del() Command {
	return Command{
		Name:        "del",
		Description: "Удалить существующий аккаунт",
		Handle: func(ctx context.Context, d Deps, msg *api.Message) error {
			chatID := msg.Chat.ID
			accName := msg.CommandArguments()
			if accName == "" {
				_, _ = d.Bot.Send(api.NewMessage(chatID, "No account name given. Usage: /del accountName"))
				return nil
			}

			exists, err := d.Storage.Exists(ctx, chatID, accName)
			if err != nil { 
				return err 
			}
			if !exists {
				_, _ = d.Bot.Send(api.NewMessage(chatID, "Requested account does not exist"))
				return nil
			}

			if err := d.Storage.RemoveAccount(ctx, chatID, accName); err != nil { 
				return err 
			}

			_, _ = d.Bot.Send(api.NewMessage(chatID, "account successfully removed"))
			return nil
		},
	}
}