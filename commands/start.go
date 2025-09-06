package commands

import (
	"context"

	api "github.com/OvyFlash/telegram-bot-api"
)

func Start(startMsg string) Command {
	return Command{
		Name:        "start",
		Description: "Начать диалог с ботом",
		Handle: func(ctx context.Context, d Deps, msg *api.Message) error {
			_, err := d.Bot.Send(api.NewMessage(msg.Chat.ID, startMsg))
			return err
		},
	}
}