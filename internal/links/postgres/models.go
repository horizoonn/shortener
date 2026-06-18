package postgres

import (
	"time"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/links"
	"github.com/horizoonn/shortener/internal/storage/postgres/pool"
)

type LinkModel struct {
	ID          uuid.UUID
	Code        string
	OriginalURL string
	IsCustom    bool
	CreatedAt   time.Time
	DisabledAt  *time.Time
}

func (m *LinkModel) Scan(row pool.Row) error {
	return row.Scan(
		&m.ID,
		&m.Code,
		&m.OriginalURL,
		&m.IsCustom,
		&m.CreatedAt,
		&m.DisabledAt,
	)
}

func modelToDomain(model LinkModel) links.Link {
	return links.Link{
		ID:          model.ID,
		Code:        model.Code,
		OriginalURL: model.OriginalURL,
		IsCustom:    model.IsCustom,
		CreatedAt:   model.CreatedAt,
		DisabledAt:  model.DisabledAt,
	}
}
