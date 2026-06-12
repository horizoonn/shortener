package pgx

import (
	"context"
	"fmt"
	"time"

	"github.com/horizoonn/shortener/internal/config"
	"github.com/horizoonn/shortener/internal/storage/postgres/pool"
	"github.com/jackc/pgx/v5/pgxpool"
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

	pool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		return nil, fmt.Errorf("create pgxpool with config: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping pgxpool: %w", err)
	}

	return &Pool{
		Pool:      pool,
		opTimeout: cfg.Timeout,
	}, nil
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

func (p *Pool) OpTimeout() time.Duration {
	return p.opTimeout
}
