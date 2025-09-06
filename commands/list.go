package commands

import (
	"context"
	"fmt"
	"strings"

	api "github.com/OvyFlash/telegram-bot-api"
)

func List() Command {
	return Command{
		Name:        "list",
		Description: "Вывести список аккаунтов",
		Handle: func(ctx context.Context, d Deps, msg *api.Message) error {
			chatID := msg.Chat.ID
			names, err := d.Storage.GetAll(ctx, chatID)
			if err != nil { 
				return err 
			}
			
			if len(names) == 0 {
				_, _ = d.Bot.Send(api.NewMessage(chatID, "You dont have any accounts yet"))
				return nil
			}

			var b strings.Builder
			b.WriteString("Your accounts:\n")
			for i, name := range names {
				fmt.Fprintf(&b, "%d) %s\n", i+1, name)
			}
			
			_, err = d.Bot.Send(api.NewMessage(chatID, b.String()))
			return err
		},
	}
}