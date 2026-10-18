package reconciler

import (
	"context"
	"log/slog"
	"time"
	"wallet-service/internal/models"
)

type AccountStorage interface {
	CheckBalances(ctx context.Context, afterID string, limit int) ([]models.BalanceCheck, error)
}

type AccountCheck struct {
	storage  AccountStorage
	logger   *slog.Logger
	interval time.Duration
	packSize int
}

func NewAccountCheck(storage AccountStorage, logger *slog.Logger, interval time.Duration, packSize int) *AccountCheck {
	return &AccountCheck{storage: storage,
		logger:   logger,
		interval: interval,
		packSize: packSize}
}

func (c *AccountCheck) Run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			afterID := "00000000-0000-0000-0000-000000000000"

			for {
				balances, err := c.storage.CheckBalances(ctx, afterID, c.packSize)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					c.logger.Error("check balances failed", "error", err)
					break
				}

				c.checkBatch(balances)

				if len(balances) < c.packSize {
					break
				}

				afterID = balances[len(balances)-1].AccountID
			}
		}
	}
}

func (c *AccountCheck) checkBatch(balances []models.BalanceCheck) {
	for _, b := range balances {
		if b.Balance != b.LedgerSum {
			c.logger.Error(
				"balance mismatch",
				"account_id", b.AccountID,
				"balance", b.Balance,
				"ledger_sum", b.LedgerSum,
			)
		}
	}
}
