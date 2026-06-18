package pgx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/horizoonn/shortener/internal/config"
	"github.com/horizoonn/shortener/internal/storage/postgres/pool"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	pingAttempts = 5
	pingBackoff  = 500 * time.Millisecond
)

type Pool struct {
	*pgxpool.Pool

	opTimeout time.Duration
}

func NewPool(ctx context.Context, cfg config.PostgresConfig) (*Pool, error) {
	pgxConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse pgxpool config: %w", err)
	}

	pgxConfig.MaxConns = cfg.MaxConns
	pgxConfig.MinConns = cfg.MinConns
	pgxConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	if cfg.TimeZone != "" {
		if pgxConfig.ConnConfig.RuntimeParams == nil {
			pgxConfig.ConnConfig.RuntimeParams = make(map[string]string)
		}
		pgxConfig.ConnConfig.RuntimeParams["TimeZone"] = cfg.TimeZone
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		return nil, fmt.Errorf("create pgxpool with config: %w", err)
	}
	if err := pingWithRetry(ctx, pool, cfg.Timeout); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping pgxpool: %w", err)
	}

	return &Pool{
		Pool:      pool,
		opTimeout: cfg.Timeout,
	}, nil
}

func pingWithRetry(ctx context.Context, pool *pgxpool.Pool, timeout time.Duration) error {
	var lastErr error

	for attempt := 1; attempt <= pingAttempts; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, timeout)
		err := pool.Ping(pingCtx)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ctx.Err()
		}
		if attempt == pingAttempts {
			break
		}

		timer := time.NewTimer(pingBackoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return lastErr
}

func (p *Pool) Query(ctx context.Context, sql string, args ...any) (pool.Rows, error) {
	rows, err := p.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, mapErrors(err)
	}

	return pgxRows{rows}, nil
}

func (p *Pool) QueryRow(ctx context.Context, sql string, args ...any) pool.Row {
	row := p.Pool.QueryRow(ctx, sql, args...)

	return pgxRow{row}
}

func (p *Pool) Exec(ctx context.Context, sql string, arguments ...any) (pool.CommandTag, error) {
	tag, err := p.Pool.Exec(ctx, sql, arguments...)
	if err != nil {
		return nil, mapErrors(err)
	}

	return pgxCommandTag{tag}, nil
}

func (p *Pool) Ping(ctx context.Context) error {
	return mapErrors(p.Pool.Ping(ctx))
}

func (p *Pool) OpTimeout() time.Duration {
	return p.opTimeout
}
