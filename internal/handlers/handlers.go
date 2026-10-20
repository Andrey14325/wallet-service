package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"wallet-service/internal/models"

	"github.com/google/uuid"
)

const maxBodySize = 1 << 20

type AccountService interface {
	CreateNewAccount(ctx context.Context, userID string) (models.Account, error)
	GetAccount(ctx context.Context, id string) (models.Account, error)
	Deposit(ctx context.Context, accountID string, amount int64) error
	Transfer(ctx context.Context, fromID, toID string, idemKey string, amount int64) error
}

type LedgerService interface {
	ListEntries(ctx context.Context, accountID string, cursor *models.Cursor, limit int) ([]models.LedgerEntry, *models.Cursor, error)
}

type Handler struct {
	logger         *slog.Logger
	accountService AccountService
	ledgerService  LedgerService
}

func New(logger *slog.Logger, accountService AccountService, ledgerService LedgerService) *Handler {
	return &Handler{
		logger:         logger,
		accountService: accountService,
		ledgerService:  ledgerService,
	}
}

func (h *Handler) CreateNewAccount(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req struct {
		UserID string `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		h.logger.Warn("invalid create account request", slog.String("error", err.Error()))
		writeError(w, http.StatusBadRequest, "invalid create account request")
		return
	}

	acc, err := h.accountService.CreateNewAccount(r.Context(), req.UserID)
	if err != nil {
		if errors.Is(err, models.ErrValidation) {
			h.logger.Warn("create account validation failed", slog.String("error", err.Error()))
			writeError(w, http.StatusBadRequest, "invalid request")
			return
		}
		h.logger.Error("error creating account", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.writeJSON(w, http.StatusCreated, acc)
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.logger.Warn("invalid account id",
			slog.String("id", r.PathValue("id")),
			slog.String("error", err.Error()),
		)
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	acc, err := h.accountService.GetAccount(r.Context(), id.String())
	if err != nil {
		if errors.Is(err, models.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}
		if errors.Is(err, models.ErrValidation) {
			writeError(w, http.StatusBadRequest, "invalid account id")
			return
		}
		h.logger.Error("error get account", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.writeJSON(w, http.StatusOK, acc)
}

func (h *Handler) Deposit(w http.ResponseWriter, r *http.Request) {
	accID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.logger.Warn("invalid account id",
			slog.String("id", r.PathValue("id")),
			slog.String("error", err.Error()),
		)
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req struct {
		Amount int64 `json:"amount"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		h.logger.Warn("invalid deposit request", slog.String("error", err.Error()))
		writeError(w, http.StatusBadRequest, "invalid deposit request")
		return
	}

	err = h.accountService.Deposit(r.Context(), accID.String(), req.Amount)
	if err != nil {
		if errors.Is(err, models.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}
		if errors.Is(err, models.ErrValidation) {
			writeError(w, http.StatusBadRequest, "invalid request")
			return
		}
		h.logger.Error("error deposit", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("Idempotency-Key")

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req models.Transfer

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		h.logger.Warn("invalid transfer request", slog.String("error", err.Error()))
		writeError(w, http.StatusBadRequest, "invalid transfer request")
		return
	}

	err := h.accountService.Transfer(r.Context(), req.FromID, req.ToID, key, req.Amount)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrValidation):
			h.logger.Warn("transfer: invalid request", slog.String("error", err.Error()))
			writeError(w, http.StatusBadRequest, "invalid request")
		case errors.Is(err, models.ErrAccountNotFound):
			h.logger.Warn("transfer: account not found", slog.String("error", err.Error()))
			writeError(w, http.StatusNotFound, "account not found")
		case errors.Is(err, models.ErrAccountNotEnoughMoney):
			h.logger.Warn("transfer: not enough money", slog.String("error", err.Error()))
			writeError(w, http.StatusUnprocessableEntity, "not enough money")
		case errors.Is(err, models.ErrIdempotencyConflict):
			h.logger.Warn("transfer: idempotency key already used", slog.String("error", err.Error()))
			writeError(w, http.StatusConflict, "idempotency key already used")
		default:
			h.logger.Error("error transfer", slog.String("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ListEntries(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.logger.Warn("invalid account id",
			slog.String("id", r.PathValue("id")),
			slog.String("error", err.Error()),
		)
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	limit := 0
	if s := r.URL.Query().Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = n
	}

	cursor, err := decodeCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		h.logger.Warn("invalid cursor", slog.String("error", err.Error()))
		writeError(w, http.StatusBadRequest, "invalid cursor")
		return
	}

	entries, nextCursor, err := h.ledgerService.ListEntries(r.Context(), id.String(), cursor, limit)
	if err != nil {
		if errors.Is(err, models.ErrAccountNotFound) {
			h.logger.Warn("list_entries: account not found", slog.String("error", err.Error()))
			writeError(w, http.StatusNotFound, "account not found")
			return
		}
		if errors.Is(err, models.ErrValidation) {
			h.logger.Warn("list_entries: invalid request", slog.String("error", err.Error()))
			writeError(w, http.StatusBadRequest, "invalid request")
			return
		}
		h.logger.Error("error list entries", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	nextStr, err := encodeCursor(nextCursor)
	if err != nil {
		h.logger.Error("error encode cursor", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if entries == nil {
		entries = []models.LedgerEntry{}
	}

	type historyResponse struct {
		Items      []models.LedgerEntry `json:"items"`
		NextCursor string               `json:"next_cursor,omitempty"`
	}

	h.writeJSON(w, http.StatusOK, historyResponse{Items: entries, NextCursor: nextStr})
}

func encodeCursor(c *models.Cursor) (string, error) {
	if c == nil {
		return "", nil
	}
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decodeCursor(s string) (*models.Cursor, error) {
	if s == "" {
		return nil, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var c models.Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(c.ID); err != nil {
		return nil, err
	}
	if c.CreatedAt.IsZero() {
		return nil, errors.New("invalid time")
	}
	return &c, nil
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error("failed to write response", slog.String("error", err.Error()))
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(ErrorResponse{Error: message}); err != nil {
		slog.Error("failed to send JSON response", slog.String("error", err.Error()))
	}
}
