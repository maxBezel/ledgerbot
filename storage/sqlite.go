package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/maxBezel/ledgerbot/model"
)

type Storage struct {
	db *sql.DB
}

func New(path string) (*Storage, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("cant open database %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cant reach database %w", err)
	}

	return &Storage{db: db}, nil
}

func (storage *Storage) Init(ctx context.Context) error {
	_, _ = storage.db.ExecContext(ctx, `PRAGMA foreign_keys = ON;`)

	accountsQ := `
	CREATE TABLE IF NOT EXISTS accounts (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT    NOT NULL,
		chat_id    INTEGER NOT NULL,
		balance    REAL    NOT NULL DEFAULT 0,
		created_at TEXT    NOT NULL,
		UNIQUE(chat_id, name)
	);`

	txsQ := `
	CREATE TABLE IF NOT EXISTS account_txns (
	 	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	 	account_id  INTEGER NOT NULL
	 	REFERENCES accounts(id) ON DELETE CASCADE,
	 	amount      REAL    NOT NULL,
		expression	TEXT    NOT NULL,
	 	note        TEXT,
	 	created_at  TEXT    NOT NULL,
	 	created_by  INTEGER
	);`

	if _, err := storage.db.ExecContext(ctx, accountsQ); err != nil {
		return fmt.Errorf("Failed to create accounts table %w", err)
	}

	if _, err := storage.db.ExecContext(ctx, txsQ); err != nil {
		return fmt.Errorf("Failed to create txs table %w", err)
	}

	return nil
}

func (s *Storage) AddAccount(ctx context.Context, acc *model.Account) error {
	if acc == nil {
		return fmt.Errorf("nil account")
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO accounts(name, chat_id, balance, created_at)
		 VALUES(?, ?, ?, ?)`,
		acc.Name, acc.ChatId, acc.Balance, acc.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert account: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}

	acc.Id = int(id)
	return nil
}

func (s *Storage) AddTransaction(ctx context.Context, txs *model.Transaction) (int64, error) {
	if txs == nil {
		return 0, fmt.Errorf("nil transaction")
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO account_txns(account_id, amount, note, expression, created_at, created_by)
		 VALUES(?, ?, ?, ?, ?, ?)`,
		txs.AccountId, txs.Amount, txs.Note, txs.Expression, txs.CreatedAt, txs.CreatedBy,
	)
	if err != nil {
		return 0, fmt.Errorf("insert txs: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}

	txs.Id = int(id)
	return int64(id), nil
}

func (s *Storage) RemoveAccount(ctx context.Context, chatId int64, name string) error {
	q := `DELETE FROM accounts WHERE chat_id = ? AND name = ?`
	res, err := s.db.ExecContext(ctx, q, chatId, name)
	if err != nil {
		return fmt.Errorf("could not remove account %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("account not found")
	}

	return nil
}

func (s *Storage) GetAll(ctx context.Context, chatId int64) ([]string, error) {
	const q = `
		SELECT name
		FROM accounts
		WHERE chat_id = ?
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, q, chatId)
	if err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan account name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return names, nil
}

func (s *Storage) AdjustBalance(ctx context.Context, chatId int64, name string, delta float64) (float64, error) {
	if name == "" {
		return 0, fmt.Errorf("empty account name")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	res, err := tx.ExecContext(ctx,
		`UPDATE accounts
		 SET balance = balance + ?
		 WHERE chat_id = ? AND name = ?`,
		delta, chatId, name,
	)
	if err != nil {
		return 0, fmt.Errorf("update balance: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return 0, fmt.Errorf("account not found")
	}

	var newBalance float64
	if err := tx.QueryRowContext(ctx,
		`SELECT balance FROM accounts WHERE chat_id = ? AND name = ?`,
		chatId, name,
	).Scan(&newBalance); err != nil {
		return 0, fmt.Errorf("select new balance: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return newBalance, nil
}

func (s *Storage) Exists(ctx context.Context, chatId int64, name string) (bool, error) {
	q := `SELECT COUNT(*) FROM accounts WHERE chat_id = ? AND name = ?`

	var count int
	if err := s.db.QueryRowContext(ctx, q, chatId, name).Scan(&count); err != nil {
		return false, fmt.Errorf("cant check if account exists %w", err)
	}

	return count > 0, nil
}


func (s *Storage) GetAccountID(ctx context.Context, chatID int64, name string) (int, error) {
	const q = `SELECT id FROM accounts WHERE chat_id = ? AND name = ? LIMIT 1`

	var id int
	if err := s.db.QueryRowContext(ctx, q, chatID, name).Scan(&id); err != nil {
		return 0, fmt.Errorf("select account id: %w", err)
	}
	return id, nil
}

func (s *Storage) RevertTransaction(ctx context.Context, txsId int) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var (
		amount    float64
		accountID int
	)

	const sel = `SELECT amount, account_id FROM account_txns WHERE id = ?`
	if err = tx.QueryRowContext(ctx, sel, txsId).Scan(&amount, &accountID); err != nil {
		return fmt.Errorf("select tx: %w", err)
	}

	const upd = `UPDATE accounts SET balance = balance - ? WHERE id = ?`
	res, err := tx.ExecContext(ctx, upd, amount, accountID)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("account not found for transaction")
	}

	const del = `DELETE FROM account_txns WHERE id = ?`
	res, err = tx.ExecContext(ctx, del, txsId)
	if err != nil {
		return fmt.Errorf("delete tx: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("transaction already deleted")
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

type AccountBalance struct {
	Name    string
	Balance float64
}

func (s *Storage) ListAccountBalances(ctx context.Context, chatID int64) ([]AccountBalance, error) {
	const q = `
		SELECT name, balance
		FROM accounts
		WHERE chat_id = ?
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, q, chatID)
	if err != nil {
		return nil, fmt.Errorf("query balances: %w", err)
	}
	defer rows.Close()

	var out []AccountBalance
	for rows.Next() {
		var ab AccountBalance
		if err := rows.Scan(&ab.Name, &ab.Balance); err != nil {
			return nil, fmt.Errorf("scan balance: %w", err)
		}
		out = append(out, ab)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return out, nil
}
