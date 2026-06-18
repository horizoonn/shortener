package service

import (
	"context"
	"errors"
	"fmt"

	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
)

func (s *Service) DisableLink(ctx context.Context, code string) (links.Link, error) {
	if s == nil {
		return links.Link{}, fmt.Errorf("links service is nil: %w", core_errors.ErrInternal)
	}
	if s.linksRepository == nil {
		return links.Link{}, fmt.Errorf("links repository is nil: %w", core_errors.ErrInternal)
	}
	if err := links.ValidateCode(code); err != nil {
		return links.Link{}, fmt.Errorf("validate link code: %w", err)
	}

	link, err := s.linksRepository.DisableLink(ctx, code)
	if err != nil {
		if errors.Is(err, core_errors.ErrNotFound) {
			return links.Link{}, fmt.Errorf("link with code %q: %w", code, core_errors.ErrNotFound)
		}

		return links.Link{}, fmt.Errorf("disable link in repository: %w", err)
	}

	s.deleteCachedLink(ctx, code)
	return link, nil
}
