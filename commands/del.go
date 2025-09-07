package commands

import (
	"context"

	api "github.com/OvyFlash/telegram-bot-api"
	msgs "github.com/maxBezel/ledgerbot/internal/messages"
)

func Del() Command {
	return Command{
		Name:        "del",
		Description: "Удалить существующий аккаунт",
		Handle: func(ctx context.Context, d Deps, msg *api.Message) error {
			chatID := msg.Chat.ID
			accName := msg.CommandArguments()
			if accName == "" {
				_, _ = d.Bot.Send(api.NewMessage(chatID, msgs.T(msgs.NoAccountName)))
				return nil
			}

			exists, err := d.Storage.Exists(ctx, chatID, accName)
			if err != nil {
				return err
			}
			if !exists {
				reply := msgs.T(msgs.AccDoesNotExist, accName)
				_, _ = d.Bot.Send(api.NewMessage(chatID, reply))
				return nil
			}

			if err := d.Storage.RemoveAccount(ctx, chatID, accName); err != nil {
				return err
			}

			reply := msgs.T(msgs.AccRemoved, accName)
			_, _ = d.Bot.Send(api.NewMessage(chatID, reply))
			return nil
		},
	}
}

