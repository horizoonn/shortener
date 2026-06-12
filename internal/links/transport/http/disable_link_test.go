package http

import (
	"context"
	"fmt"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
)

func TestHandlerDisableLinkSuccess(t *testing.T) {
	t.Parallel()

	disabledAt := time.Now().UTC()
	calls := 0
	handler := NewHandler(fakeLinksService{
		disableLink: func(_ context.Context, code string) (links.Link, error) {
			calls++
			if code != "abc12345" {
				t.Fatalf("expected code abc12345, got %q", code)
			}

			return links.Link{
				ID:          uuid.New(),
				Code:        code,
				OriginalURL: "https://example.com",
				DisabledAt:  &disabledAt,
			}, nil
		},
	}, "http://localhost:8080")

	rec := executeDisableLinkRequest(t, handler, "abc12345")

	if rec.Code != nethttp.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", nethttp.StatusNoContent, rec.Code, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty response body, got %q", rec.Body.String())
	}
	if calls != 1 {
		t.Fatalf("expected one DisableLink call, got %d", calls)
	}
}

func TestHandlerDisableLinkNotFound(t *testing.T) {
	t.Parallel()

	handler := NewHandler(fakeLinksService{
		disableLink: func(_ context.Context, _ string) (links.Link, error) {
			return links.Link{}, fmt.Errorf("link missing: %w", core_errors.ErrNotFound)
		},
	}, "http://localhost:8080")

	rec := executeDisableLinkRequest(t, handler, "missing1")

	assertErrorResponse(t, rec, nethttp.StatusNotFound, "not_found")
}

func TestHandlerDisableLinkInvalidCode(t *testing.T) {
	t.Parallel()

	handler := NewHandler(fakeLinksService{
		disableLink: func(_ context.Context, _ string) (links.Link, error) {
			return links.Link{}, fmt.Errorf("invalid code: %w", core_errors.ErrInvalidArgument)
		},
	}, "http://localhost:8080")

	rec := executeDisableLinkRequest(t, handler, "bad code")

	assertErrorResponse(t, rec, nethttp.StatusBadRequest, "invalid_argument")
}

func executeDisableLinkRequest(t *testing.T, handler *Handler, code string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(nethttp.MethodDelete, "/api/v1/links/test-code", nil)
	req.SetPathValue(linkCodePathValue, code)
	rec := httptest.NewRecorder()

	handler.DisableLink(rec, req)

	return rec
}
