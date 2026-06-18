package http

import (
	"context"
	"fmt"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/httpapi/server"
	"github.com/horizoonn/shortener/internal/links"
)

type fakeQRGenerator struct {
	generatePNG func(content string, size int) ([]byte, error)
	calls       int
}

func (g *fakeQRGenerator) GeneratePNG(content string, size int) ([]byte, error) {
	g.calls++
	if g.generatePNG == nil {
		return []byte{0x89, 'P', 'N', 'G'}, nil
	}

	return g.generatePNG(content, size)
}

func TestHandlerGetQRCodeSuccess(t *testing.T) {
	t.Parallel()

	linkID := uuid.New()
	generator := &fakeQRGenerator{
		generatePNG: func(content string, size int) ([]byte, error) {
			if content != "http://localhost:8080/s/abc12345" {
				t.Fatalf("expected QR content to be short URL, got %q", content)
			}
			if size != 512 {
				t.Fatalf("expected size 512, got %d", size)
			}

			return []byte{0x89, 'P', 'N', 'G', 1, 2, 3}, nil
		},
	}
	handler := NewHandlerWithDependencies(
		fakeLinksService{
			resolveLink: func(_ context.Context, code string) (links.Link, error) {
				if code != "abc12345" {
					t.Fatalf("expected code abc12345, got %q", code)
				}

				return links.Link{
					ID:          linkID,
					Code:        code,
					OriginalURL: "https://example.com/target",
				}, nil
			},
		},
		nil,
		nil,
		"http://localhost:8080",
		generator,
	)

	rec := executeGetQRCodeRequest(t, handler, "abc12345", "size=512")

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", nethttp.StatusOK, rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "image/png" {
		t.Fatalf("expected image/png content type, got %q", contentType)
	}
	if cacheControl := rec.Header().Get("Cache-Control"); cacheControl != qrCacheControl {
		t.Fatalf("expected cache control %q, got %q", qrCacheControl, cacheControl)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected non-empty PNG response")
	}
	if generator.calls != 1 {
		t.Fatalf("expected one generator call, got %d", generator.calls)
	}
}

func TestHandlerGetQRCodeDefaultSize(t *testing.T) {
	t.Parallel()

	generator := &fakeQRGenerator{
		generatePNG: func(_ string, size int) ([]byte, error) {
			if size != 256 {
				t.Fatalf("expected default size 256, got %d", size)
			}
			return []byte{0x89, 'P', 'N', 'G'}, nil
		},
	}
	handler := NewHandlerWithDependencies(
		fakeLinksService{
			resolveLink: func(_ context.Context, code string) (links.Link, error) {
				return links.Link{ID: uuid.New(), Code: code, OriginalURL: "https://example.com"}, nil
			},
		},
		nil,
		nil,
		"http://localhost:8080",
		generator,
	)

	rec := executeGetQRCodeRequest(t, handler, "abc12345", "")

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", nethttp.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestHandlerGetQRCodeInvalidSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{name: "not integer", query: "size=abc"},
		{name: "too small", query: "size=127"},
		{name: "too large", query: "size=1025"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			generator := &fakeQRGenerator{}
			handler := NewHandlerWithDependencies(
				fakeLinksService{
					resolveLink: func(_ context.Context, _ string) (links.Link, error) {
						t.Fatal("ResolveLink must not be called for invalid size")
						return links.Link{}, nil
					},
				},
				nil,
				nil,
				"http://localhost:8080",
				generator,
			)

			rec := executeGetQRCodeRequest(t, handler, "abc12345", tt.query)

			assertErrorResponse(t, rec, nethttp.StatusBadRequest, "invalid_argument")
			if generator.calls != 0 {
				t.Fatalf("expected no generator calls, got %d", generator.calls)
			}
		})
	}
}

func TestHandlerGetQRCodeLinkNotFound(t *testing.T) {
	t.Parallel()

	generator := &fakeQRGenerator{}
	handler := NewHandlerWithDependencies(
		fakeLinksService{
			resolveLink: func(_ context.Context, _ string) (links.Link, error) {
				return links.Link{}, fmt.Errorf("link missing: %w", core_errors.ErrNotFound)
			},
		},
		nil,
		nil,
		"http://localhost:8080",
		generator,
	)

	rec := executeGetQRCodeRequest(t, handler, "missing1", "")

	assertErrorResponse(t, rec, nethttp.StatusNotFound, "not_found")
	if generator.calls != 0 {
		t.Fatalf("expected no generator calls, got %d", generator.calls)
	}
}

func TestHandlerGetQRCodeGeneratorError(t *testing.T) {
	t.Parallel()

	generator := &fakeQRGenerator{
		generatePNG: func(_ string, _ int) ([]byte, error) {
			return nil, fmt.Errorf("png encoder failed")
		},
	}
	handler := NewHandlerWithDependencies(
		fakeLinksService{
			resolveLink: func(_ context.Context, code string) (links.Link, error) {
				return links.Link{ID: uuid.New(), Code: code, OriginalURL: "https://example.com"}, nil
			},
		},
		nil,
		nil,
		"http://localhost:8080",
		generator,
	)

	rec := executeGetQRCodeRequest(t, handler, "abc12345", "")

	assertErrorResponse(t, rec, nethttp.StatusInternalServerError, "internal_error")
	if generator.calls != 1 {
		t.Fatalf("expected one generator call, got %d", generator.calls)
	}
}

func TestHandlerGetQRCodeRouteRegistered(t *testing.T) {
	t.Parallel()

	handler := NewHandlerWithDependencies(
		fakeLinksService{
			resolveLink: func(_ context.Context, code string) (links.Link, error) {
				return links.Link{ID: uuid.New(), Code: code, OriginalURL: "https://example.com"}, nil
			},
		},
		nil,
		nil,
		"http://localhost:8080",
		&fakeQRGenerator{},
	)
	apiRouter := server.NewAPIVersionRouter(server.APIVersion1)
	apiRouter.AddRoutes(handler.Routes()...)
	mux := nethttp.NewServeMux()
	apiRouter.RegisterRoutesTo(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/links/abc12345/qr?size=256", nil)

	mux.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected route to return status %d, got %d: %s", nethttp.StatusOK, rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "image/png" {
		t.Fatalf("expected image/png content type, got %q", contentType)
	}
}

func executeGetQRCodeRequest(t *testing.T, handler *Handler, code string, query string) *httptest.ResponseRecorder {
	t.Helper()

	path := "/api/v1/links/test-code/qr"
	if query != "" {
		path += "?" + query
	}
	req := httptest.NewRequest(nethttp.MethodGet, path, nil)
	req.SetPathValue(linkCodePathValue, code)
	rec := httptest.NewRecorder()

	handler.GetQRCode(rec, req)

	return rec
}
