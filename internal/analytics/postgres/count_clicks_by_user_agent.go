package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	core_errors "github.com/horizoonn/shortener/internal/errors"
)

func (r *Repository) CountClicksByUserAgent(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) ([]analytics.UserAgentCount, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if linkID == uuid.Nil {
		return nil, fmt.Errorf("link id is empty: %w", core_errors.ErrInvalidArgument)
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
	SELECT user_agent, COUNT(*)
	FROM clicks
	`)

	args := appendClickFilter(&queryBuilder, linkID, filter)
	queryBuilder.WriteString(`
	GROUP BY user_agent
	ORDER BY COUNT(*) DESC, user_agent ASC;
	`)

	rows, err := r.pool.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query user-agent click counts: %w", err)
	}
	defer rows.Close()

	counts := make([]analytics.UserAgentCount, 0)
	for rows.Next() {
		var count analytics.UserAgentCount
		if err := rows.Scan(&count.UserAgent, &count.Count); err != nil {
			return nil, fmt.Errorf("scan user-agent click count: %w", err)
		}
		counts = append(counts, count)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user-agent click counts: %w", err)
	}

	return counts, nil
}
