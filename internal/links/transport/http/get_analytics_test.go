package http

import (
	"context"
	"errors"
	"fmt"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
)

type fakeAnalyticsReader struct {
	getLinkAnalytics func(ctx context.Context, linkID uuid.UUID, filter analytics.ClickFilter, recentLimit int) (analytics.LinkAnalytics, error)
	calls            int
}

func (r *fakeAnalyticsReader) GetLinkAnalytics(
	ctx context.Context,
	linkID uuid.UUID,
	filter analytics.ClickFilter,
	recentLimit int,
) (analytics.LinkAnalytics, error) {
	r.calls++
	if r.getLinkAnalytics == nil {
		return analytics.LinkAnalytics{}, fmt.Errorf("get link analytics not implemented")
	}

	return r.getLinkAnalytics(ctx, linkID, filter, recentLimit)
}

func TestHandlerGetAnalyticsSuccess(t *testing.T) {
	t.Parallel()

	linkID := uuid.New()
	clickedAt := time.Date(2026, 6, 12, 12, 34, 56, 0, time.UTC)
	referer := "https://example.org"
	ip := "127.0.0.1"
	reader := &fakeAnalyticsReader{
		getLinkAnalytics: func(_ context.Context, gotLinkID uuid.UUID, filter analytics.ClickFilter, recentLimit int) (analytics.LinkAnalytics, error) {
			if gotLinkID != linkID {
				t.Fatalf("expected link ID %s, got %s", linkID, gotLinkID)
			}
			if filter.From == nil || !filter.From.Equal(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)) {
				t.Fatalf("unexpected from filter: %v", filter.From)
			}
			if filter.To == nil || !filter.To.Equal(time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)) {
				t.Fatalf("expected inclusive to date to become 2026-06-13, got %v", filter.To)
			}
			if recentLimit != 5 {
				t.Fatalf("expected recent limit 5, got %d", recentLimit)
			}

			return analytics.LinkAnalytics{
				TotalClicks: 42,
				ClicksByDay: []analytics.TimeBucketCount{
					{Bucket: time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC), Count: 10},
				},
				ClicksByMonth: []analytics.TimeBucketCount{
					{Bucket: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), Count: 42},
				},
				ClicksByUserAgent: []analytics.UserAgentCount{
					{UserAgent: "Mozilla/5.0", Count: 30},
				},
				RecentClicks: []analytics.Click{
					{
						ID:        uuid.New(),
						LinkID:    linkID,
						ClickedAt: clickedAt,
						UserAgent: "Mozilla/5.0",
						Referer:   &referer,
						IP:        &ip,
					},
				},
			}, nil
		},
	}
	handler := NewHandlerWithDependencies(fakeLinksService{
		getLink: func(_ context.Context, code string) (links.Link, error) {
			if code != "abc12345" {
				t.Fatalf("expected code abc12345, got %q", code)
			}

			return links.Link{
				ID:          linkID,
				Code:        code,
				OriginalURL: "https://example.com",
			}, nil
		},
	}, nil, reader, "http://localhost:8080")

	rec := executeGetAnalyticsRequest(t, handler, "abc12345", "from=2026-06-01&to=2026-06-12&recent_limit=5")

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", nethttp.StatusOK, rec.Code, rec.Body.String())
	}

	var response AnalyticsResponse
	decodeResponse(t, rec, &response)
	if response.Code != "abc12345" {
		t.Fatalf("expected code abc12345, got %q", response.Code)
	}
	if response.OriginalURL != "https://example.com" {
		t.Fatalf("expected original URL, got %q", response.OriginalURL)
	}
	if response.TotalClicks != 42 {
		t.Fatalf("expected total clicks 42, got %d", response.TotalClicks)
	}
	if len(response.ClicksByDay) != 1 || response.ClicksByDay[0].Day != "2026-06-12" || response.ClicksByDay[0].Clicks != 10 {
		t.Fatalf("unexpected day counts: %+v", response.ClicksByDay)
	}
	if len(response.ClicksByMonth) != 1 || response.ClicksByMonth[0].Month != "2026-06" || response.ClicksByMonth[0].Clicks != 42 {
		t.Fatalf("unexpected month counts: %+v", response.ClicksByMonth)
	}
	if len(response.ClicksByUserAgent) != 1 ||
		response.ClicksByUserAgent[0].UserAgent != "Mozilla/5.0" ||
		response.ClicksByUserAgent[0].Clicks != 30 {
		t.Fatalf("unexpected user-agent counts: %+v", response.ClicksByUserAgent)
	}
	if len(response.RecentClicks) != 1 ||
		!response.RecentClicks[0].ClickedAt.Equal(clickedAt) ||
		response.RecentClicks[0].UserAgent != "Mozilla/5.0" ||
		response.RecentClicks[0].Referer == nil ||
		*response.RecentClicks[0].Referer != referer ||
		response.RecentClicks[0].IP == nil ||
		*response.RecentClicks[0].IP != ip {
		t.Fatalf("unexpected recent clicks: %+v", response.RecentClicks)
	}
}

