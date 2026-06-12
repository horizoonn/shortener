package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	core_errors "github.com/horizoonn/shortener/internal/errors"
)

func (r *Repository) RecentClicks(ctx context.Context, linkID uuid.UUID, limit int) ([]analytics.Click, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if linkID == uuid.Nil {
		return nil, fmt.Errorf("link id is empty: %w", core_errors.ErrInvalidArgument)
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive: %w", core_errors.ErrInvalidArgument)
	}

	query := `
	SELECT id, link_id, clicked_at, user_agent, referer, ip::text
	FROM clicks
	WHERE link_id=$1
	ORDER BY clicked_at DESC
	LIMIT $2;
	`

	rows, err := r.pool.Query(ctx, query, linkID, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent clicks: %w", err)
	}
	defer rows.Close()

	clicks := make([]analytics.Click, 0)
	for rows.Next() {
		var clickModel ClickModel
		if err := rows.Scan(
			&clickModel.ID,
			&clickModel.LinkID,
			&clickModel.ClickedAt,
			&clickModel.UserAgent,
			&clickModel.Referer,
			&clickModel.IP,
		); err != nil {
			return nil, fmt.Errorf("scan recent click: %w", err)
		}
		clicks = append(clicks, modelToDomain(clickModel))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent clicks: %w", err)
	}

	return clicks, nil
}
