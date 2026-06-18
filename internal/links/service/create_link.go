package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
)

func (s *Service) CreateLink(ctx context.Context, originalURL string, customAlias *string, expiresAt *time.Time) (links.Link, error) {
	if s == nil {
		return links.Link{}, fmt.Errorf("links service is nil: %w", core_errors.ErrInternal)
	}
	if s.linksRepository == nil {
		return links.Link{}, fmt.Errorf("links repository is nil: %w", core_errors.ErrInternal)
	}

	validOriginalURL, err := links.ValidateOriginalURL(originalURL)
	if err != nil {
		return links.Link{}, fmt.Errorf("validate original URL: %w", err)
	}

	if expiresAt != nil && time.Now().After(*expiresAt) {
		return links.Link{}, fmt.Errorf("expiration time is in the past: %w", core_errors.ErrInvalidArgument)
	}

	if customAlias != nil {
		return s.createCustomLink(ctx, validOriginalURL, *customAlias, expiresAt)
	}

	return s.createGeneratedLink(ctx, validOriginalURL, expiresAt)
}

func (s *Service) createCustomLink(ctx context.Context, originalURL string, customAlias string, expiresAt *time.Time) (links.Link, error) {
	if err := links.ValidateCustomAlias(customAlias); err != nil {
		return links.Link{}, fmt.Errorf("validate custom alias: %w", err)
	}

	link, err := s.linksRepository.CreateLink(ctx, newLink(customAlias, originalURL, true, expiresAt))
	if err != nil {
		if errors.Is(err, core_errors.ErrConflict) {
			return links.Link{}, fmt.Errorf("custom alias %q already exists: %w", customAlias, core_errors.ErrConflict)
		}

		return links.Link{}, fmt.Errorf("create custom link in repository: %w", err)
	}

	s.deleteCachedLink(ctx, customAlias)
	if s.metrics != nil {
		s.metrics.RecordLinkCreated(true)
	}
	return link, nil
}

func (s *Service) createGeneratedLink(ctx context.Context, originalURL string, expiresAt *time.Time) (links.Link, error) {
	if s.codeGenerator == nil {
		return links.Link{}, fmt.Errorf("code generator is nil: %w", core_errors.ErrInternal)
	}

	var lastConflictCode string

	for attempt := 1; attempt <= MaxCodeGenerationAttempts; attempt++ {
		code, err := s.codeGenerator.Generate()
		if err != nil {
			return links.Link{}, fmt.Errorf("generate short code: %w", err)
		}

		link, err := s.linksRepository.CreateLink(ctx, newLink(code, originalURL, false, expiresAt))
		if err == nil {
			s.deleteCachedLink(ctx, code)
			if s.metrics != nil {
				s.metrics.RecordLinkCreated(false)
			}
			return link, nil
		}

		if errors.Is(err, core_errors.ErrConflict) {
			lastConflictCode = code
			continue
		}

		return links.Link{}, fmt.Errorf("create generated link in repository: %w", err)
	}

	return links.Link{}, fmt.Errorf(
		"short code collision after %d attempts, last code %q: %w",
		MaxCodeGenerationAttempts,
		lastConflictCode,
		core_errors.ErrConflict,
	)
}
