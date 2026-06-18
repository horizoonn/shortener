package postgres

import "github.com/horizoonn/shortener/internal/storage/postgres/pool"

type Repository struct {
	pool pool.Pool
}

func NewRepository(pool pool.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}
