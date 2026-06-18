package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/horizoonn/shortener/internal/httpapi/request"
)

func TestRateLimit_AllowsWithinLimit(t *testing.T) {
	rl := NewIPRateLimiter(10, 10)
	resolver, _ := request.NewIPResolver(nil)

	handler := RateLimit(rl, resolver)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected %d, got %d", i, http.StatusOK, rec.Code)
		}
	}
}

func TestRateLimit_RejectsExceedingLimit(t *testing.T) {
	rl := NewIPRateLimiter(1, 2)
	resolver, _ := request.NewIPResolver(nil)

	handler := RateLimit(rl, resolver)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected %d, got %d", i, http.StatusOK, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected %d, got %d", http.StatusTooManyRequests, rec.Code)
	}
}

func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	rl := NewIPRateLimiter(1, 1)
	resolver, _ := request.NewIPResolver(nil)

	handler := RateLimit(rl, resolver)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ips := []string{"192.0.2.1:12345", "192.0.2.2:12345"}
	for _, ip := range ips {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("IP %s: expected %d, got %d", ip, http.StatusOK, rec.Code)
		}
	}
}

func TestRateLimit_NilLimiter(t *testing.T) {
	resolver, _ := request.NewIPResolver(nil)

	handler := RateLimit(nil, resolver)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRateLimit_EmptyClientIP_UsesFallbackKey(t *testing.T) {
	rl := NewIPRateLimiter(1, 1)
	resolver, _ := request.NewIPResolver(nil)

	handler := RateLimit(rl, resolver)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "invalid"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "invalid"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected %d for second request with empty IP, got %d", http.StatusTooManyRequests, rec2.Code)
	}
}

func TestRateLimit_SkipsIgnoredPaths(t *testing.T) {
	rl := NewIPRateLimiter(1, 1)
	resolver, _ := request.NewIPResolver(nil)

	handler := RateLimit(rl, resolver)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	paths := []string{"/healthz", "/readyz", "/metrics", "/docs", "/docs/openapi.yaml"}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = "192.0.2.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("path %s: expected %d, got %d", path, http.StatusOK, rec.Code)
		}
	}
}

func TestRateLimit_WithXForwardedFor(t *testing.T) {
	rl := NewIPRateLimiter(1, 1)
	resolver, _ := request.NewIPResolver([]string{"10.0.0.0/8"})

	handler := RateLimit(rl, resolver)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "10.0.0.1:12345"
	req2.Header.Set("X-Forwarded-For", "203.0.113.50")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected %d for XFF IP, got %d", http.StatusTooManyRequests, rec2.Code)
	}
}