func TestHandlerGetAnalyticsLinkNotFound(t *testing.T) {
	t.Parallel()

	reader := &fakeAnalyticsReader{}
	handler := NewHandlerWithDependencies(fakeLinksService{
		getLink: func(_ context.Context, _ string) (links.Link, error) {
			return links.Link{}, fmt.Errorf("link not found: %w", core_errors.ErrNotFound)
		},
	}, nil, reader, "http://localhost:8080")

	rec := executeGetAnalyticsRequest(t, handler, "missing1", "")

	assertErrorResponse(t, rec, nethttp.StatusNotFound, "not_found")
	if reader.calls != 0 {
		t.Fatalf("expected analytics reader not to be called, got %d calls", reader.calls)
	}
}

func TestHandlerGetAnalyticsDisabledLinkSuccess(t *testing.T) {
	t.Parallel()

	linkID := uuid.New()
	disabledAt := time.Now().UTC()
	reader := &fakeAnalyticsReader{
		getLinkAnalytics: func(_ context.Context, gotLinkID uuid.UUID, _ analytics.ClickFilter, _ int) (analytics.LinkAnalytics, error) {
			if gotLinkID != linkID {
				t.Fatalf("expected link ID %s, got %s", linkID, gotLinkID)
			}

			return analytics.LinkAnalytics{TotalClicks: 7}, nil
		},
	}
	handler := NewHandlerWithDependencies(fakeLinksService{
		getLink: func(_ context.Context, code string) (links.Link, error) {
			return links.Link{
				ID:          linkID,
				Code:        code,
				OriginalURL: "https://example.com/disabled",
				DisabledAt:  &disabledAt,
			}, nil
		},
	}, nil, reader, "http://localhost:8080")

	rec := executeGetAnalyticsRequest(t, handler, "disabled1", "")

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", nethttp.StatusOK, rec.Code, rec.Body.String())
	}

	var response AnalyticsResponse
	decodeResponse(t, rec, &response)
	if response.Code != "disabled1" {
		t.Fatalf("expected code disabled1, got %q", response.Code)
	}
	if response.TotalClicks != 7 {
		t.Fatalf("expected total clicks 7, got %d", response.TotalClicks)
	}
}

func TestHandlerGetAnalyticsInvalidFromDate(t *testing.T) {
	t.Parallel()

	handler := newGetAnalyticsValidationTestHandler(t)

	rec := executeGetAnalyticsRequest(t, handler, "abc12345", "from=2026-99-99")

	assertErrorResponse(t, rec, nethttp.StatusBadRequest, "invalid_argument")
}

func TestHandlerGetAnalyticsInvalidToDate(t *testing.T) {
	t.Parallel()

	handler := newGetAnalyticsValidationTestHandler(t)

	rec := executeGetAnalyticsRequest(t, handler, "abc12345", "to=bad-date")

	assertErrorResponse(t, rec, nethttp.StatusBadRequest, "invalid_argument")
}

func TestHandlerGetAnalyticsInvalidRecentLimit(t *testing.T) {
	t.Parallel()

	handler := newGetAnalyticsValidationTestHandler(t)

	rec := executeGetAnalyticsRequest(t, handler, "abc12345", "recent_limit=abc")

	assertErrorResponse(t, rec, nethttp.StatusBadRequest, "invalid_argument")
}

func TestHandlerGetAnalyticsTooLargeRecentLimit(t *testing.T) {
	t.Parallel()

	handler := newGetAnalyticsValidationTestHandler(t)

	rec := executeGetAnalyticsRequest(t, handler, "abc12345", "recent_limit=101")

	assertErrorResponse(t, rec, nethttp.StatusBadRequest, "invalid_argument")
}

func TestHandlerGetAnalyticsInternalAnalyticsError(t *testing.T) {
	t.Parallel()

	handler := NewHandlerWithDependencies(fakeLinksService{
		getLink: func(_ context.Context, code string) (links.Link, error) {
			return links.Link{
				ID:          uuid.New(),
				Code:        code,
				OriginalURL: "https://example.com",
			}, nil
		},
	}, nil, &fakeAnalyticsReader{
		getLinkAnalytics: func(_ context.Context, _ uuid.UUID, _ analytics.ClickFilter, _ int) (analytics.LinkAnalytics, error) {
			return analytics.LinkAnalytics{}, errors.New("analytics database unavailable")
		},
	}, "http://localhost:8080")

	rec := executeGetAnalyticsRequest(t, handler, "abc12345", "")

	assertErrorResponse(t, rec, nethttp.StatusInternalServerError, "internal_error")
}

func newGetAnalyticsValidationTestHandler(t *testing.T) *Handler {
	t.Helper()

	return NewHandlerWithDependencies(fakeLinksService{
		getLink: func(_ context.Context, _ string) (links.Link, error) {
			t.Fatal("GetLink must not be called for invalid query")
			return links.Link{}, nil
		},
	}, nil, &fakeAnalyticsReader{
		getLinkAnalytics: func(_ context.Context, _ uuid.UUID, _ analytics.ClickFilter, _ int) (analytics.LinkAnalytics, error) {
			t.Fatal("analytics reader must not be called for invalid query")
			return analytics.LinkAnalytics{}, nil
		},
	}, "http://localhost:8080")
}

func executeGetAnalyticsRequest(t *testing.T, handler *Handler, code string, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()

	target := "/api/v1/analytics/test-code"
	if rawQuery != "" {
		target += "?" + rawQuery
	}

	req := httptest.NewRequest(nethttp.MethodGet, target, nil)
	req.SetPathValue(linkCodePathValue, code)
	rec := httptest.NewRecorder()

	handler.GetAnalytics(rec, req)

	return rec
}
