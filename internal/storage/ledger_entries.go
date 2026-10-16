package storage

import "github.com/jackc/pgx/v5/pgxpool"

type LedgerEntries struct {
	pool *pgxpool.Pool
}

func NewLedgerEntries(pool *pgxpool.Pool) *LedgerEntries {
	return &LedgerEntries{pool: pool}
}
