package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
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

type Handler struct {
	logger         *slog.Logger
	accountService AccountService
}

func New(logger *slog.Logger, accountService AccountService) *Handler {
	return &Handler{
		logger:         logger,
		accountService: accountService,
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
