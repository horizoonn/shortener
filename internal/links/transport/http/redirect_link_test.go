package http

import (
	"context"
	"fmt"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
)

type fakeClickRecorder struct {
	recordClick func(ctx context.Context, linkID uuid.UUID, userAgent string, referer *string, ip *string) error
	calls       int
}

func (r *fakeClickRecorder) RecordClick(
	ctx context.Context,
	linkID uuid.UUID,
	userAgent string,
	referer *string,
	ip *string,
) error {
	r.calls++
	if r.recordClick == nil {
		return nil
	}

	return r.recordClick(ctx, linkID, userAgent, referer, ip)
}

func TestHandlerRedirectLinkSuccess(t *testing.T) {
	t.Parallel()

	linkID := uuid.New()
	serviceCalls := 0
	recorder := &fakeClickRecorder{
		recordClick: func(_ context.Context, gotLinkID uuid.UUID, userAgent string, referer *string, ip *string) error {
			if gotLinkID != linkID {
				t.Fatalf("expected link ID %s, got %s", linkID, gotLinkID)
			}
			if userAgent != "shortener-test-agent" {
				t.Fatalf("expected user agent to be passed, got %q", userAgent)
			}
			if referer == nil || *referer != "https://referer.example/path" {
				t.Fatalf("expected referer header, got %v", referer)
			}
			if ip == nil || *ip != "192.0.2.10" {
				t.Fatalf("expected remote IP 192.0.2.10, got %v", ip)
			}

			return nil
		},
	}
	handler := NewHandlerWithClickRecorder(fakeLinksService{
		resolveLink: func(_ context.Context, code string) (links.Link, error) {
			serviceCalls++
			if code != "abc12345" {
				t.Fatalf("expected code abc12345, got %q", code)
			}

			return links.Link{
				ID:          linkID,
				Code:        code,
				OriginalURL: "https://example.com/target",
			}, nil
		},
	}, recorder, "http://localhost:8080")

	rec := executeRedirectLinkRequest(t, handler, "abc12345", func(req *nethttp.Request) {
		req.Header.Set("User-Agent", "shortener-test-agent")
		req.Header.Set("Referer", "https://referer.example/path")
		req.RemoteAddr = "192.0.2.10:54321"
	})

	if rec.Code != nethttp.StatusFound {
		t.Fatalf("expected status %d, got %d: %s", nethttp.StatusFound, rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); location != "https://example.com/target" {
		t.Fatalf("expected Location https://example.com/target, got %q", location)
	}
	if serviceCalls != 1 {
		t.Fatalf("expected one ResolveLink call, got %d", serviceCalls)
	}
	if recorder.calls != 1 {
		t.Fatalf("expected one RecordClick call, got %d", recorder.calls)
	}
}

func TestHandlerRedirectLinkIgnoresRecordClickError(t *testing.T) {
	t.Parallel()

	recorder := &fakeClickRecorder{
		recordClick: func(_ context.Context, _ uuid.UUID, _ string, _ *string, _ *string) error {
			return fmt.Errorf("analytics insert failed")
		},
	}
	handler := NewHandlerWithClickRecorder(fakeLinksService{
		resolveLink: func(_ context.Context, code string) (links.Link, error) {
			return links.Link{
				ID:          uuid.New(),
				Code:        code,
				OriginalURL: "https://example.com/target",
			}, nil
		},
	}, recorder, "http://localhost:8080")

	rec := executeRedirectLinkRequest(t, handler, "abc12345", nil)

	if rec.Code != nethttp.StatusFound {
		t.Fatalf("expected status %d, got %d: %s", nethttp.StatusFound, rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); location != "https://example.com/target" {
		t.Fatalf("expected redirect location, got %q", location)
	}
	if recorder.calls != 1 {
		t.Fatalf("expected one RecordClick call, got %d", recorder.calls)
	}
}

func TestHandlerRedirectLinkNotFound(t *testing.T) {
	t.Parallel()

	recorder := &fakeClickRecorder{}
	handler := NewHandlerWithClickRecorder(fakeLinksService{
		resolveLink: func(_ context.Context, _ string) (links.Link, error) {
			return links.Link{}, fmt.Errorf("link missing: %w", core_errors.ErrNotFound)
		},
	}, recorder, "http://localhost:8080")

	rec := executeRedirectLinkRequest(t, handler, "missing1", nil)

	assertErrorResponse(t, rec, nethttp.StatusNotFound, "not_found")
	if recorder.calls != 0 {
		t.Fatalf("expected no RecordClick calls, got %d", recorder.calls)
	}
}

func TestHandlerRedirectLinkDisabledAsNotFound(t *testing.T) {
	t.Parallel()

	recorder := &fakeClickRecorder{}
	handler := NewHandlerWithClickRecorder(fakeLinksService{
		resolveLink: func(_ context.Context, _ string) (links.Link, error) {
			return links.Link{}, fmt.Errorf("link disabled: %w", core_errors.ErrNotFound)
		},
	}, recorder, "http://localhost:8080")

	rec := executeRedirectLinkRequest(t, handler, "disabled1", nil)

	assertErrorResponse(t, rec, nethttp.StatusNotFound, "not_found")
	if recorder.calls != 0 {
		t.Fatalf("expected no RecordClick calls, got %d", recorder.calls)
	}
}

func TestHandlerRedirectLinkEmptyCode(t *testing.T) {
	t.Parallel()

	recorder := &fakeClickRecorder{}
	handler := NewHandlerWithClickRecorder(fakeLinksService{
		resolveLink: func(_ context.Context, _ string) (links.Link, error) {
			t.Fatal("ResolveLink must not be called for empty code")
			return links.Link{}, nil
		},
	}, recorder, "http://localhost:8080")

	rec := executeRedirectLinkRequest(t, handler, "", nil)

	assertErrorResponse(t, rec, nethttp.StatusBadRequest, "invalid_argument")
	if recorder.calls != 0 {
		t.Fatalf("expected no RecordClick calls, got %d", recorder.calls)
	}
}

func TestHandlerRedirectLinkInvalidCode(t *testing.T) {
	t.Parallel()

	recorder := &fakeClickRecorder{}
	handler := NewHandlerWithClickRecorder(fakeLinksService{
		resolveLink: func(_ context.Context, _ string) (links.Link, error) {
			return links.Link{}, fmt.Errorf("invalid code: %w", core_errors.ErrInvalidArgument)
		},
	}, recorder, "http://localhost:8080")

	rec := executeRedirectLinkRequest(t, handler, "bad code", nil)

	assertErrorResponse(t, rec, nethttp.StatusBadRequest, "invalid_argument")
	if recorder.calls != 0 {
		t.Fatalf("expected no RecordClick calls, got %d", recorder.calls)
	}
}

func TestHandlerRedirectLinkInvalidRemoteAddrPassesNilIP(t *testing.T) {
	t.Parallel()

	recorder := &fakeClickRecorder{
		recordClick: func(_ context.Context, _ uuid.UUID, _ string, _ *string, ip *string) error {
			if ip != nil {
				t.Fatalf("expected nil IP for invalid remote addr, got %q", *ip)
			}

			return nil
		},
	}
	handler := NewHandlerWithClickRecorder(fakeLinksService{
		resolveLink: func(_ context.Context, code string) (links.Link, error) {
			return links.Link{
				ID:          uuid.New(),
				Code:        code,
				OriginalURL: "https://example.com/target",
			}, nil
		},
	}, recorder, "http://localhost:8080")

	rec := executeRedirectLinkRequest(t, handler, "abc12345", func(req *nethttp.Request) {
		req.RemoteAddr = "not-an-ip"
	})

	if rec.Code != nethttp.StatusFound {
		t.Fatalf("expected status %d, got %d: %s", nethttp.StatusFound, rec.Code, rec.Body.String())
	}
}

func executeRedirectLinkRequest(
	t *testing.T,
	handler *Handler,
	code string,
	mutate func(req *nethttp.Request),
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(nethttp.MethodGet, "/s/test-code", nil)
	req.SetPathValue(linkCodePathValue, code)
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()

	handler.RedirectLink(rec, req)

	return rec
}
