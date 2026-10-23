package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

type contextKey string

const RequestIDKey contextKey = "request_id"

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.statusCode != 0 {
		return
	}
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) StatusCode() int {
	if rw.statusCode == 0 {
		return http.StatusOK
	}
	return rw.statusCode
}

func (h *Handler) RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				h.logger.Error("panic recovered",
					slog.String("request_id", GetRequestID(r.Context())),
					slog.Any("error", recovered),
					slog.String("stack", string(debug.Stack())),
				)

				writeError(
					w,
					http.StatusInternalServerError,
					"internal server error",
				)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Milliseconds()

		h.logger.Info("Request handler",
			slog.String("request_id", GetRequestID(r.Context())),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.StatusCode()),
			slog.Int64("duration_ms", duration),
			slog.String("user_agent", r.UserAgent()),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("proto", r.Proto))
	})
}

func (h *Handler) RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqId := uuid.New().String()
		ctx := context.WithValue(r.Context(), RequestIDKey, reqId)
		r = r.WithContext(ctx)
		w.Header().Set("X-Request-ID", reqId)
		next.ServeHTTP(w, r)
	})
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok && id != "" {
		return id
	}
	return "unknown"
}
