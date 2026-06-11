package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/horizoonn/shortener/internal/config"
	"github.com/horizoonn/shortener/internal/httpapi/middleware"
	"github.com/horizoonn/shortener/internal/logger"
	"go.uber.org/zap"
)

func TestHealthz(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	log := &logger.Logger{Logger: zap.NewNop()}
	srv, err := NewHTTPServer(
		config.HTTPConfig{
			Addr:              ":0",
			ShutdownTimeout:   1,
			ReadHeaderTimeout: 1,
			ReadTimeout:       1,
			WriteTimeout:      1,
			IdleTimeout:       1,
			AllowedOrigins:    []string{"*"},
			AllowedMethods:    []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		},
		log,
		middleware.RequestID(),
		middleware.Logger(log),
		middleware.Trace(),
		middleware.Panic(),
		middleware.CORS([]string{"*"}, []string{http.MethodGet, http.MethodPost, http.MethodOptions}),
	)
	if err != nil {
		t.Fatalf("init HTTP server: %v", err)
	}
	srv.RegisterRoutes(HealthRoute())
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content type application/json, got %q", got)
	}
	if got := rec.Body.String(); got != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected response body: %q", got)
	}
	if got := rec.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("expected request id header")
	}
}
