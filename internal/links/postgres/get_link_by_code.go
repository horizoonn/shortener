package postgres

import (
	"context"
	"errors"
	"fmt"

	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
	"github.com/horizoonn/shortener/internal/storage/postgres/pool"
)

func (r *Repository) GetLinkByCode(ctx context.Context, code string) (links.Link, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if code == "" {
		return links.Link{}, fmt.Errorf("link code is empty: %w", core_errors.ErrInvalidArgument)
	}

	query := `
	SELECT id, code, original_url, is_custom, created_at, disabled_at, expires_at
	FROM links
	WHERE code=$1;
	`

	row := r.pool.QueryRow(ctx, query, code)

	var linkModel LinkModel
	if err := linkModel.Scan(row); err != nil {
		if errors.Is(err, pool.ErrNoRows) {
			return links.Link{}, fmt.Errorf("link with code=%q: %w", code, core_errors.ErrNotFound)
		}

		return links.Link{}, fmt.Errorf("scan link by code: %w", err)
	}

	return modelToDomain(linkModel), nil
}
