package storage

import (
	"context"
	"fmt"
	"wallet-service/internal/models"
)

func (s *AccountStorage) CheckBalances(ctx context.Context, afterID string, limit int) ([]models.BalanceCheck, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.account_id, a.balance, COALESCE(SUM(l.amount), 0)::bigint AS ledger_sum
		FROM (
			SELECT account_id, balance
			FROM accounts
			WHERE account_id > $1
			ORDER BY account_id
			LIMIT $2
		) a
		LEFT JOIN ledger_entries l ON l.account_id = a.account_id
		GROUP BY a.account_id, a.balance
		ORDER BY a.account_id
	`, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("check balance: %w", err)
	}
	defer rows.Close()

	balances := make([]models.BalanceCheck, 0)

	for rows.Next() {
		var balance models.BalanceCheck

		err := rows.Scan(
			&balance.AccountID,
			&balance.Balance,
			&balance.LedgerSum,
		)
		if err != nil {
			return nil, fmt.Errorf("scan balance: %w", err)
		}

		balances = append(balances, balance)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate balances: %w", err)
	}

	return balances, nil
}
