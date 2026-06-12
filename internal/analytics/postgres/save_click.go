package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/storage/postgres/pool"
)

func (r *Repository) SaveClick(ctx context.Context, click analytics.Click) (analytics.Click, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if click.ID == uuid.Nil {
		return analytics.Click{}, fmt.Errorf("click id is empty: %w", core_errors.ErrInvalidArgument)
	}
	if click.LinkID == uuid.Nil {
		return analytics.Click{}, fmt.Errorf("link id is empty: %w", core_errors.ErrInvalidArgument)
	}
	if click.UserAgent == "" {
		return analytics.Click{}, fmt.Errorf("user agent is empty: %w", core_errors.ErrInvalidArgument)
	}

	query := `
	INSERT INTO clicks (id, link_id, clicked_at, user_agent, referer, ip)
	VALUES ($1, $2, $3, $4, $5, CAST($6 AS inet))
	RETURNING id, link_id, clicked_at, user_agent, referer, ip::text;
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		click.ID,
		click.LinkID,
		click.ClickedAt,
		click.UserAgent,
		click.Referer,
		click.IP,
	)

	var clickModel ClickModel
	if err := clickModel.Scan(row); err != nil {
		if errors.Is(err, pool.ErrViolatesForeignKey) {
			return analytics.Click{}, fmt.Errorf("link with id=%q: %w", click.LinkID, core_errors.ErrInvalidArgument)
		}

		return analytics.Click{}, fmt.Errorf("scan saved click: %w", err)
	}

	return modelToDomain(clickModel), nil
}
