package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
)

func (s *Service) ResolveLink(ctx context.Context, code string) (links.Link, error) {
	if s == nil {
		return links.Link{}, fmt.Errorf("links service is nil: %w", core_errors.ErrInternal)
	}
	if s.linksRepository == nil {
		return links.Link{}, fmt.Errorf("links repository is nil: %w", core_errors.ErrInternal)
	}
	if err := links.ValidateCode(code); err != nil {
		return links.Link{}, fmt.Errorf("validate link code: %w", err)
	}

	cachedLink, cacheHit, cachedDisabled, cachedNotFound := s.getCachedLink(ctx, code)
	if cachedNotFound {
		return links.Link{}, fmt.Errorf("cached link with code %q does not exist: %w", code, core_errors.ErrNotFound)
	}
	if cachedDisabled {
		s.setCachedLinkNotFound(ctx, code)
		return links.Link{}, fmt.Errorf("cached link with code %q is disabled: %w", code, core_errors.ErrNotFound)
	}
	if cacheHit {
		if isExpired(cachedLink) {
			s.deleteCachedLink(ctx, code)
			s.setCachedLinkNotFound(ctx, code)
			return links.Link{}, fmt.Errorf("link with code %q is expired: %w", code, core_errors.ErrNotFound)
		}

		if s.metrics != nil {
			s.metrics.RecordLinkResolved()
		}
		return cachedLink, nil
	}

	link, err := s.linksRepository.GetLinkByCode(ctx, code)
	if err != nil {
		if errors.Is(err, core_errors.ErrNotFound) {
			s.setCachedLinkNotFound(ctx, code)
			return links.Link{}, fmt.Errorf("link with code %q: %w", code, core_errors.ErrNotFound)
		}

		return links.Link{}, fmt.Errorf("get link by code from repository: %w", err)
	}

	if link.DisabledAt != nil {
		s.deleteCachedLink(ctx, code)
		s.setCachedLinkNotFound(ctx, code)
		return links.Link{}, fmt.Errorf("link with code %q is disabled: %w", code, core_errors.ErrNotFound)
	}

	if isExpired(link) {
		s.deleteCachedLink(ctx, code)
		s.setCachedLinkNotFound(ctx, code)
		return links.Link{}, fmt.Errorf("link with code %q is expired: %w", code, core_errors.ErrNotFound)
	}

	s.setCachedLink(ctx, link)
	if s.metrics != nil {
		s.metrics.RecordLinkResolved()
	}
	return link, nil
}

func (s *Service) getCachedLink(ctx context.Context, code string) (links.Link, bool, bool, bool) {
	if s.linkCache == nil {
		return links.Link{}, false, false, false
	}

	link, err := s.linkCache.GetLink(ctx, code)
	if err != nil {
		if errors.Is(err, core_errors.ErrNotFound) {
			return links.Link{}, false, false, true
		}

		return links.Link{}, false, false, false
	}

	if link.DisabledAt != nil {
		s.deleteCachedLink(ctx, code)
		return links.Link{}, false, true, false
	}

	return link, true, false, false
}

func isExpired(link links.Link) bool {
	return link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt)
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

func (s *Service) setCachedLinkNotFound(ctx context.Context, code string) {
	if s.linkCache == nil {
		return
	}

	_ = s.linkCache.SetLinkNotFound(ctx, code)
}
