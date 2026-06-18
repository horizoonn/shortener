package service

import (
	"context"
	"fmt"
	"time"

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
	CountClicks(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) (int64, error)
	CountClicksByDay(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) ([]analytics.TimeBucketCount, error)
	CountClicksByMonth(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) ([]analytics.TimeBucketCount, error)
	CountClicksByUserAgent(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) ([]analytics.UserAgentCount, error)
	RecentClicks(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter, limit int) ([]analytics.Click, error)
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

func (s *Service) GetLinkAnalytics(
	ctx context.Context,
	linkID uuid.UUID,
	filter analytics.ClickFilter,
	recentLimit int,
) (analytics.LinkAnalytics, error) {
	if s == nil {
		return analytics.LinkAnalytics{}, fmt.Errorf("analytics service is nil: %w", core_errors.ErrInternal)
	}
	if s.clicksRepository == nil {
		return analytics.LinkAnalytics{}, fmt.Errorf("clicks repository is nil: %w", core_errors.ErrInternal)
	}
	if linkID == uuid.Nil {
		return analytics.LinkAnalytics{}, fmt.Errorf("link id is empty: %w", core_errors.ErrInvalidArgument)
	}
	if recentLimit <= 0 {
		return analytics.LinkAnalytics{}, fmt.Errorf("recent limit must be positive: %w", core_errors.ErrInvalidArgument)
	}
	if recentLimit > analytics.MaxRecentClicksLimit {
		return analytics.LinkAnalytics{}, fmt.Errorf("recent limit must be less than or equal to %d: %w", analytics.MaxRecentClicksLimit, core_errors.ErrInvalidArgument)
	}

	aggregationFilter := boundedAggregationFilter(filter)

	totalClicks, err := s.clicksRepository.CountClicks(ctx, linkID, filter)
	if err != nil {
		return analytics.LinkAnalytics{}, fmt.Errorf("count clicks: %w", err)
	}

	clicksByDay, err := s.clicksRepository.CountClicksByDay(ctx, linkID, aggregationFilter)
	if err != nil {
		return analytics.LinkAnalytics{}, fmt.Errorf("count clicks by day: %w", err)
	}

	clicksByMonth, err := s.clicksRepository.CountClicksByMonth(ctx, linkID, aggregationFilter)
	if err != nil {
		return analytics.LinkAnalytics{}, fmt.Errorf("count clicks by month: %w", err)
	}

	clicksByUserAgent, err := s.clicksRepository.CountClicksByUserAgent(ctx, linkID, aggregationFilter)
	if err != nil {
		return analytics.LinkAnalytics{}, fmt.Errorf("count clicks by user agent: %w", err)
	}

	recentClicks, err := s.clicksRepository.RecentClicks(ctx, linkID, aggregationFilter, recentLimit)
	if err != nil {
		return analytics.LinkAnalytics{}, fmt.Errorf("get recent clicks: %w", err)
	}

	return analytics.LinkAnalytics{
		TotalClicks:       totalClicks,
		ClicksByDay:       clicksByDay,
		ClicksByMonth:     clicksByMonth,
		ClicksByUserAgent: clicksByUserAgent,
		RecentClicks:      recentClicks,
	}, nil
}

func boundedAggregationFilter(filter analytics.ClickFilter) analytics.ClickFilter {
	if filter.From != nil || filter.To != nil {
		return filter
	}

	from := time.Now().UTC().AddDate(0, 0, -analytics.DefaultAggregationDaysBack)
	return analytics.ClickFilter{
		From: &from,
	}
}
