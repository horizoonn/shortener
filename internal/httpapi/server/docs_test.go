package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocsRoute(t *testing.T) {
	route := DocsRoute()

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()

	route.Handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("DocsRoute status = %d, want %d", rec.Code, http.StatusOK)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("DocsRoute Content-Type = %q, want text/html", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "swagger-ui") {
		t.Error("DocsRoute body does not contain swagger-ui")
	}
}

func TestDocsOpenAPIRoute(t *testing.T) {
	route := DocsOpenAPIRoute()

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	route.Handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("DocsOpenAPIRoute status = %d, want %d", rec.Code, http.StatusOK)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/yaml") {
		t.Errorf("DocsOpenAPIRoute Content-Type = %q, want text/yaml", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "openapi") {
		t.Error("DocsOpenAPIRoute body does not contain openapi")
	}
}
