package commands

import (
	"context"
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	api "github.com/OvyFlash/telegram-bot-api"
)

func Get() Command {
	return Command{
		Name:        "get",
		Description: "Возвращает баланс всех аккаунтов",
		Handle: func(ctx context.Context, d Deps, msg *api.Message) error {
			chatID := msg.Chat.ID

			bals, err := d.Storage.ListAccountBalances(ctx, chatID)
			if err != nil {
				return err
			}
			if len(bals) == 0 {
				_, _ = d.Bot.Send(api.NewMessage(chatID, "У вас пока нет аккаунтов"))
				return nil
			}

			// Title: group name if present, otherwise "вас"
			who := "вас"
			if t := strings.TrimSpace(msg.Chat.Title); t != "" {
				who = html.EscapeString(t)
			}

			// Pre-format numbers and compute max width for padding
			formatted := make([]string, len(bals))
			maxw := 0
			for i, ab := range bals {
				s := formatAmount(ab.Balance)
				formatted[i] = s
				if w := utf8.RuneCountInString(s); w > maxw {
					maxw = w
				}
			}

			var b strings.Builder
			// Header
			fmt.Fprintf(&b, "<b>Средств на руках у %s:</b>\n", who)

			// Monospace aligned table
			b.WriteString("<pre>")
			for i, ab := range bals {
				amt := formatted[i]
				name := html.EscapeString(ab.Name)

				// left pad amount to maxw
				pad := maxw - utf8.RuneCountInString(amt)
				if pad > 0 {
					b.WriteString(strings.Repeat(" ", pad))
				}
				// amount + two spaces + name
				b.WriteString(amt)
				b.WriteString("  ")
				b.WriteString(name)
				if i < len(bals)-1 {
					b.WriteByte('\n')
				}
			}
			b.WriteString("</pre>")

			out := api.NewMessage(chatID, b.String())
			out.ParseMode = "HTML"

			// Optional: add the button under the message
			btn := api.NewInlineKeyboardButtonData("Получить выписку", "statement")
			out.ReplyMarkup = api.NewInlineKeyboardMarkup(api.NewInlineKeyboardRow(btn))

			_, _ = d.Bot.Send(out)
			return nil
		},
	}
}

// formatAmount prints a float as an integer if possible, otherwise with 2 decimals,
// and inserts thousands separators using U+2019 (’), e.g. 8’136’901.
func formatAmount(v float64) string {
	sep := '’' // U+2019 right single quotation mark
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}

	if math.Trunc(v) == v {
		// integer
		s := strconv.FormatInt(int64(v), 10)
		return sign + insertSep(s, sep)
	}

	// show 2 decimals, trim trailing zeros if you prefer:
	s := strconv.FormatFloat(v, 'f', 2, 64) // e.g. "12345.60"
	intPart, frac := s, ""
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		intPart, frac = s[:dot], s[dot+1:]
	}
	intPart = insertSep(intPart, sep)

	frac = strings.TrimRight(frac, "0")
	if frac == "" {
		return sign + intPart
	}
	return sign + intPart + "." + frac
}

// insertSep inserts a thousands separator into a base-10 integer string.
// s must be digits only (no sign).
func insertSep(s string, sep rune) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	first := n % 3
	if first == 0 {
		first = 3
	}
	var b strings.Builder
	b.WriteString(s[:first])
	for i := first; i < n; i += 3 {
		b.WriteRune(sep)
		b.WriteString(s[i : i+3])
	}
	return b.String()
}