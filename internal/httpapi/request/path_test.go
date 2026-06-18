package request

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	core_errors "github.com/horizoonn/shortener/internal/errors"
)

func TestGetStringPathValue(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/s/abc12345", nil)
	req.SetPathValue("code", "abc12345")

	got, err := GetStringPathValue(req, "code")
	if err != nil {
		t.Fatalf("get path value: %v", err)
	}
	if got != "abc12345" {
		t.Fatalf("expected abc12345, got %q", got)
	}
}

func TestGetStringPathValueMissing(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/s/", nil)

	_, err := GetStringPathValue(req, "code")
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}
