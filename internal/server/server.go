package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

var (
	ErrAddressMissing = errors.New("address is missing")
)

const (
	readHeaderTimeout = 2 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
	shutdownTimeout   = 60 * time.Second
	maxHeaderBytes    = 1 << 20
)

type Server struct {
	serv   *http.Server
	mux    *http.ServeMux
	logger *slog.Logger
}

func NewServer(logger *slog.Logger, address string) (*Server, error) {
	if address == "" {
		return nil, ErrAddressMissing
	}

	mux := http.NewServeMux()

	return &Server{
		serv: &http.Server{
			Addr:              address,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			MaxHeaderBytes:    maxHeaderBytes,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		},
		mux:    mux,
		logger: logger,
	}, nil
}

func (s *Server) SetHandler(handler http.Handler) {
	s.serv.Handler = handler
}

func (s *Server) RunServer(ctx context.Context) error {
	serverErr := make(chan error, 1)
	s.logger.Info("server started", slog.String("addr", s.serv.Addr))
	go func() {
		serverErr <- s.serv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("listen and serve error: %w", err)

	case <-ctx.Done():
		s.logger.Info("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			shutdownTimeout,
		)
		defer cancel()

		if err := s.serv.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("graceful shutdown failed",
				slog.Any("error", err),
			)

			if closeErr := s.serv.Close(); closeErr != nil {
				s.logger.Error("forceful close failed",
					slog.Any("error", closeErr),
				)
			}

			return fmt.Errorf("graceful shutdown failed: %w", err)
		}

		s.logger.Info("graceful shutdown completed")
		return nil
	}
}

func (s *Server) GetMux() *http.ServeMux {
	return s.mux
}
