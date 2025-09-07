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
	msgs "github.com/maxBezel/ledgerbot/internal/messages"
	"github.com/maxBezel/ledgerbot/model"
)

func Transaction() Command {
	return Command{
		Name:        "transaction",
		Description: "Выполняет транзакцию для указанного аккаунта",
		Hidden: true,
		Handle: func(ctx context.Context, d Deps, msg *api.Message) error {
			accName := msg.Command()
			usrId := msg.From.ID
			chatID := msg.Chat.ID
			args := msg.CommandArguments()
			if accName == "" && strings.HasPrefix(msg.Text, "/") {
				accName, args = parseSlash(msg.Text)
			}
			expression, note, err := splitExprAndComment(args)
			if err != nil {
				if err.Error() != "empty arguments" || accName != "" {
					_, _ = d.Bot.Send(api.NewMessage(chatID, msgs.T(msgs.NoExpression)))
				}
				
				return nil
			}

			exists, err := d.Storage.Exists(ctx, chatID, accName)
			if err != nil {
				return err
			}
			if !exists {
				_, _ = d.Bot.Send(api.NewMessage(chatID, msgs.T(msgs.AccDoesNotExist, accName)))
				return nil
			}

			val, err := Eval(expression)
			if err != nil {
				return err
			}

			accountId, err := d.Storage.GetAccountID(ctx, chatID, accName)
			if err != nil {
				return err
			}

			balance, err := d.Storage.GetCurrentBalance(ctx, accountId)
			if err != nil {
				return err
			}

			newBalance := balance + val
			txs := model.NewTransaction(accountId, val, note, newBalance, expression, usrId)
			newBalance, txsId, err := d.Storage.ApplyDeltaAndLog(ctx, chatID, accName, val, txs)

			btn := api.NewInlineKeyboardButtonData("↩️ Откатить это изменение", fmt.Sprintf("undo:%d", txsId))
			kb := api.NewInlineKeyboardMarkup(api.NewInlineKeyboardRow(btn))

			reply := msgs.T(
				msgs.BalanceUpdated,
				strconv.FormatFloat(val, 'f', 2, 64),
				accName,
				strconv.FormatFloat(newBalance, 'f', 2, 64),
			)

			msgOK := api.NewMessage(chatID, reply)
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

func parseSlash(s string) (cmd, args string) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "/") {
		return "", ""
	}
	s = s[1:]

	i := strings.IndexByte(s, ' ')
	if i < 0 {
		cmd = s
		return cmd, ""
	}
	cmd, args = s[:i], strings.TrimSpace(s[i+1:])

	if at := strings.IndexByte(cmd, '@'); at >= 0 {
		cmd = cmd[:at]
	}
	return cmd, args
}

