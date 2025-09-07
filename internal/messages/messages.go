package messages

import (
	"fmt"
	"log"
)

type ID string

const (
	Start                 ID = "start"
	AccountCreated        ID = "account_created"
	NoAccountName         ID = "no_name"
	NoAccountsYet         ID = "no_accounts"
	NoExpression          ID = "no_expression"
	AccDoesNotExist       ID = "acc_does_not_exist"
	AccAlreadyExist       ID = "acc_already_exist"
	AccRemoved            ID = "acc_removed"
	BalanceUpdated        ID = "balance_updated"
	UnsuccessfulOperation ID = "unsuccessful_operation"
)

var rus = map[ID]string{
	Start:                 "Привет",
	AccountCreated:        "Счет %s создан",
	NoAccountName:         "Не указано имя счета. Пример: /<команда> <имя_счета>",
	NoAccountsYet:         "У вас пока нет счетов. Используйте /new <имя_счета>",
	NoExpression:          "Неверный формат комманды. Используйте /<имя счета> <выражение> [комментарий]",
	AccDoesNotExist:       "Счет %s не существует.",
	AccAlreadyExist:       "Счет с таким именем уже существует",
	AccRemoved:            "Счет %s удален",
	BalanceUpdated:        "Запомнил %s на счет %s \nБаланс: %s",
	UnsuccessfulOperation: "Неудалось выполнить операцию",
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
