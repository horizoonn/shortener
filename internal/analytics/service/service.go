package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	core_errors "github.com/horizoonn/shortener/internal/errors"
)

const UnknownUserAgent = "unknown"

type Service struct {
	clicksRepository ClicksRepository
}

type ClicksRepository interface {
	SaveClick(ctx context.Context, click analytics.Click) (analytics.Click, error)
}

func NewService(clicksRepository ClicksRepository) *Service {
	return &Service{
		clicksRepository: clicksRepository,
	}
}

func (s *Service) RecordClick(
	ctx context.Context,
	linkID uuid.UUID,
	userAgent string,
	referer *string,
	ip *string,
) error {
	if s == nil {
		return fmt.Errorf("analytics service is nil: %w", core_errors.ErrInternal)
	}
	if s.clicksRepository == nil {
		return fmt.Errorf("clicks repository is nil: %w", core_errors.ErrInternal)
	}
	if linkID == uuid.Nil {
		return fmt.Errorf("link id is empty: %w", core_errors.ErrInvalidArgument)
	}
	if userAgent == "" {
		userAgent = UnknownUserAgent
	}

	click := analytics.Click{
		ID:        uuid.New(),
		LinkID:    linkID,
		UserAgent: userAgent,
		Referer:   referer,
		IP:        ip,
	}

	if _, err := s.clicksRepository.SaveClick(ctx, click); err != nil {
		return fmt.Errorf("save click: %w", err)
	}

	return nil
}
