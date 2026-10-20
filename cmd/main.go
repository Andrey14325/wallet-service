package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"wallet-service/internal/config"
	"wallet-service/internal/handlers"
	"wallet-service/internal/reconciler"
	"wallet-service/internal/server"
	"wallet-service/internal/service"
	"wallet-service/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	cfg := config.ParseConfig()

	pool, err := storage.NewPostgresStorage(ctx, cfg.Database)
	if err != nil {
		logger.Error("error database connection", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	accountStorage := storage.NewAccountStorage(pool)
	accountService := service.NewAccountService(accountStorage)
	ledgerStorage := storage.NewLedgerEntries(pool)
	ledgerService := service.NewLedgerService(ledgerStorage)

	handle := handlers.New(logger, accountService, ledgerService)

	serv, err := server.NewServer(logger, cfg.ServAddress)
	if err != nil {
		logger.Error("error creating server", slog.Any("error", err))
		os.Exit(1)
	}
	server.SetupRoutes(serv, handle)

	checker := reconciler.NewAccountCheck(accountStorage, logger, cfg.Interval, 100)
	go checker.Run(ctx)

	if err := serv.RunServer(ctx); err != nil {
		logger.Error("error running server", slog.Any("error", err))
		os.Exit(1)
	}
}
