package service

import (
	"context"
	"errors"
	"fmt"

	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
)

func (s *Service) ResolveLink(ctx context.Context, code string) (links.Link, error) {
	if s == nil {
		return links.Link{}, fmt.Errorf("links service is nil: %w", core_errors.ErrInternal)
	}
	if err := links.ValidateCustomAlias(code); err != nil {
		return links.Link{}, fmt.Errorf("validate link code: %w", err)
	}

	cachedLink, cacheHit, cachedDisabled := s.getCachedLink(ctx, code)
	if cachedDisabled {
		return links.Link{}, fmt.Errorf("cached link with code %q is disabled: %w", code, core_errors.ErrNotFound)
	}
	if cacheHit {
		return cachedLink, nil
	}

	link, err := s.linksRepository.GetLinkByCode(ctx, code)
	if err != nil {
		if errors.Is(err, core_errors.ErrNotFound) {
			return links.Link{}, fmt.Errorf("link with code %q: %w", code, core_errors.ErrNotFound)
		}

		return links.Link{}, fmt.Errorf("get link by code from repository: %w", err)
	}

	if link.DisabledAt != nil {
		s.deleteCachedLink(ctx, code)
		return links.Link{}, fmt.Errorf("link with code %q is disabled: %w", code, core_errors.ErrNotFound)
	}

	s.setCachedLink(ctx, link)
	return link, nil
}

func (s *Service) getCachedLink(ctx context.Context, code string) (links.Link, bool, bool) {
	if s.linkCache == nil {
		return links.Link{}, false, false
	}

	link, err := s.linkCache.GetLink(ctx, code)
	if err != nil {
		return links.Link{}, false, false
	}

	if link.DisabledAt != nil {
		s.deleteCachedLink(ctx, code)
		return links.Link{}, false, true
	}

	return link, true, false
}

func (s *Service) setCachedLink(ctx context.Context, link links.Link) {
	if s.linkCache == nil || link.DisabledAt != nil {
		return
	}

	_ = s.linkCache.SetLink(ctx, link)
}

func (s *Service) deleteCachedLink(ctx context.Context, code string) {
	if s.linkCache == nil {
		return
	}

	_ = s.linkCache.DeleteLink(ctx, code)
}
