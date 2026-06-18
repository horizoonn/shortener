package service

import (
	"context"
	"errors"
	"fmt"

	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
)

func (s *Service) GetLink(ctx context.Context, code string) (links.Link, error) {
	if s == nil {
		return links.Link{}, fmt.Errorf("links service is nil: %w", core_errors.ErrInternal)
	}
	if s.linksRepository == nil {
		return links.Link{}, fmt.Errorf("links repository is nil: %w", core_errors.ErrInternal)
	}
	if err := links.ValidateCode(code); err != nil {
		return links.Link{}, fmt.Errorf("validate link code: %w", err)
	}

	link, err := s.linksRepository.GetLinkByCode(ctx, code)
	if err != nil {
		if errors.Is(err, core_errors.ErrNotFound) {
			return links.Link{}, fmt.Errorf("link with code %q: %w", code, core_errors.ErrNotFound)
		}

		return links.Link{}, fmt.Errorf("get link by code from repository: %w", err)
	}

	return link, nil
}
