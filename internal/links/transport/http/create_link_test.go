package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
)

type fakeLinksService struct {
	createLink  func(ctx context.Context, originalURL string, customAlias *string, expiresAt *time.Time) (links.Link, error)
	getLink     func(ctx context.Context, code string) (links.Link, error)
	resolveLink func(ctx context.Context, code string) (links.Link, error)
	disableLink func(ctx context.Context, code string) (links.Link, error)
}

func (s fakeLinksService) CreateLink(ctx context.Context, originalURL string, customAlias *string, expiresAt *time.Time) (links.Link, error) {
	if s.createLink == nil {
		return links.Link{}, fmt.Errorf("create link not implemented")
	}

	return s.createLink(ctx, originalURL, customAlias, expiresAt)
}

func (s fakeLinksService) ResolveLink(ctx context.Context, code string) (links.Link, error) {
	if s.resolveLink == nil {
		return links.Link{}, fmt.Errorf("resolve link not implemented")
	}

	return s.resolveLink(ctx, code)
}

func (s fakeLinksService) GetLink(ctx context.Context, code string) (links.Link, error) {
	if s.getLink == nil {
		return links.Link{}, fmt.Errorf("get link not implemented")
	}

	return s.getLink(ctx, code)
}

func (s fakeLinksService) DisableLink(ctx context.Context, code string) (links.Link, error) {
	if s.disableLink == nil {
		return links.Link{}, fmt.Errorf("disable link not implemented")
	}

	return s.disableLink(ctx, code)
}

func TestHandlerCreateLinkGeneratedSuccess(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	linkID := uuid.New()
	handler := NewHandler(fakeLinksService{
		createLink: func(_ context.Context, originalURL string, customAlias *string, _ *time.Time) (links.Link, error) {
			if originalURL != "https://example.com/path" {
				t.Fatalf("expected original URL to be passed to service, got %q", originalURL)
			}
			if customAlias != nil {
				t.Fatalf("expected nil custom alias, got %q", *customAlias)
			}

			return links.Link{
				ID:          linkID,
				Code:        "abc12345",
				OriginalURL: originalURL,
				IsCustom:    false,
				CreatedAt:   now,
			}, nil
		},
	}, "http://localhost:8080")

	rec := executeCreateLinkRequest(t, handler, `{"original_url":"https://example.com/path"}`)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", nethttp.StatusCreated, rec.Code, rec.Body.String())
	}

	var response CreateLinkResponse
	decodeResponse(t, rec, &response)

	if response.ID != linkID {
		t.Fatalf("expected id %s, got %s", linkID, response.ID)
	}
	if response.Code != "abc12345" {
		t.Fatalf("expected code abc12345, got %q", response.Code)
	}
	if response.ShortURL != "http://localhost:8080/s/abc12345" {
		t.Fatalf("unexpected short URL: %q", response.ShortURL)
	}
	if response.IsCustom {
		t.Fatal("expected generated link to not be custom")
	}
}

func TestHandlerCreateLinkCustomAliasSuccess(t *testing.T) {
	t.Parallel()

	handler := NewHandler(fakeLinksService{
		createLink: func(_ context.Context, originalURL string, customAlias *string, _ *time.Time) (links.Link, error) {
			if customAlias == nil || *customAlias != "my-link" {
				t.Fatalf("expected custom alias my-link, got %v", customAlias)
			}

			return links.Link{
				ID:          uuid.New(),
				Code:        *customAlias,
				OriginalURL: originalURL,
				IsCustom:    true,
				CreatedAt:   time.Now().UTC(),
			}, nil
		},
	}, "http://localhost:8080/")

	rec := executeCreateLinkRequest(t, handler, `{"original_url":"https://example.com","custom_alias":"my-link"}`)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", nethttp.StatusCreated, rec.Code, rec.Body.String())
	}

	var response CreateLinkResponse
	decodeResponse(t, rec, &response)

	if response.Code != "my-link" {
		t.Fatalf("expected custom alias code, got %q", response.Code)
	}
	if response.ShortURL != "http://localhost:8080/s/my-link" {
		t.Fatalf("unexpected short URL: %q", response.ShortURL)
	}
	if !response.IsCustom {
		t.Fatal("expected custom link response")
	}
}

