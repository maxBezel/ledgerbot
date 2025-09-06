package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
		acc.Name, acc.ChatId, acc.Balance, acc.CreatedAt.Format(time.RFC3339Nano),
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
