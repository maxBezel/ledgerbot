package commands

import (
	"context"

	api "github.com/OvyFlash/telegram-bot-api"
	msgs "github.com/maxBezel/ledgerbot/internal/messages"
)

func Start() Command {
	return Command{
		Name:        "start",
		Description: "Начать диалог с ботом",
		Handle: func(ctx context.Context, d Deps, msg *api.Message) error {
			reply := msgs.T(msgs.Start)
			_, err := d.Bot.Send(api.NewMessage(msg.Chat.ID, reply))
			return err
		},
	}
}

