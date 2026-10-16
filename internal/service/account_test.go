package service

import (
	"context"
	"errors"
	"testing"
	"wallet-service/internal/models"
)

const validID = "11111111-1111-1111-1111-111111111111"
const otherID = "22222222-2222-2222-2222-222222222222"

var errBoom = errors.New("boom") // произвольная ошибка storage

type mockAccountStorage struct {
	createNewAccountFunc func(ctx context.Context, userID string) (models.Account, error)
	getAccountFunc       func(ctx context.Context, id string) (models.Account, error)
	depositFunc          func(ctx context.Context, accountID, operationID string, amount int64) error
	transferFunc         func(ctx context.Context, fromID, toID, operationID string, amount int64) error
}

func (m *mockAccountStorage) CreateNewAccount(ctx context.Context, userID string) (models.Account, error) {
	if m.createNewAccountFunc == nil {
		panic("CreateNewAccount was not expected to be called in this test")
	}
	return m.createNewAccountFunc(ctx, userID)
}

func (m *mockAccountStorage) GetAccount(ctx context.Context, id string) (models.Account, error) {
	if m.getAccountFunc == nil {
		panic("GetAccount was not expected to be called in this test")
	}
	return m.getAccountFunc(ctx, id)
}

func (m *mockAccountStorage) Deposit(ctx context.Context, accountID, operationID string, amount int64) error {
	if m.depositFunc == nil {
		panic("Deposit was not expected to be called in this test")
	}
	return m.depositFunc(ctx, accountID, operationID, amount)
}

func (m *mockAccountStorage) Transfer(ctx context.Context, fromID, toID, operationID string, amount int64) error {
	if m.transferFunc == nil {
		panic("Transfer was not expected to be called in this test")
	}
	return m.transferFunc(ctx, fromID, toID, operationID, amount)
}

