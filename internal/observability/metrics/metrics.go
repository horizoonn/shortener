package metrics

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/horizoonn/shortener/internal/httpapi/middleware"
	"github.com/horizoonn/shortener/internal/httpapi/response"
	"github.com/horizoonn/shortener/internal/storage/postgres/pool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace   = "shortener"
	metricsPath = "/metrics"
)

type Metrics struct {
	registry *prometheus.Registry

	httpRequests    *prometheus.CounterVec
	httpDuration    *prometheus.HistogramVec
	httpInFlight    prometheus.Gauge
	cacheHits       *prometheus.CounterVec
	cacheMisses     *prometheus.CounterVec
	dbQueryDuration *prometheus.HistogramVec
	dbQueryErrors   *prometheus.CounterVec
	linksCreated    *prometheus.CounterVec
	linksResolved   prometheus.Counter
}

func New() *Metrics {
	registry := prometheus.NewRegistry()
	m := &Metrics{
		registry: registry,
		httpRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests handled by the service.",
			},
			[]string{"method", "route", "status"},
		),
		httpDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds.",
				Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"method", "route", "status"},
		),
		httpInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_requests_in_flight",
				Help:      "Current number of HTTP requests being handled.",
			},
		),
		cacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits.",
			},
			[]string{"cache"},
		),
		cacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses.",
			},
			[]string{"cache"},
		),
		dbQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds.",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"query"},
		),
		dbQueryErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_query_errors_total",
				Help:      "Total number of database query errors.",
			},
			[]string{"query"},
		),
		linksCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "links_created_total",
				Help:      "Total number of shortened links created.",
			},
			[]string{"type"},
		),
		linksResolved: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "links_resolved_total",
				Help:      "Total number of shortened links resolved.",
			},
		),
	}

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewBuildInfoCollector(),
		m.httpRequests,
		m.httpDuration,
		m.httpInFlight,
		m.cacheHits,
		m.cacheMisses,
		m.dbQueryDuration,
		m.dbQueryErrors,
		m.linksCreated,
		m.linksResolved,
	)

	return m
}

func (m *Metrics) Handler() http.Handler {
	if m == nil {
		return http.NotFoundHandler()
	}

	return promhttp.HandlerFor(
		m.registry,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	)
}

func (m *Metrics) Middleware() middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m == nil || r.URL.Path == metricsPath {
				next.ServeHTTP(w, r)
				return
			}

			rw := response.NewResponseWriter(w)
			startedAt := time.Now()
			m.httpInFlight.Inc()
			defer m.httpInFlight.Dec()

			next.ServeHTTP(rw, r)

			route := routeLabel(r)
			status := strconv.Itoa(rw.GetStatusCode())
			method := strings.ToLower(r.Method)

			m.httpRequests.WithLabelValues(method, route, status).Inc()
			m.httpDuration.WithLabelValues(method, route, status).Observe(time.Since(startedAt).Seconds())
		})
	}
}

func (m *Metrics) Registry() *prometheus.Registry {
	if m == nil {
		return nil
	}

	return m.registry
}

func (m *Metrics) RecordCacheHit(cache string) {
	if m == nil {
		return
	}
	m.cacheHits.WithLabelValues(cache).Inc()
}

func (m *Metrics) RecordCacheMiss(cache string) {
	if m == nil {
		return
	}
	m.cacheMisses.WithLabelValues(cache).Inc()
}

func (m *Metrics) RecordDBQuery(query string, duration time.Duration, err error) {
	if m == nil {
		return
	}
	cleanQuery := cleanSQL(query)
	m.dbQueryDuration.WithLabelValues(cleanQuery).Observe(duration.Seconds())
	if err != nil && isOperationalError(err) {
		m.dbQueryErrors.WithLabelValues(cleanQuery).Inc()
	}
}

func (m *Metrics) RecordLinkCreated(isCustom bool) {
	if m == nil {
		return
	}
	linkType := "generated"
	if isCustom {
		linkType = "custom"
	}
	m.linksCreated.WithLabelValues(linkType).Inc()
}

func (m *Metrics) RecordLinkResolved() {
	if m == nil {
		return
	}
	m.linksResolved.Inc()
}

func routeLabel(r *http.Request) string {
	pattern := r.Pattern
	if pattern == "" {
		return "unmatched"
	}

	fields := strings.Fields(pattern)
	if len(fields) == 2 {
		pattern = fields[1]
	}
	if pattern == "" {
		return "unmatched"
	}

	return pattern
}

func cleanSQL(query string) string {
	fields := strings.Fields(query)
	return strings.Join(fields, " ")
}

func isOperationalError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pool.ErrNoRows) || errors.Is(err, pool.ErrUniqueViolation) || errors.Is(err, pool.ErrViolatesForeignKey) {
		return false
	}
	return true
}
