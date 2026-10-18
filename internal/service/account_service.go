package service

import (
	"context"
	"fmt"
	"strings"
	"wallet-service/internal/models"

	"github.com/google/uuid"
)

type AccountStorage interface {
	CreateNewAccount(ctx context.Context, userID string) (models.Account, error)
	GetAccount(ctx context.Context, id string) (models.Account, error)
	Deposit(ctx context.Context, accountID, operationID string, amount int64) error
	Transfer(ctx context.Context, fromID, toID, operationID string, amount int64) error
}

type AccountService struct {
	storage AccountStorage
}

func NewAccountService(storage AccountStorage) *AccountService {
	return &AccountService{storage: storage}
}

func (s *AccountService) CreateNewAccount(ctx context.Context, userID string) (models.Account, error) {
	if strings.TrimSpace(userID) == "" {
		return models.Account{}, fmt.Errorf("userID is required: %w", models.ErrValidation)
	}

	acc, err := s.storage.CreateNewAccount(ctx, userID)
	if err != nil {
		return models.Account{}, fmt.Errorf("create account: %w", err)
	}

	return acc, nil
}

func (s *AccountService) GetAccount(ctx context.Context, id string) (models.Account, error) {
	if _, err := uuid.Parse(id); err != nil {
		return models.Account{}, fmt.Errorf("invalid account id: %w", models.ErrValidation)
	}

	acc, err := s.storage.GetAccount(ctx, id)
	if err != nil {
		return models.Account{}, fmt.Errorf("get account: %w", err)
	}

	return acc, nil
}

func (s *AccountService) Deposit(ctx context.Context, accountID string, amount int64) error {
	if _, err := uuid.Parse(accountID); err != nil {
		return fmt.Errorf("invalid account id: %w", models.ErrValidation)
	}
	if amount <= 0 {
		return fmt.Errorf("unacceptable amount: %w", models.ErrValidation)
	}
	operationID := uuid.NewString()
	err := s.storage.Deposit(ctx, accountID, operationID, amount)
	if err != nil {
		return fmt.Errorf("deposit: %w", err) //не делеаю проверку на разные типы ошибкок, так как тут на по сути все равно, и надо будет проверять в хендлере при ответе
	}

	return nil
}

func (s *AccountService) Transfer(ctx context.Context, fromID, toID, idemKey string, amount int64) error {
	from, err := uuid.Parse(fromID)
	if err != nil {
		return fmt.Errorf("invalid sender id: %w", models.ErrValidation)
	}
	to, err := uuid.Parse(toID)
	if err != nil {
		return fmt.Errorf("invalid receiver id: %w", models.ErrValidation)
	}
	key, err := uuid.Parse(idemKey)
	if err != nil {
		return fmt.Errorf("invalid idempotency key: %w", models.ErrValidation)
	}
	if from == to {
		return fmt.Errorf("transfer to yourself: %w", models.ErrValidation)
	}
	if amount <= 0 {
		return fmt.Errorf("unacceptable amount: %w", models.ErrValidation)
	}

	if err := s.storage.Transfer(ctx, from.String(), to.String(), key.String(), amount); err != nil {
		return fmt.Errorf("transfer: %w", err)
	}
	return nil
}