func TestAccountService_CreateNewAccount_InvalidUserID(t *testing.T) {
	srv := NewAccountService(&mockAccountStorage{})

	_, err := srv.CreateNewAccount(context.Background(), "")
	if !errors.Is(err, models.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestAccountService_CreateNewAccount_Success(t *testing.T) {
	mock := &mockAccountStorage{
		createNewAccountFunc: func(ctx context.Context, userID string) (models.Account, error) {
			return models.Account{ID: validID, UserID: userID}, nil
		},
	}
	srv := NewAccountService(mock)

	acc, err := srv.CreateNewAccount(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if acc.ID != validID {
		t.Errorf("expected id %q, got %q", validID, acc.ID)
	}
}

func TestAccountService_CreateNewAccount_StorageError(t *testing.T) {
	mock := &mockAccountStorage{
		createNewAccountFunc: func(ctx context.Context, userID string) (models.Account, error) {
			return models.Account{}, errBoom
		},
	}
	srv := NewAccountService(mock)

	_, err := srv.CreateNewAccount(context.Background(), "user-1")
	if !errors.Is(err, errBoom) {
		t.Errorf("expected errBoom, got %v", err)
	}
}

func TestAccountService_GetAccount_InvalidID(t *testing.T) {
	srv := NewAccountService(&mockAccountStorage{})

	_, err := srv.GetAccount(context.Background(), "abc")
	if !errors.Is(err, models.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestAccountService_GetAccount_NotFound(t *testing.T) {
	mock := &mockAccountStorage{
		getAccountFunc: func(ctx context.Context, id string) (models.Account, error) {
			return models.Account{}, models.ErrAccountNotFound
		},
	}
	srv := NewAccountService(mock)

	_, err := srv.GetAccount(context.Background(), validID)
	if !errors.Is(err, models.ErrAccountNotFound) {
		t.Errorf("expected ErrAccountNotFound, got %v", err)
	}
}

func TestAccountService_GetAccount_Success(t *testing.T) {
	mock := &mockAccountStorage{
		getAccountFunc: func(ctx context.Context, id string) (models.Account, error) {
			return models.Account{ID: id, Balance: 500}, nil
		},
	}
	srv := NewAccountService(mock)

	acc, err := srv.GetAccount(context.Background(), validID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if acc.Balance != 500 {
		t.Errorf("expected balance 500, got %d", acc.Balance)
	}
}

func TestAccountService_Deposit_InvalidAccountID(t *testing.T) {
	srv := NewAccountService(&mockAccountStorage{})

	err := srv.Deposit(context.Background(), "abc", 100)
	if !errors.Is(err, models.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestAccountService_Deposit_ZeroAmount(t *testing.T) {
	srv := NewAccountService(&mockAccountStorage{})

	err := srv.Deposit(context.Background(), validID, 0)
	if !errors.Is(err, models.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestAccountService_Deposit_NegativeAmount(t *testing.T) {
	srv := NewAccountService(&mockAccountStorage{})

	err := srv.Deposit(context.Background(), validID, -50)
	if !errors.Is(err, models.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestAccountService_Deposit_AccountNotFound(t *testing.T) {
	mock := &mockAccountStorage{
		depositFunc: func(ctx context.Context, accountID, operationID string, amount int64) error {
			return models.ErrAccountNotFound
		},
	}
	srv := NewAccountService(mock)

	err := srv.Deposit(context.Background(), validID, 100)
	if !errors.Is(err, models.ErrAccountNotFound) {
		t.Errorf("expected ErrAccountNotFound, got %v", err)
	}
}

func TestAccountService_Deposit_Success(t *testing.T) {
	mock := &mockAccountStorage{
		depositFunc: func(ctx context.Context, accountID, operationID string, amount int64) error {
			if amount != 100 {
				t.Errorf("storage got amount %d, want 100", amount)
			}
			return nil
		},
	}
	srv := NewAccountService(mock)

	err := srv.Deposit(context.Background(), validID, 100)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestAccountService_Transfer_InvalidFromID(t *testing.T) {
	srv := NewAccountService(&mockAccountStorage{})

	err := srv.Transfer(context.Background(), "abc", otherID, 100)
	if !errors.Is(err, models.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestAccountService_Transfer_InvalidToID(t *testing.T) {
	srv := NewAccountService(&mockAccountStorage{})

	err := srv.Transfer(context.Background(), validID, "abc", 100)
	if !errors.Is(err, models.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestAccountService_Transfer_SameAccount(t *testing.T) {
	srv := NewAccountService(&mockAccountStorage{})

	err := srv.Transfer(context.Background(), validID, validID, 100)
	if !errors.Is(err, models.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestAccountService_Transfer_SameAccountDifferentCase(t *testing.T) {
	srv := NewAccountService(&mockAccountStorage{})

	// тот же UUID, записанный по-другому (проверяет нормализацию)
	err := srv.Transfer(context.Background(), "AAAAAAAA-AAAA-AAAA-AAAA-AAAAAAAAAAAA", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", 100)
	if !errors.Is(err, models.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestAccountService_Transfer_InvalidAmount(t *testing.T) {
	srv := NewAccountService(&mockAccountStorage{})

	err := srv.Transfer(context.Background(), validID, otherID, 0)
	if !errors.Is(err, models.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestAccountService_Transfer_NotEnoughMoney(t *testing.T) {
	mock := &mockAccountStorage{
		transferFunc: func(ctx context.Context, fromID, toID, operationID string, amount int64) error {
			return models.ErrAccountNotEnoughMoney
		},
	}
	srv := NewAccountService(mock)

	err := srv.Transfer(context.Background(), validID, otherID, 100)
	if !errors.Is(err, models.ErrAccountNotEnoughMoney) {
		t.Errorf("expected ErrAccountNotEnoughMoney, got %v", err)
	}
}

func TestAccountService_Transfer_Success(t *testing.T) {
	mock := &mockAccountStorage{
		transferFunc: func(ctx context.Context, fromID, toID, operationID string, amount int64) error {
			if fromID != validID || toID != otherID || amount != 100 {
				t.Errorf("storage got (%s, %s, %d), want (%s, %s, 100)", fromID, toID, amount, validID, otherID)
			}
			return nil
		},
	}
	srv := NewAccountService(mock)

	err := srv.Transfer(context.Background(), validID, otherID, 100)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
