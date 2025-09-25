package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	api "github.com/OvyFlash/telegram-bot-api"
	"github.com/expr-lang/expr"
	"github.com/maxBezel/ledgerbot/exprsplit"
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
			expression, note, err := exprsplit.SplitExprAndComment(args)
			if err != nil {
				if err.Error() != "no valid math expression found" || accName != "" {
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
				_, _ = d.Bot.Send(api.NewMessage(chatID, msgs.T(msgs.InvalidExpression)))
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

			btn := api.NewInlineKeyboardButtonData("↩️ Откатить это изменение", fmt.Sprintf("undo:%d,%s", txsId, accName))
			kb := api.NewInlineKeyboardMarkup(api.NewInlineKeyboardRow(btn))

			if note == "" {
				note = "Нет"
			}
			reply := msgs.T(
				msgs.BalanceUpdated,
				strconv.FormatFloat(val, 'f', 2, 64),
				accName,
				note,
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

