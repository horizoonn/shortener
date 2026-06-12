package pgx

import (
	"errors"
	"fmt"

	"github.com/horizoonn/shortener/internal/storage/postgres/pool"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	postgresUniqueViolationCode     = "23505"
	postgresForeignKeyViolationCode = "23503"
)

type pgxRows struct {
	pgx.Rows
}

type pgxRow struct {
	pgx.Row
}

func (r pgxRow) Scan(dest ...any) error {
	if err := r.Row.Scan(dest...); err != nil {
		return mapErrors(err)
	}

	return nil
}

type pgxCommandTag struct {
	pgconn.CommandTag
}

func mapErrors(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return pool.ErrNoRows
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case postgresUniqueViolationCode:
			return fmt.Errorf("%v: %w", err, pool.ErrUniqueViolation)
		case postgresForeignKeyViolationCode:
			return fmt.Errorf("%v: %w", err, pool.ErrViolatesForeignKey)
		}
	}

	return fmt.Errorf("%v: %w", err, pool.ErrUnknown)
}
