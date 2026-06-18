package request

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	core_errors "github.com/horizoonn/shortener/internal/errors"
)

type testRequest struct {
	OriginalURL string `json:"original_url" validate:"required,url"`
}

func TestDecodeAndValidateJSONSuccess(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"original_url":"https://example.com"}`))
	rec := httptest.NewRecorder()

	var dst testRequest
	if err := DecodeAndValidateJSON(rec, req, &dst, 1024); err != nil {
		t.Fatalf("decode and validate json: %v", err)
	}
	if dst.OriginalURL != "https://example.com" {
		t.Fatalf("expected original URL, got %q", dst.OriginalURL)
	}
}

func TestDecodeAndValidateJSONRejectsInvalidBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "malformed", body: `{"original_url":`},
		{name: "unknown field", body: `{"original_url":"https://example.com","extra":true}`},
		{name: "multiple values", body: `{"original_url":"https://example.com"} {}`},
		{name: "validation", body: `{"original_url":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			var dst testRequest
			err := DecodeAndValidateJSON(rec, req, &dst, 1024)
			if !errors.Is(err, core_errors.ErrInvalidArgument) {
				t.Fatalf("expected invalid argument, got %v", err)
			}
		})
	}
}

func TestDecodeAndValidateJSONRejectsOversizedBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"original_url":"https://example.com"}`))
	rec := httptest.NewRecorder()

	var dst testRequest
	err := DecodeAndValidateJSON(rec, req, &dst, 8)
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}
