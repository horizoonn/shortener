package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	core_errors "github.com/horizoonn/shortener/internal/errors"
)

func (r *Repository) CountClicks(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if linkID == uuid.Nil {
		return 0, fmt.Errorf("link id is empty: %w", core_errors.ErrInvalidArgument)
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
	SELECT COUNT(*)
	FROM clicks
	`)

	args := appendClickFilter(&queryBuilder, nil, linkID, filter)

	row := r.pool.QueryRow(ctx, queryBuilder.String(), args...)

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("scan click count: %w", err)
	}

	return count, nil
}
