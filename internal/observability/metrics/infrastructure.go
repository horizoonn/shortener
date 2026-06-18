package metrics

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	goredis "github.com/redis/go-redis/v9"
)

type PostgresPoolStats interface {
	Stat() *pgxpool.Stat
}

type RedisPoolStats interface {
	PoolStats() *goredis.PoolStats
}

func (m *Metrics) RegisterPostgresPool(pool PostgresPoolStats) {
	if m == nil || pool == nil {
		return
	}

	m.registry.MustRegister(
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "postgres_pool",
				Name:      "acquired_connections",
				Help:      "Current number of acquired PostgreSQL connections.",
			},
			func() float64 { return float64(pool.Stat().AcquiredConns()) },
		),
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "postgres_pool",
				Name:      "idle_connections",
				Help:      "Current number of idle PostgreSQL connections.",
			},
			func() float64 { return float64(pool.Stat().IdleConns()) },
		),
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "postgres_pool",
				Name:      "total_connections",
				Help:      "Current total number of PostgreSQL connections.",
			},
			func() float64 { return float64(pool.Stat().TotalConns()) },
		),
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "postgres_pool",
				Name:      "max_connections",
				Help:      "Configured maximum number of PostgreSQL connections.",
			},
			func() float64 { return float64(pool.Stat().MaxConns()) },
		),
		prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "postgres_pool",
				Name:      "acquires_total",
				Help:      "Total number of successful PostgreSQL pool acquires.",
			},
			func() float64 { return float64(pool.Stat().AcquireCount()) },
		),
		prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "postgres_pool",
				Name:      "canceled_acquires_total",
				Help:      "Total number of PostgreSQL pool acquires canceled by context.",
			},
			func() float64 { return float64(pool.Stat().CanceledAcquireCount()) },
		),
		prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "postgres_pool",
				Name:      "empty_acquires_total",
				Help:      "Total number of PostgreSQL pool acquires that waited because the pool was empty.",
			},
			func() float64 { return float64(pool.Stat().EmptyAcquireCount()) },
		),
		prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "postgres_pool",
				Name:      "new_connections_total",
				Help:      "Total number of PostgreSQL connections opened by the pool.",
			},
			func() float64 { return float64(pool.Stat().NewConnsCount()) },
		),
		prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "postgres_pool",
				Name:      "acquire_duration_seconds_total",
				Help:      "Total duration spent acquiring PostgreSQL connections.",
			},
			func() float64 { return pool.Stat().AcquireDuration().Seconds() },
		),
	)
}

func (m *Metrics) RegisterRedisPool(pool RedisPoolStats) {
	if m == nil || pool == nil {
		return
	}

	m.registry.MustRegister(
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "redis_pool",
				Name:      "total_connections",
				Help:      "Current total number of Redis connections.",
			},
			func() float64 { return float64(pool.PoolStats().TotalConns) },
		),
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "redis_pool",
				Name:      "idle_connections",
				Help:      "Current number of idle Redis connections.",
			},
			func() float64 { return float64(pool.PoolStats().IdleConns) },
		),
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "redis_pool",
				Name:      "pending_requests",
				Help:      "Current number of Redis requests waiting for a connection.",
			},
			func() float64 { return float64(pool.PoolStats().PendingRequests) },
		),
		prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "redis_pool",
				Name:      "hits_total",
				Help:      "Total number of Redis pool hits.",
			},
			func() float64 { return float64(pool.PoolStats().Hits) },
		),
		prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "redis_pool",
				Name:      "misses_total",
				Help:      "Total number of Redis pool misses.",
			},
			func() float64 { return float64(pool.PoolStats().Misses) },
		),
		prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "redis_pool",
				Name:      "timeouts_total",
				Help:      "Total number of Redis pool wait timeouts.",
			},
			func() float64 { return float64(pool.PoolStats().Timeouts) },
		),
		prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "redis_pool",
				Name:      "waits_total",
				Help:      "Total number of Redis pool waits.",
			},
			func() float64 { return float64(pool.PoolStats().WaitCount) },
		),
		prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "redis_pool",
				Name:      "wait_duration_seconds_total",
				Help:      "Total duration spent waiting for Redis connections.",
			},
			func() float64 {
				return time.Duration(pool.PoolStats().WaitDurationNs).Seconds()
			},
		),
		prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "redis_pool",
				Name:      "stale_connections_total",
				Help:      "Total number of stale Redis connections removed from the pool.",
			},
			func() float64 { return float64(pool.PoolStats().StaleConns) },
		),
		prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "redis_pool",
				Name:      "unusable_connections_total",
				Help:      "Total number of unusable Redis connections found by the pool.",
			},
			func() float64 { return float64(pool.PoolStats().Unusable) },
		),
	)
}
