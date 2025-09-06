package commands

import (
	"context"
	"fmt"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	api "github.com/OvyFlash/telegram-bot-api"
	"github.com/expr-lang/expr"
	"github.com/maxBezel/ledgerbot/model"
)

func Transaction() Command {
	return Command{
		Name:        "transaction",
		Description: "Выполняет транзакцию для указанного аккаунта",
		Handle: func(ctx context.Context, d Deps, msg *api.Message) error {
			accName := msg.Command()
			usrId := msg.From.ID
			chatID := msg.Chat.ID
			expression, note, err := splitExprAndComment(msg.CommandArguments())
			if err != nil {
				_, _ = d.Bot.Send(api.NewMessage(chatID, "Invalid expression. Usage: /<account name> expr"))
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

			val, err := Eval(expression)
			if err != nil {
				return err
			}

			newBalance, err := d.Storage.AdjustBalance(ctx, chatID, accName, val)
			if err != nil {
				return err
			}

			accountId, err := d.Storage.GetAccountID(ctx, chatID, accName)
			if err != nil {
				return err
			}

			txs := model.NewTransaction(accountId, val, note, expression, usrId)
			txsId, err := d.Storage.AddTransaction(ctx, txs)
			if err != nil {
				return err
			}

			btn := api.NewInlineKeyboardButtonData("↩️ Undo", fmt.Sprintf("undo:%d", txsId))
			kb  := api.NewInlineKeyboardMarkup(api.NewInlineKeyboardRow(btn))

			msgOK := api.NewMessage(chatID,
					"Balance successfully updated. New balance: "+
					strconv.FormatFloat(newBalance, 'f', 2, 64),
			)
			msgOK.ReplyMarkup = kb
			_, _ = d.Bot.Send(msgOK)
					
			return nil
		},
	}
}

func Eval(s string) (float64, error) {
	prog, err := expr.Compile(s, expr.AsFloat64())
	if err != nil {
		return 0, fmt.Errorf("compile: %w", err)
	}
	out, err := expr.Run(prog, nil)
	if err != nil {
		return 0, fmt.Errorf("run: %w", err)
	}
	return out.(float64), nil
}

func splitExprAndComment(args string) (expr, comment string, err error) {
	s := strings.TrimSpace(args)
	if s == "" {
		return "", "", fmt.Errorf("empty arguments")
	}

	lastOK := -1
	fset := token.NewFileSet()

	for i := 1; i <= len(s); i++ {
		prefix := strings.TrimSpace(s[:i])
		if prefix == "" {
			continue
		}
		//жесткие костыли на самом деле боги литкода меня бы убили
		if _, e := types.Eval(fset, nil, token.NoPos, prefix); e == nil {
			lastOK = i
		}
	}

	if lastOK == -1 {
		return "", "", fmt.Errorf("no valid expression at start")
	}
	expr = strings.TrimSpace(s[:lastOK])
	comment = strings.TrimSpace(s[lastOK:])
	return
}