package metrics

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMiddlewareRecordsHTTPMetricsWithRoutePattern(t *testing.T) {
	t.Parallel()

	m := New()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/links/{code}/qr", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/links/demo/qr", nil)
	rec := httptest.NewRecorder()

	m.Middleware()(mux).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if !hasMetricWithLabels(t, m, "shortener_http_requests_total", map[string]string{
		"method": "get",
		"route":  "/api/v1/links/{code}/qr",
		"status": "201",
	}) {
		t.Fatal("expected HTTP request metric with route pattern labels")
	}
}

func TestMiddlewareSkipsMetricsEndpoint(t *testing.T) {
	t.Parallel()

	m := New()
	mux := http.NewServeMux()
	mux.Handle("/metrics", m.Handler())

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	m.Middleware()(mux).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if hasMetricWithLabels(t, m, "shortener_http_requests_total", map[string]string{
		"method": "get",
		"route":  "/metrics",
		"status": "200",
	}) {
		t.Fatal("expected /metrics request to be excluded from HTTP request metrics")
	}
}

func TestMetricsRecordCacheHitMiss(t *testing.T) {
	t.Parallel()

	m := New()
	m.RecordCacheHit("links")
	m.RecordCacheMiss("links")

	if !hasMetricWithLabels(t, m, "shortener_cache_hits_total", map[string]string{"cache": "links"}) {
		t.Fatal("expected cache hit metric to be recorded")
	}
	if !hasMetricWithLabels(t, m, "shortener_cache_misses_total", map[string]string{"cache": "links"}) {
		t.Fatal("expected cache miss metric to be recorded")
	}
}

func TestMetricsRecordDBQueryAndErrors(t *testing.T) {
	t.Parallel()

	m := New()
	m.RecordDBQuery("SELECT * FROM links", 100*time.Millisecond, nil)
	m.RecordDBQuery("INSERT INTO links", 50*time.Millisecond, errors.New("connection failed"))

	if !hasMetricWithLabels(t, m, "shortener_db_query_duration_seconds", map[string]string{"query": "SELECT * FROM links"}) {
		t.Fatal("expected DB query duration metric to be recorded")
	}
	if !hasMetricWithLabels(t, m, "shortener_db_query_errors_total", map[string]string{"query": "INSERT INTO links"}) {
		t.Fatal("expected DB query error metric to be recorded")
	}
}

func TestMetricsRecordLinkCreatedResolved(t *testing.T) {
	t.Parallel()

	m := New()
	m.RecordLinkCreated(true)
	m.RecordLinkCreated(false)
	m.RecordLinkResolved()

	if !hasMetricWithLabels(t, m, "shortener_links_created_total", map[string]string{"type": "custom"}) {
		t.Fatal("expected custom link creation metric to be recorded")
	}
	if !hasMetricWithLabels(t, m, "shortener_links_created_total", map[string]string{"type": "generated"}) {
		t.Fatal("expected generated link creation metric to be recorded")
	}
	if !hasMetricWithLabels(t, m, "shortener_links_resolved_total", map[string]string{}) {
		t.Fatal("expected link resolution metric to be recorded")
	}
}

func hasMetricWithLabels(t *testing.T, m *Metrics, name string, labels map[string]string) bool {
	t.Helper()

	families, err := m.Registry().Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			actual := make(map[string]string, len(metric.GetLabel()))
			for _, pair := range metric.GetLabel() {
				actual[pair.GetName()] = pair.GetValue()
			}
			if hasLabels(actual, labels) {
				return true
			}
		}
	}

	return false
}

func hasLabels(actual map[string]string, labels map[string]string) bool {
	for key, expectedValue := range labels {
		if actual[key] != expectedValue {
			return false
		}
	}

	return true
}
