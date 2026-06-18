package pgx

import (
	"errors"
	"testing"

	"github.com/horizoonn/shortener/internal/storage/postgres/pool"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "no rows",
			err:  pgx.ErrNoRows,
			want: pool.ErrNoRows,
		},
		{
			name: "unique violation",
			err:  &pgconn.PgError{Code: postgresUniqueViolationCode},
			want: pool.ErrUniqueViolation,
		},
		{
			name: "foreign key violation",
			err:  &pgconn.PgError{Code: postgresForeignKeyViolationCode},
			want: pool.ErrViolatesForeignKey,
		},
		{
			name: "invalid text",
			err:  &pgconn.PgError{Code: postgresInvalidTextCode},
			want: pool.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := mapErrors(tt.err); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}
