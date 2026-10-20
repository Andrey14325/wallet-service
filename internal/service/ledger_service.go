package service

import (
	"context"
	"fmt"
	"wallet-service/internal/models"

	"github.com/google/uuid"
)

type LedgerStorage interface {
	ListEntries(ctx context.Context, accountID string, cursor *models.Cursor, limit int) ([]models.LedgerEntry, error)
}

type LedgerService struct {
	storage LedgerStorage
}

func NewLedgerService(storage LedgerStorage) *LedgerService {
	return &LedgerService{storage: storage}
}

const (
	defaultLimit = 50
	maxLimit     = 100
)

func (s *LedgerService) ListEntries(ctx context.Context, accountID string, cursor *models.Cursor, limit int) ([]models.LedgerEntry, *models.Cursor, error) {
	if _, err := uuid.Parse(accountID); err != nil {
		return nil, nil, fmt.Errorf("invalid account id: %w", models.ErrValidation)
	}

	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	entries, err := s.storage.ListEntries(ctx, accountID, cursor, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list entries: %w", err)
	}

	var next *models.Cursor
	if len(entries) > limit {
		entries = entries[:limit]
		last := entries[len(entries)-1]
		next = &models.Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}

	return entries, next, nil
}
