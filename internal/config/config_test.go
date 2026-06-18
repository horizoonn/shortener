package config

import (
	"strings"
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
	t.Setenv("SHORTENER_REDIS_ADDR", "redis:6379")
	t.Setenv("SHORTENER_REDIS_DB", "2")
	t.Setenv("SHORTENER_REDIS_TIMEOUT", "1500ms")
	t.Setenv("SHORTENER_REDIS_CACHE_TTL", "3m")
	t.Setenv("SHORTENER_REDIS_MISS_TTL", "45s")
	t.Setenv("SHORTENER_HTTP_RATE_LIMIT_RPS", "5.5")
	t.Setenv("SHORTENER_HTTP_RATE_LIMIT_BURST", "12")
	t.Setenv("SHORTENER_HTTP_TRUSTED_PROXIES", "10.0.0.0/24,192.168.1.1/32")

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
	if cfg.RedisAddr != "redis:6379" {
		t.Fatalf("expected redis addr from SHORTENER_REDIS_ADDR, got %q", cfg.RedisAddr)
	}
	if cfg.RedisDB != 2 {
		t.Fatalf("expected redis db 2, got %d", cfg.RedisDB)
	}
	if cfg.RedisTimeout != 1500*time.Millisecond {
		t.Fatalf("expected redis timeout 1500ms, got %s", cfg.RedisTimeout)
	}
	if cfg.RedisCacheTTL != 3*time.Minute {
		t.Fatalf("expected redis cache TTL 3m, got %s", cfg.RedisCacheTTL)
	}
	if cfg.RedisMissTTL != 45*time.Second {
		t.Fatalf("expected redis miss TTL 45s, got %s", cfg.RedisMissTTL)
	}
	if cfg.HTTP.RateLimitRPS != 5.5 {
		t.Fatalf("expected HTTP rate limit RPS 5.5, got %f", cfg.HTTP.RateLimitRPS)
	}
	if cfg.HTTP.RateLimitBurst != 12 {
		t.Fatalf("expected HTTP rate limit burst 12, got %d", cfg.HTTP.RateLimitBurst)
	}
	if len(cfg.HTTP.TrustedProxies) != 2 || cfg.HTTP.TrustedProxies[0] != "10.0.0.0/24" || cfg.HTTP.TrustedProxies[1] != "192.168.1.1/32" {
		t.Fatalf("expected HTTP trusted proxies, got %v", cfg.HTTP.TrustedProxies)
	}
	if cfg.Postgres.TimeZone != "UTC" {
		t.Fatalf("expected postgres time zone UTC, got %q", cfg.Postgres.TimeZone)
	}
}

func TestLoadRejectsInvalidPublicBaseURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "missing scheme",
			url:  "localhost:8080",
		},
		{
			name: "with query",
			url:  "https://short.example?source=test",
		},
		{
			name: "with fragment",
			url:  "https://short.example#links",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SHORTENER_HTTP_PUBLIC_BASE_URL", tt.url)

			_, err := Load()
			if err == nil {
				t.Fatal("expected invalid public base URL error")
			}
			if !strings.Contains(err.Error(), "public base URL") {
				t.Fatalf("expected public base URL error, got %v", err)
			}
		})
	}
}
