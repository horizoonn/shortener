//go:build integration

package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/horizoonn/shortener/internal/config"
	pgx_pool "github.com/horizoonn/shortener/internal/storage/postgres/pool/pgx"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	databaseName = "shortener_test"
	databaseUser = "shortener"
	databasePass = "shortener"
)

type Database struct {
	Pool      *pgx_pool.Pool
	container *postgres.PostgresContainer
}

func Start(ctx context.Context) (*Database, error) {
	container, err := postgres.Run(
		ctx,
		"postgres:18-alpine",
		postgres.WithDatabase(databaseName),
		postgres.WithUsername(databaseUser),
		postgres.WithPassword(databasePass),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("get postgres connection string: %w", err)
	}

	pool, err := pgx_pool.NewPool(ctx, config.PostgresConfig{
		URL:             databaseURL,
		Timeout:         5 * time.Second,
		MaxConns:        4,
		MinConns:        0,
		MaxConnIdleTime: time.Minute,
	})
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	db := &Database{
		Pool:      pool,
		container: container,
	}
	if err := db.ApplyMigrations(ctx); err != nil {
		_ = db.Close(ctx)
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	return db, nil
}

func (db *Database) ApplyMigrations(ctx context.Context) error {
	migrationsDir, err := findMigrationsDir()
	if err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("no up migrations found in %s", migrationsDir)
	}

	for _, file := range files {
		query, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		if _, err := db.Pool.Exec(ctx, string(query)); err != nil {
			return fmt.Errorf("execute migration %s: %w", filepath.Base(file), err)
		}
	}

	return nil
}

func (db *Database) Clean(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, "TRUNCATE TABLE clicks, links RESTART IDENTITY CASCADE;")
	if err != nil {
		return fmt.Errorf("clean postgres tables: %w", err)
	}

	return nil
}

func (db *Database) Close(ctx context.Context) error {
	if db.Pool != nil {
		db.Pool.Close()
	}
	if db.container == nil {
		return nil
	}
	if err := db.container.Terminate(ctx); err != nil {
		return fmt.Errorf("terminate postgres container: %w", err)
	}

	return nil
}

func findMigrationsDir() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for dir := workingDir; ; dir = filepath.Dir(dir) {
		migrationsDir := filepath.Join(dir, "migrations")
		if info, err := os.Stat(migrationsDir); err == nil && info.IsDir() {
			return migrationsDir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("migrations directory not found from %s", workingDir)
		}
	}
}
