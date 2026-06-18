package config

import (
	"testing"
	"time"
)

func TestLoadUsesDocumentedEnvironmentNames(t *testing.T) {
	t.Setenv("SHORTENER_LOG_LEVEL", "INFO")
	t.Setenv("SHORTENER_LOG_FOLDER", ".out/test-logs")
	t.Setenv("SHORTENER_HTTP_ADDR", ":9090")
	t.Setenv("SHORTENER_HTTP_PUBLIC_BASE_URL", "https://short.example")
	t.Setenv("SHORTENER_HTTP_SHUTDOWN_TIMEOUT", "3s")
	t.Setenv("SHORTENER_DATABASE_URL", "postgres://user:pass@db:5432/shortener?sslmode=disable")
	t.Setenv("SHORTENER_POSTGRES_TIMEOUT", "4s")
	t.Setenv("SHORTENER_POSTGRES_MAX_CONNS", "7")
	t.Setenv("SHORTENER_POSTGRES_MIN_CONNS", "1")
	t.Setenv("SHORTENER_POSTGRES_MAX_CONN_IDLE_TIME", "2m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Logger.Level != "INFO" {
		t.Fatalf("expected logger level from SHORTENER_LOG_LEVEL, got %q", cfg.Logger.Level)
	}
	if cfg.Logger.Folder != ".out/test-logs" {
		t.Fatalf("expected logger folder from SHORTENER_LOG_FOLDER, got %q", cfg.Logger.Folder)
	}
	if cfg.HTTP.Addr != ":9090" {
		t.Fatalf("expected HTTP addr from SHORTENER_HTTP_ADDR, got %q", cfg.HTTP.Addr)
	}
	if cfg.HTTP.PublicBaseURL != "https://short.example" {
		t.Fatalf("expected HTTP public base URL from SHORTENER_HTTP_PUBLIC_BASE_URL, got %q", cfg.HTTP.PublicBaseURL)
	}
	if cfg.HTTP.ShutdownTimeout != 3*time.Second {
		t.Fatalf("expected HTTP shutdown timeout 3s, got %s", cfg.HTTP.ShutdownTimeout)
	}
	if cfg.Postgres.URL != "postgres://user:pass@db:5432/shortener?sslmode=disable" {
		t.Fatalf("expected postgres URL from SHORTENER_DATABASE_URL, got %q", cfg.Postgres.URL)
	}
	if cfg.Postgres.Timeout != 4*time.Second {
		t.Fatalf("expected postgres timeout 4s, got %s", cfg.Postgres.Timeout)
	}
	if cfg.Postgres.MaxConns != 7 {
		t.Fatalf("expected postgres max conns 7, got %d", cfg.Postgres.MaxConns)
	}
	if cfg.Postgres.MinConns != 1 {
		t.Fatalf("expected postgres min conns 1, got %d", cfg.Postgres.MinConns)
	}
	if cfg.Postgres.MaxConnIdleTime != 2*time.Minute {
		t.Fatalf("expected postgres max idle time 2m, got %s", cfg.Postgres.MaxConnIdleTime)
	}
}
