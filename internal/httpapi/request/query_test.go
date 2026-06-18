package request

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core_errors "github.com/horizoonn/shortener/internal/errors"
)

func TestGetDateQueryParam(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/?from=2026-06-12", nil)

	got, err := GetDateQueryParam(req, "from")
	if err != nil {
		t.Fatalf("get date query param: %v", err)
	}
	want := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)
	if got == nil || !got.Equal(want) {
		t.Fatalf("expected %s, got %v", want, got)
	}
}

func TestGetDateQueryParamInvalid(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/?from=bad", nil)

	_, err := GetDateQueryParam(req, "from")
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestGetIntQueryParam(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/?limit=25", nil)

	got, err := GetIntQueryParam(req, "limit")
	if err != nil {
		t.Fatalf("get int query param: %v", err)
	}
	if got == nil || *got != 25 {
		t.Fatalf("expected 25, got %v", got)
	}
}

func TestGetIntQueryParamInvalid(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/?limit=bad", nil)

	_, err := GetIntQueryParam(req, "limit")
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}
