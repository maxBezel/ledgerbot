package commands

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	api "github.com/OvyFlash/telegram-bot-api"
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

			val, err := EvalQalc(ctx, expression, 20)
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
				formatAmount(val),
				accName,
				note,
				formatAmount(newBalance),
			)

			msgOK := api.NewMessage(chatID, reply)
			msgOK.ReplyMarkup = kb
			_, _ = d.Bot.Send(msgOK)

			return nil
		},
	}
}

func EvalQalc(ctx context.Context, raw string, precision int) (float64, error) {
	expr := strings.TrimSpace(raw)

	path, err := exec.LookPath("qalc")
	if err != nil {
		return 0, fmt.Errorf("qalc not found in PATH: %w", err)
	}

	precArg := fmt.Sprintf("prec %d", precision)
	args := []string{"--terse", "--set", precArg, "--", expr}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, path, args...)
	env := os.Environ()
	env = append(env, "LC_ALL=C", "LANG=C", "LC_NUMERIC=C")
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("qalc run error: %v (stderr=%q)", err, strings.TrimSpace(stderr.String()))
	}

	res := strings.TrimSpace(stdout.String())
	res = strings.ReplaceAll(res, "−", "-")
	res = strings.ReplaceAll(res, " ", "")
	res = strings.ReplaceAll(res, "\u2009", "")
	res = strings.ReplaceAll(res, "\u202F", "")

	v, err := strconv.ParseFloat(res, 64)
	if err != nil {
		return 0, fmt.Errorf("parse qalc result %q failed for expr %q", res, expr)
	}
	return v, nil
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

