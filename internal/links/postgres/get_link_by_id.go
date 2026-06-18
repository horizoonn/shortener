package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
	"github.com/horizoonn/shortener/internal/storage/postgres/pool"
)

func (r *Repository) GetLinkByID(ctx context.Context, id uuid.UUID) (links.Link, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if id == uuid.Nil {
		return links.Link{}, fmt.Errorf("link id is empty: %w", core_errors.ErrInvalidArgument)
	}

	query := `
	SELECT id, code, original_url, is_custom, created_at, disabled_at
	FROM links
	WHERE id=$1;
	`

	row := r.pool.QueryRow(ctx, query, id)

	var linkModel LinkModel
	if err := linkModel.Scan(row); err != nil {
		if errors.Is(err, pool.ErrNoRows) {
			return links.Link{}, fmt.Errorf("link with id=%q: %w", id, core_errors.ErrNotFound)
		}

		return links.Link{}, fmt.Errorf("scan link by id: %w", err)
	}

	return modelToDomain(linkModel), nil
}
