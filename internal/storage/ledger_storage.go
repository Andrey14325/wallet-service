package storage

import (
	"context"
	"fmt"
	"wallet-service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LedgerStorage struct {
	pool *pgxpool.Pool
}

func NewLedgerEntries(pool *pgxpool.Pool) *LedgerStorage {
	return &LedgerStorage{pool: pool}
}

func (s *LedgerStorage) ListEntries(ctx context.Context, accountID string, cursor *models.Cursor, limit int) ([]models.LedgerEntry, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM accounts WHERE account_id = $1
		)`,
		accountID,
	).Scan(&exists)

	if err != nil {
		return nil, fmt.Errorf("check account: %w", err)
	}

	if !exists {
		return nil, models.ErrAccountNotFound
	}

	var pgRows pgx.Rows

	if cursor != nil {
		pgRowCurs, err := s.pool.Query(ctx, `
			SELECT id, account_id, operation_id, amount, created_at
			FROM ledger_entries
			WHERE account_id = $1
			AND (created_at, id) < ($2, $3)
			ORDER BY created_at DESC, id DESC
			LIMIT $4`, accountID, cursor.CreatedAt, cursor.ID, limit)
		if err != nil {
			return nil, fmt.Errorf("get ledger: %w", err)
		}
		pgRows = pgRowCurs
	} else {
		pgRowNoCurs, err := s.pool.Query(ctx, `
			SELECT id, account_id, operation_id, amount, created_at
			FROM ledger_entries
			WHERE account_id = $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2`, accountID, limit)
		if err != nil {
			return nil, fmt.Errorf("get ledger: %w", err)
		}
		pgRows = pgRowNoCurs
	}

	defer pgRows.Close()

	entries := make([]models.LedgerEntry, 0, limit)

	for pgRows.Next() {
		var entry models.LedgerEntry

		err := pgRows.Scan(
			&entry.ID,
			&entry.AccountID,
			&entry.OperationID,
			&entry.Amount,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ledger entry: %w", err)
		}

		entries = append(entries, entry)
	}
	if err := pgRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ledger entries: %w", err)
	}

	return entries, nil
}
