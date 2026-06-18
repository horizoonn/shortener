package middleware

import (
	"net/http"
	"time"

	"github.com/horizoonn/shortener/internal/httpapi/response"
	"github.com/horizoonn/shortener/internal/logger"
	"go.uber.org/zap"
)

func Logger(log *logger.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := RequestIDFromContext(r.Context())
			scopedLogger := log.With(
				zap.String("request_id", requestID),
				zap.String("url", r.URL.String()),
				zap.String("remote_addr", r.RemoteAddr),
			)

			ctx := logger.ToContext(r.Context(), scopedLogger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Trace() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.FromContext(r.Context())
			rw := response.NewResponseWriter(w)

			startedAt := time.Now()
			log.Debug(
				">>> incoming HTTP request",
				zap.String("http_method", r.Method),
				zap.Time("time", startedAt.UTC()),
				zap.String("user_agent", r.UserAgent()),
			)

			next.ServeHTTP(rw, r)

			log.Debug(
				"<<< done HTTP request",
				zap.Int("status_code", rw.GetStatusCode()),
				zap.Duration("latency", time.Since(startedAt)),
			)
		})
	}
}