func TestHandlerCreateLinkInvalidJSON(t *testing.T) {
	t.Parallel()

	handler := NewHandler(fakeLinksService{
		createLink: func(_ context.Context, _ string, _ *string, _ *time.Time) (links.Link, error) {
			t.Fatal("service should not be called for invalid JSON")
			return links.Link{}, nil
		},
	}, "http://localhost:8080")

	rec := executeCreateLinkRequest(t, handler, `{"original_url":`)

	assertErrorResponse(t, rec, nethttp.StatusBadRequest, "invalid_argument")
}

func TestHandlerCreateLinkMissingOriginalURL(t *testing.T) {
	t.Parallel()

	handler := NewHandler(fakeLinksService{
		createLink: func(_ context.Context, _ string, _ *string, _ *time.Time) (links.Link, error) {
			t.Fatal("service should not be called for invalid request")
			return links.Link{}, nil
		},
	}, "http://localhost:8080")

	rec := executeCreateLinkRequest(t, handler, `{"custom_alias":"my-link"}`)

	assertErrorResponse(t, rec, nethttp.StatusBadRequest, "invalid_argument")
}

func TestHandlerCreateLinkRequestBodyTooLarge(t *testing.T) {
	t.Parallel()

	handler := NewHandler(fakeLinksService{
		createLink: func(_ context.Context, _ string, _ *string, _ *time.Time) (links.Link, error) {
			t.Fatal("service should not be called for oversized request")
			return links.Link{}, nil
		},
	}, "http://localhost:8080")

	body := `{"original_url":"https://example.com/` + string(bytes.Repeat([]byte("a"), maxCreateLinkRequestBytes)) + `"}`
	rec := executeCreateLinkRequest(t, handler, body)

	assertErrorResponse(t, rec, nethttp.StatusBadRequest, "invalid_argument")
}

func TestHandlerCreateLinkInvalidURL(t *testing.T) {
	t.Parallel()

	handler := NewHandler(fakeLinksService{
		createLink: func(_ context.Context, _ string, _ *string, _ *time.Time) (links.Link, error) {
			return links.Link{}, fmt.Errorf("invalid URL: %w", core_errors.ErrInvalidArgument)
		},
	}, "http://localhost:8080")

	rec := executeCreateLinkRequest(t, handler, `{"original_url":"ftp://example.com"}`)

	assertErrorResponse(t, rec, nethttp.StatusBadRequest, "invalid_argument")
}

func TestHandlerCreateLinkDuplicateAlias(t *testing.T) {
	t.Parallel()

	handler := NewHandler(fakeLinksService{
		createLink: func(_ context.Context, _ string, _ *string, _ *time.Time) (links.Link, error) {
			return links.Link{}, fmt.Errorf("custom alias conflict: %w", core_errors.ErrConflict)
		},
	}, "http://localhost:8080")

	rec := executeCreateLinkRequest(t, handler, `{"original_url":"https://example.com","custom_alias":"taken"}`)

	assertErrorResponse(t, rec, nethttp.StatusConflict, "conflict")
}

func TestHandlerCreateLinkInternalError(t *testing.T) {
	t.Parallel()

	handler := NewHandler(fakeLinksService{
		createLink: func(_ context.Context, _ string, _ *string, _ *time.Time) (links.Link, error) {
			return links.Link{}, errors.New("database unavailable")
		},
	}, "http://localhost:8080")

	rec := executeCreateLinkRequest(t, handler, `{"original_url":"https://example.com"}`)

	assertErrorResponse(t, rec, nethttp.StatusInternalServerError, "internal_error")
}

func executeCreateLinkRequest(t *testing.T, handler *Handler, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/shorten", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateLink(rec, req)

	return rec
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()

	if err := json.Unmarshal(rec.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode response: %v; body: %s", err, rec.Body.String())
	}
}

func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("expected status %d, got %d: %s", wantStatus, rec.Code, rec.Body.String())
	}

	var responseBody struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	decodeResponse(t, rec, &responseBody)

	if responseBody.Code != wantCode {
		t.Fatalf("expected error code %q, got %q", wantCode, responseBody.Code)
	}
}
