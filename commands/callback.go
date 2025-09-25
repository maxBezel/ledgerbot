package commands

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	api "github.com/OvyFlash/telegram-bot-api"
	msgs "github.com/maxBezel/ledgerbot/internal/messages"
)

func HandleCallback(ctx context.Context, d Deps, cq *api.CallbackQuery) {
	data := cq.Data
	if strings.HasPrefix(data, "undo:") {
		handleUndo(ctx, d, cq, data)
	} else if strings.HasPrefix(data, "statement:") {
		handleStatement(ctx, d, cq, data)
	}
	_ = answerCB(d.Bot, cq, "Unknown action", true)
}

func handleUndo(ctx context.Context, d Deps, cq *api.CallbackQuery, data string) {
	parts := strings.Split(strings.TrimPrefix(data, "undo:"), ",")
	txID, err := strconv.Atoi(parts[0])
	accName := parts[1]
	if err == nil {
		newBalance, delta, err := d.Storage.RevertTransaction(ctx, int64(txID))
		if err != nil {
			_ = answerCB(d.Bot, cq, msgs.T(msgs.UnsuccessfulOperation), true)
			return
		}
		_ = answerCB(d.Bot, cq, "Transaction reverted", false)

		edit := api.NewEditMessageText(cq.Message.Chat.ID, cq.Message.MessageID,
			cq.Message.Text+"\n\nДанное изменение отменено ✅")
		_, _ = d.Bot.Send(edit)

		reply_msg := msgs.T(
			msgs.BalanceReverted,
			formatAmount(-delta),
			accName,
			formatAmount(newBalance),
		)

		reply := api.NewMessage(cq.Message.Chat.ID, reply_msg)
		reply.ReplyParameters.MessageID = cq.Message.MessageID
		_, _ = d.Bot.Send(reply)
		return
	}
}

func handleStatement(ctx context.Context, d Deps, cq *api.CallbackQuery, data string) {
	chatID := cq.Message.Chat.ID
	if s := strings.TrimPrefix(data, "statement:"); s != data {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			chatID = v
		}
	}

	answerCB(d.Bot, cq, "Готовлю выписку…", false)

	ts := time.Now().UTC().Format("20060102_150405Z")
	filename := fmt.Sprintf("statement_%d_%s.csv", chatID, ts)

	if err := d.Storage.WriteTransactionsCsv(ctx, chatID, filename); err != nil {
		answerCB(d.Bot, cq, "Не удалось сформировать выписку: "+err.Error(), true)
		return
	}
	defer os.Remove(filename)

	doc := api.NewDocument(chatID, api.FilePath(filename))
	doc.Caption = "Выписка по счетам"
	if _, err := d.Bot.Send(doc); err != nil {
		answerCB(d.Bot, cq, "Не удалось отправить файл: "+err.Error(), true)
		return
	}
}

func answerCB(bot Bot, cq *api.CallbackQuery, text string, alert bool) error {
	cb := api.NewCallback(cq.ID, text)
	if alert {
		cb.ShowAlert = true
	}

	_, err := bot.Send(cb)
	return err
}

