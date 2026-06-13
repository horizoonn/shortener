package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	core_errors "github.com/horizoonn/shortener/internal/errors"
)

func (r *Repository) CountClicksByDay(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) ([]analytics.TimeBucketCount, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if linkID == uuid.Nil {
		return nil, fmt.Errorf("link id is empty: %w", core_errors.ErrInvalidArgument)
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
	SELECT date_trunc('day', clicked_at) AS bucket, COUNT(*)
	FROM clicks
	`)

	args := appendClickFilter(&queryBuilder, linkID, filter)
	queryBuilder.WriteString(`
	GROUP BY bucket
	ORDER BY bucket;
	`)

	rows, err := r.pool.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query daily click counts: %w", err)
	}
	defer rows.Close()

	counts := make([]analytics.TimeBucketCount, 0)
	for rows.Next() {
		var count analytics.TimeBucketCount
		if err := rows.Scan(&count.Bucket, &count.Count); err != nil {
			return nil, fmt.Errorf("scan daily click count: %w", err)
		}
		counts = append(counts, count)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate daily click counts: %w", err)
	}

	return counts, nil
}
