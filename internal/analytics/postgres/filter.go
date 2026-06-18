package postgres

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
)

func appendClickFilter(queryBuilder *strings.Builder, args []any, linkID uuid.UUID, filter analytics.ClickFilter) []any {
	args = append(args, linkID)
	conditions := []string{fmt.Sprintf("link_id=$%d", len(args))}

	if filter.From != nil {
		args = append(args, *filter.From)
		conditions = append(conditions, fmt.Sprintf("clicked_at>=$%d", len(args)))
	}

	if filter.To != nil {
		args = append(args, *filter.To)
		conditions = append(conditions, fmt.Sprintf("clicked_at<$%d", len(args)))
	}

	queryBuilder.WriteString(" WHERE " + strings.Join(conditions, " AND "))

	return args
}
