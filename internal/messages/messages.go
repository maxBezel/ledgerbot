package messages

import (
	"fmt"
	"log"
)

type ID string

const (
	Start           ID = "start"
	AccountCreated  ID = "account_created"
	NoAccountName   ID = "no_name"
	NoAccountsYet   ID = "no_accounts"
	AccDoesNotExist ID = "acc_does_not_exist"
	AccRemoved      ID = "acc_removed"
	BalanceUpdated  ID = "balance_updated"
)

var rus = map[ID]string{
	Start:           "Привет",
	AccountCreated:  "Аккаунт %s создан",
	NoAccountName:   "Не указано имя счета. Пример: /<команда> <имя_счета>",
	NoAccountsYet:   "У вас пока нет счетов. Используйте /new <имя_счета>",
	AccDoesNotExist: "Счет %s не существует.",
	AccRemoved:      "Аккаунт %s удален",
	BalanceUpdated:  "Запомнил %s на счет %s \nБаланс: %s",
}

func T(id ID, args ...any) string {
	reply, ok := rus[id]
	if !ok {
		log.Printf("missing text %s", string(id))
		return "Ошибка"
	}

	if len(args) == 0 {
		return reply
	}

	return fmt.Sprintf(reply, args...)
}
