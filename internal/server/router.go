package server

import (
	"net/http"
	"wallet-service/internal/handlers"
)

func SetupRoutes(serv *Server, h *handlers.Handler) {
	mux := serv.GetMux()

	mux.HandleFunc("POST /accounts", h.CreateNewAccount)
	mux.HandleFunc("GET /accounts/{id}", h.GetAccount)
	mux.HandleFunc("POST /accounts/{id}/deposit", h.Deposit)
	mux.HandleFunc("POST /transfers", h.Transfer)
	mux.HandleFunc("GET /accounts/{id}/history", h.ListEntries)

	var handler http.Handler = mux

	serv.SetHandler(handler)
}
