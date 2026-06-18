package postgres

import (
	"time"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	"github.com/horizoonn/shortener/internal/storage/postgres/pool"
)

type ClickModel struct {
	ID        uuid.UUID
	LinkID    uuid.UUID
	ClickedAt time.Time
	UserAgent string
	Referer   *string
	IP        *string
}

func (m *ClickModel) Scan(row pool.Row) error {
	return row.Scan(
		&m.ID,
		&m.LinkID,
		&m.ClickedAt,
		&m.UserAgent,
		&m.Referer,
		&m.IP,
	)
}

func modelToDomain(model ClickModel) analytics.Click {
	return analytics.Click{
		ID:        model.ID,
		LinkID:    model.LinkID,
		ClickedAt: model.ClickedAt,
		UserAgent: model.UserAgent,
		Referer:   model.Referer,
		IP:        model.IP,
	}
}
