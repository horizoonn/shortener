package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	core_errors "github.com/horizoonn/shortener/internal/errors"
)

type fakeClicksRepository struct {
	saveClick              func(ctx context.Context, click analytics.Click) (analytics.Click, error)
	countClicks            func(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) (int64, error)
	countClicksByDay       func(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) ([]analytics.TimeBucketCount, error)
	countClicksByMonth     func(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) ([]analytics.TimeBucketCount, error)
	countClicksByUserAgent func(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) ([]analytics.UserAgentCount, error)
	recentClicks           func(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter, limit int) ([]analytics.Click, error)
}

func (r fakeClicksRepository) SaveClick(ctx context.Context, click analytics.Click) (analytics.Click, error) {
	if r.saveClick == nil {
		return analytics.Click{}, fmt.Errorf("save click not implemented")
	}

	return r.saveClick(ctx, click)
}

func (r fakeClicksRepository) CountClicks(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) (int64, error) {
	if r.countClicks == nil {
		return 0, fmt.Errorf("count clicks not implemented")
	}

	return r.countClicks(ctx, linkID, filter)
}

func (r fakeClicksRepository) CountClicksByDay(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) ([]analytics.TimeBucketCount, error) {
	if r.countClicksByDay == nil {
		return nil, fmt.Errorf("count clicks by day not implemented")
	}

	return r.countClicksByDay(ctx, linkID, filter)
}

func (r fakeClicksRepository) CountClicksByMonth(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) ([]analytics.TimeBucketCount, error) {
	if r.countClicksByMonth == nil {
		return nil, fmt.Errorf("count clicks by month not implemented")
	}

	return r.countClicksByMonth(ctx, linkID, filter)
}

func (r fakeClicksRepository) CountClicksByUserAgent(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter) ([]analytics.UserAgentCount, error) {
	if r.countClicksByUserAgent == nil {
		return nil, fmt.Errorf("count clicks by user agent not implemented")
	}

	return r.countClicksByUserAgent(ctx, linkID, filter)
}

func (r fakeClicksRepository) RecentClicks(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter, limit int) ([]analytics.Click, error) {
	if r.recentClicks == nil {
		return nil, fmt.Errorf("recent clicks not implemented")
	}

	return r.recentClicks(ctx, linkID, filter, limit)
}

func TestServiceRecordClickSuccess(t *testing.T) {
	t.Parallel()

	linkID := uuid.New()
	referer := "https://example.com/source"
	ip := "192.0.2.10"
	repository := fakeClicksRepository{
		saveClick: func(_ context.Context, click analytics.Click) (analytics.Click, error) {
			if click.ID == uuid.Nil {
				t.Fatal("expected generated click ID")
			}
			if click.LinkID != linkID {
				t.Fatalf("expected link ID %s, got %s", linkID, click.LinkID)
			}
			if click.UserAgent != "test-agent" {
				t.Fatalf("expected user agent test-agent, got %q", click.UserAgent)
			}
			if click.Referer == nil || *click.Referer != referer {
				t.Fatalf("expected referer %q, got %v", referer, click.Referer)
			}
			if click.IP == nil || *click.IP != ip {
				t.Fatalf("expected IP %q, got %v", ip, click.IP)
			}

			return click, nil
		},
	}
	service := NewService(repository)

	if err := service.RecordClick(context.Background(), linkID, "test-agent", &referer, &ip); err != nil {
		t.Fatalf("record click: %v", err)
	}
}

func TestServiceRecordClickUsesUnknownUserAgent(t *testing.T) {
	t.Parallel()

	repository := fakeClicksRepository{
		saveClick: func(_ context.Context, click analytics.Click) (analytics.Click, error) {
			if click.UserAgent != UnknownUserAgent {
				t.Fatalf("expected user agent %q, got %q", UnknownUserAgent, click.UserAgent)
			}

			return click, nil
		},
	}
	service := NewService(repository)

	if err := service.RecordClick(context.Background(), uuid.New(), "", nil, nil); err != nil {
		t.Fatalf("record click: %v", err)
	}
}

