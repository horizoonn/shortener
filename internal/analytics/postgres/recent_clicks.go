package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	core_errors "github.com/horizoonn/shortener/internal/errors"
)

func (r *Repository) RecentClicks(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter, limit int) ([]analytics.Click, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if linkID == uuid.Nil {
		return nil, fmt.Errorf("link id is empty: %w", core_errors.ErrInvalidArgument)
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive: %w", core_errors.ErrInvalidArgument)
	}
	if limit > analytics.MaxRecentClicksLimit {
		return nil, fmt.Errorf("limit must be less than or equal to %d: %w", analytics.MaxRecentClicksLimit, core_errors.ErrInvalidArgument)
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
	SELECT id, link_id, clicked_at, user_agent, referer, ip::text
	FROM clicks
	`)
	args := appendClickFilter(&queryBuilder, nil, linkID, filter)
	args = append(args, limit)
	queryBuilder.WriteString(fmt.Sprintf(`
	ORDER BY clicked_at DESC
	LIMIT $%d;
	`, len(args)))

	rows, err := r.pool.Query(ctx, queryBuilder.String(), args...)
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
