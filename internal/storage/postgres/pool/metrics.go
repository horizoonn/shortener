package pool

import (
	"context"
	"time"
)

// Metrics defines the interface required to record database query metrics.
type Metrics interface {
	RecordDBQuery(query string, duration time.Duration, err error)
}

type metricsPool struct {
	Pool
	metrics Metrics
}

// NewMetricsPool wraps the given Pool to record query duration and execution errors.
func NewMetricsPool(p Pool, m Metrics) Pool {
	if m == nil {
		return p
	}
	return &metricsPool{
		Pool:    p,
		metrics: m,
	}
}

func (p *metricsPool) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	start := time.Now()
	rows, err := p.Pool.Query(ctx, sql, args...)
	if err != nil {
		p.metrics.RecordDBQuery(sql, time.Since(start), err)
		return nil, err
	}
	return &metricsRows{
		Rows:    rows,
		metrics: p.metrics,
		sql:     sql,
		start:   start,
	}, nil
}

func (p *metricsPool) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return &metricsRow{
		Row:     p.Pool.QueryRow(ctx, sql, args...),
		metrics: p.metrics,
		sql:     sql,
		start:   time.Now(),
	}
}

func (p *metricsPool) Exec(ctx context.Context, sql string, arguments ...any) (CommandTag, error) {
	start := time.Now()
	tag, err := p.Pool.Exec(ctx, sql, arguments...)
	p.metrics.RecordDBQuery(sql, time.Since(start), err)
	return tag, err
}

type metricsRow struct {
	Row
	metrics Metrics
	sql     string
	start   time.Time
}

func (r *metricsRow) Scan(dest ...any) error {
	err := r.Row.Scan(dest...)
	r.metrics.RecordDBQuery(r.sql, time.Since(r.start), err)
	return err
}

type metricsRows struct {
	Rows
	metrics  Metrics
	sql      string
	start    time.Time
	recorded bool
}

func (r *metricsRows) Scan(dest ...any) error {
	err := r.Rows.Scan(dest...)
	if err != nil && !r.recorded {
		r.metrics.RecordDBQuery(r.sql, time.Since(r.start), err)
		r.recorded = true
	}
	return err
}

func (r *metricsRows) Close() {
	r.Rows.Close()
	if !r.recorded {
		r.metrics.RecordDBQuery(r.sql, time.Since(r.start), r.Err())
		r.recorded = true
	}
}