func TestServiceRecordClickInvalidLinkID(t *testing.T) {
	t.Parallel()

	service := NewService(fakeClicksRepository{})

	err := service.RecordClick(context.Background(), uuid.Nil, "agent", nil, nil)
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestServiceGetLinkAnalyticsSuccess(t *testing.T) {
	t.Parallel()

	linkID := uuid.New()
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	filter := analytics.ClickFilter{From: &from, To: &to}
	clickedAt := time.Date(2026, 6, 12, 12, 34, 56, 0, time.UTC)
	repository := fakeClicksRepository{
		countClicks: func(_ context.Context, gotLinkID uuid.UUID, gotFilter analytics.ClickFilter) (int64, error) {
			assertAnalyticsRequest(t, gotLinkID, linkID, gotFilter, filter)
			return 42, nil
		},
		countClicksByDay: func(_ context.Context, gotLinkID uuid.UUID, gotFilter analytics.ClickFilter) ([]analytics.TimeBucketCount, error) {
			assertAnalyticsRequest(t, gotLinkID, linkID, gotFilter, filter)
			return []analytics.TimeBucketCount{{Bucket: clickedAt, Count: 10}}, nil
		},
		countClicksByMonth: func(_ context.Context, gotLinkID uuid.UUID, gotFilter analytics.ClickFilter) ([]analytics.TimeBucketCount, error) {
			assertAnalyticsRequest(t, gotLinkID, linkID, gotFilter, filter)
			return []analytics.TimeBucketCount{{Bucket: clickedAt, Count: 42}}, nil
		},
		countClicksByUserAgent: func(_ context.Context, gotLinkID uuid.UUID, gotFilter analytics.ClickFilter) ([]analytics.UserAgentCount, error) {
			assertAnalyticsRequest(t, gotLinkID, linkID, gotFilter, filter)
			return []analytics.UserAgentCount{{UserAgent: "Mozilla/5.0", Count: 30}}, nil
		},
		recentClicks: func(_ context.Context, gotLinkID uuid.UUID, gotFilter analytics.ClickFilter, limit int) ([]analytics.Click, error) {
			if gotLinkID != linkID {
				t.Fatalf("expected link ID %s, got %s", linkID, gotLinkID)
			}
			if gotFilter != filter {
				t.Fatalf("expected filter %+v, got %+v", filter, gotFilter)
			}
			if limit != 20 {
				t.Fatalf("expected limit 20, got %d", limit)
			}
			return []analytics.Click{{ID: uuid.New(), LinkID: linkID, ClickedAt: clickedAt, UserAgent: "Mozilla/5.0"}}, nil
		},
	}
	service := NewService(repository)

	result, err := service.GetLinkAnalytics(context.Background(), linkID, filter, 20)
	if err != nil {
		t.Fatalf("get link analytics: %v", err)
	}
	if result.TotalClicks != 42 {
		t.Fatalf("expected total clicks 42, got %d", result.TotalClicks)
	}
	if len(result.ClicksByDay) != 1 || result.ClicksByDay[0].Count != 10 {
		t.Fatalf("unexpected day counts: %+v", result.ClicksByDay)
	}
	if len(result.ClicksByMonth) != 1 || result.ClicksByMonth[0].Count != 42 {
		t.Fatalf("unexpected month counts: %+v", result.ClicksByMonth)
	}
	if len(result.ClicksByUserAgent) != 1 || result.ClicksByUserAgent[0].Count != 30 {
		t.Fatalf("unexpected user-agent counts: %+v", result.ClicksByUserAgent)
	}
	if len(result.RecentClicks) != 1 || result.RecentClicks[0].ClickedAt != clickedAt {
		t.Fatalf("unexpected recent clicks: %+v", result.RecentClicks)
	}
}

func TestServiceGetLinkAnalyticsInvalidLimit(t *testing.T) {
	t.Parallel()

	service := NewService(fakeClicksRepository{})

	_, err := service.GetLinkAnalytics(context.Background(), uuid.New(), analytics.ClickFilter{}, 0)
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func assertAnalyticsRequest(
	t *testing.T,
	gotLinkID uuid.UUID,
	wantLinkID uuid.UUID,
	gotFilter analytics.ClickFilter,
	wantFilter analytics.ClickFilter,
) {
	t.Helper()

	if gotLinkID != wantLinkID {
		t.Fatalf("expected link ID %s, got %s", wantLinkID, gotLinkID)
	}
	if gotFilter != wantFilter {
		t.Fatalf("expected filter %+v, got %+v", wantFilter, gotFilter)
	}
}
