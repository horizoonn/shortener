package service

import (
	"context"
	"errors"
	"fmt"

	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
)

func (s *Service) CreateLink(ctx context.Context, originalURL string, customAlias *string) (links.Link, error) {
	if s == nil {
		return links.Link{}, fmt.Errorf("links service is nil: %w", core_errors.ErrInternal)
	}

	validOriginalURL, err := links.ValidateOriginalURL(originalURL)
	if err != nil {
		return links.Link{}, fmt.Errorf("validate original URL: %w", err)
	}

	if customAlias != nil {
		return s.createCustomLink(ctx, validOriginalURL, *customAlias)
	}

	return s.createGeneratedLink(ctx, validOriginalURL)
}

func (s *Service) createCustomLink(ctx context.Context, originalURL string, customAlias string) (links.Link, error) {
	if err := links.ValidateCustomAlias(customAlias); err != nil {
		return links.Link{}, fmt.Errorf("validate custom alias: %w", err)
	}

	link, err := s.linksRepository.CreateLink(ctx, newLink(customAlias, originalURL, true))
	if err != nil {
		if errors.Is(err, core_errors.ErrConflict) {
			return links.Link{}, fmt.Errorf("custom alias %q already exists: %w", customAlias, core_errors.ErrConflict)
		}

		return links.Link{}, fmt.Errorf("create custom link in repository: %w", err)
	}

	return link, nil
}

func (s *Service) createGeneratedLink(ctx context.Context, originalURL string) (links.Link, error) {
	var lastConflictCode string

	for attempt := 1; attempt <= MaxCodeGenerationAttempts; attempt++ {
		code, err := s.codeGenerator.Generate()
		if err != nil {
			return links.Link{}, fmt.Errorf("generate short code: %w", err)
		}

		link, err := s.linksRepository.CreateLink(ctx, newLink(code, originalURL, false))
		if err == nil {
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
