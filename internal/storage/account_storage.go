package storage

import (
	"context"
	"errors"
	"fmt"
	"wallet-service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountStorage struct {
	pool *pgxpool.Pool
}

func NewAccountStorage(pool *pgxpool.Pool) *AccountStorage {
	return &AccountStorage{pool: pool}
}

func (s *AccountStorage) CreateNewAccount(ctx context.Context, userID string) (models.Account, error) {
	var acc models.Account
	err := s.pool.QueryRow(ctx, `
		INSERT INTO accounts (user_id, balance)
		VALUES ($1, 0)
		RETURNING account_id, user_id, balance, created_at, updated_at`, userID,
	).Scan(&acc.ID, &acc.UserID, &acc.Balance, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		return models.Account{}, fmt.Errorf("create account: %w", err)
	}
	return acc, nil
}

func (s *AccountStorage) GetAccount(ctx context.Context, id string) (models.Account, error) {
	pgRow := s.pool.QueryRow(ctx, `
		SELECT account_id, user_id, balance, created_at, updated_at
		FROM accounts
		WHERE account_id = $1`, id)

	var acc models.Account
	err := pgRow.Scan(&acc.ID, &acc.UserID, &acc.Balance, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Account{}, fmt.Errorf("get account: %w", models.ErrAccountNotFound)
		}
		return models.Account{}, fmt.Errorf("get account: %w", err)
	}

	return acc, nil
}

func (s *AccountStorage) Deposit(ctx context.Context, accountID, operationID string, amount int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("deposit begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		INSERT INTO ledger_entries (account_id, operation_id, amount)
		VALUES ($1, $2, $3)
		ON CONFLICT (operation_id, account_id) DO NOTHING`,
		accountID, operationID, amount,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return fmt.Errorf("deposit: %w", models.ErrAccountNotFound)
		}
		return fmt.Errorf("create ledger entry: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return nil // повтор операции, ничего не меняем
	}

	tag, err = tx.Exec(ctx, `
		UPDATE accounts
		SET balance = balance + $1, updated_at = NOW()
		WHERE account_id = $2`,
		amount, accountID,
	)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("deposit: %w", models.ErrAccountNotFound)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("deposit commit: %w", err)
	}
	return nil
}

func (s *AccountStorage) Transfer(ctx context.Context, fromID, toID, operationID string, amount int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("transfer begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	firstID, secondID := fromID, toID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	balances := make(map[string]int64, 2)
	for _, id := range []string{firstID, secondID} {
		balance, err := lockAccount(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("transfer: %w", err)
		}
		balances[id] = balance
	}

	var total, matching int64
	err = tx.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (
				WHERE (account_id = $2 AND amount = $4)
				OR (account_id = $3 AND amount = $5)
			)
		FROM ledger_entries
		WHERE operation_id = $1`,
		operationID, fromID, toID, -amount, amount,
	).Scan(&total, &matching)
	if err != nil {
		return fmt.Errorf("check operation: %w", err)
	}

	switch {
	case total == 0:
		// новая операция, идём дальше
	case total == 2 && matching == 2:
		return nil // тот же перевод, повтор
	default:
		return fmt.Errorf("transfer: %w", models.ErrIdempotencyConflict)
	}

	if balances[fromID] < amount {
		return fmt.Errorf("transfer: %w", models.ErrAccountNotEnoughMoney)
	}

	_, err = tx.Exec(ctx, `
		UPDATE accounts
		SET balance = balance - $1, updated_at = NOW()
		WHERE account_id = $2`, amount, fromID)
	if err != nil {
		return fmt.Errorf("debit sender: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE accounts
		SET balance = balance + $1, updated_at = NOW()
		WHERE account_id = $2`, amount, toID)
	if err != nil {
		return fmt.Errorf("credit receiver: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (account_id, operation_id, amount)
		VALUES ($1, $2, $3)`, fromID, operationID, -amount)
	if err != nil {
		return fmt.Errorf("ledger entry for sender: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (account_id, operation_id, amount)
		VALUES ($1, $2, $3)`, toID, operationID, amount)
	if err != nil {
		return fmt.Errorf("ledger entry for receiver: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("transfer commit: %w", err)
	}
	return nil
}

func lockAccount(ctx context.Context, tx pgx.Tx, id string) (int64, error) {
	var balance int64
	err := tx.QueryRow(ctx, `
		SELECT balance
		FROM accounts
		WHERE account_id = $1
		FOR UPDATE`, id).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, models.ErrAccountNotFound
		}
		return 0, fmt.Errorf("lock account: %w", err)
	}
	return balance, nil
}
