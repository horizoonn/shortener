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

func (r *Repository) CreateLink(ctx context.Context, link links.Link) (links.Link, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if link.ID == uuid.Nil {
		return links.Link{}, fmt.Errorf("link id is empty: %w", core_errors.ErrInvalidArgument)
	}
	if link.Code == "" {
		return links.Link{}, fmt.Errorf("link code is empty: %w", core_errors.ErrInvalidArgument)
	}
	if link.OriginalURL == "" {
		return links.Link{}, fmt.Errorf("original URL is empty: %w", core_errors.ErrInvalidArgument)
	}

	query := `
	INSERT INTO links (id, code, original_url, is_custom, disabled_at, expires_at)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, code, original_url, is_custom, created_at, disabled_at, expires_at;
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		link.ID,
		link.Code,
		link.OriginalURL,
		link.IsCustom,
		link.DisabledAt,
		link.ExpiresAt,
	)

	var linkModel LinkModel
	if err := linkModel.Scan(row); err != nil {
		if errors.Is(err, pool.ErrUniqueViolation) {
			return links.Link{}, fmt.Errorf("link code %q already exists: %w", link.Code, core_errors.ErrConflict)
		}

		return links.Link{}, fmt.Errorf("scan created link: %w", err)
	}

	return modelToDomain(linkModel), nil
}
