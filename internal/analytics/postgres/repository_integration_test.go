//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	analyticspg "github.com/horizoonn/shortener/internal/analytics/postgres"
	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
	linkspg "github.com/horizoonn/shortener/internal/links/postgres"
	testpostgres "github.com/horizoonn/shortener/internal/testsupport/postgres"
)

var analyticsTestDB *testpostgres.Database

func TestMain(m *testing.M) {
	ctx := context.Background()

	db, err := testpostgres.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres test database: %v\n", err)
		os.Exit(1)
	}
	analyticsTestDB = db

	code := m.Run()

	if err := db.Close(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "close postgres test database: %v\n", err)
		code = 1
	}

	os.Exit(code)
}

func TestRepositorySaveClickSuccess(t *testing.T) {
	cleanAnalyticsDB(t)

	linkRepository := linkspg.NewRepository(analyticsTestDB.Pool)
	clickRepository := analyticspg.NewRepository(analyticsTestDB.Pool)
	link := createAnalyticsTestLink(t, linkRepository, "saveclk1")
	referer := "https://referer.example"
	ip := "127.0.0.1"

	click, err := clickRepository.SaveClick(context.Background(), analytics.Click{
		ID:        uuid.New(),
		LinkID:    link.ID,
		UserAgent: "Mozilla/5.0",
		Referer:   &referer,
		IP:        &ip,
	})
	if err != nil {
		t.Fatalf("save click: %v", err)
	}

	if click.LinkID != link.ID {
		t.Fatalf("expected link ID %s, got %s", link.ID, click.LinkID)
	}
	if click.ClickedAt.IsZero() {
		t.Fatal("expected clicked_at to be filled by database")
	}
	if click.UserAgent != "Mozilla/5.0" {
		t.Fatalf("expected user agent Mozilla/5.0, got %q", click.UserAgent)
	}
	if click.Referer == nil || *click.Referer != referer {
		t.Fatalf("expected referer %q, got %v", referer, click.Referer)
	}
	if click.IP == nil || *click.IP != ip {
		t.Fatalf("expected IP %q, got %v", ip, click.IP)
	}
}

func TestRepositoryClickAggregations(t *testing.T) {
	cleanAnalyticsDB(t)

	linkRepository := linkspg.NewRepository(analyticsTestDB.Pool)
	clickRepository := analyticspg.NewRepository(analyticsTestDB.Pool)
	link := createAnalyticsTestLink(t, linkRepository, "aggr001")

	insertClickAt(t, link.ID, time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), "Mozilla/5.0", nil, nil)
	insertClickAt(t, link.ID, time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC), "Mozilla/5.0", nil, nil)
	insertClickAt(t, link.ID, time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC), "curl/8.0", nil, nil)
	insertClickAt(t, link.ID, time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC), "curl/8.0", nil, nil)

	filter := analytics.ClickFilter{}
	total, err := clickRepository.CountClicks(context.Background(), link.ID, filter)
	if err != nil {
		t.Fatalf("count clicks: %v", err)
	}
	if total != 4 {
		t.Fatalf("expected total 4, got %d", total)
	}

	byDay, err := clickRepository.CountClicksByDay(context.Background(), link.ID, filter)
	if err != nil {
		t.Fatalf("count clicks by day: %v", err)
	}
	assertTimeBucketCounts(t, byDay, "2006-01-02", map[string]int64{
		"2026-06-01": 2,
		"2026-06-02": 1,
		"2026-07-01": 1,
	})

	byMonth, err := clickRepository.CountClicksByMonth(context.Background(), link.ID, filter)
	if err != nil {
		t.Fatalf("count clicks by month: %v", err)
	}
	assertTimeBucketCounts(t, byMonth, "2006-01", map[string]int64{
		"2026-06": 3,
		"2026-07": 1,
	})

	byUserAgent, err := clickRepository.CountClicksByUserAgent(context.Background(), link.ID, filter)
	if err != nil {
		t.Fatalf("count clicks by user agent: %v", err)
	}
	assertUserAgentCounts(t, byUserAgent, map[string]int64{
		"Mozilla/5.0": 2,
		"curl/8.0":    2,
	})
}

func TestRepositoryRecentClicks(t *testing.T) {
	cleanAnalyticsDB(t)

	linkRepository := linkspg.NewRepository(analyticsTestDB.Pool)
	clickRepository := analyticspg.NewRepository(analyticsTestDB.Pool)
	link := createAnalyticsTestLink(t, linkRepository, "recent1")
	referer := "https://example.org"
	ip := "192.0.2.10"

	insertClickAt(t, link.ID, time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), "old-agent", nil, nil)
	insertClickAt(t, link.ID, time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), "mid-agent", nil, nil)
	insertClickAt(t, link.ID, time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC), "new-agent", &referer, &ip)

	recentClicks, err := clickRepository.RecentClicks(context.Background(), link.ID, 2)
	if err != nil {
		t.Fatalf("recent clicks: %v", err)
	}

	if len(recentClicks) != 2 {
		t.Fatalf("expected 2 recent clicks, got %d", len(recentClicks))
	}
	if recentClicks[0].UserAgent != "new-agent" {
		t.Fatalf("expected latest click first, got %+v", recentClicks[0])
	}
	if recentClicks[1].UserAgent != "mid-agent" {
		t.Fatalf("expected second latest click second, got %+v", recentClicks[1])
	}
	if recentClicks[0].Referer == nil || *recentClicks[0].Referer != referer {
		t.Fatalf("expected referer %q, got %v", referer, recentClicks[0].Referer)
	}
	if recentClicks[0].IP == nil || *recentClicks[0].IP != ip {
		t.Fatalf("expected IP %q, got %v", ip, recentClicks[0].IP)
	}
}

func TestRepositoryDateFilters(t *testing.T) {
	cleanAnalyticsDB(t)

	linkRepository := linkspg.NewRepository(analyticsTestDB.Pool)
	clickRepository := analyticspg.NewRepository(analyticsTestDB.Pool)
	link := createAnalyticsTestLink(t, linkRepository, "filter1")

	insertClickAt(t, link.ID, time.Date(2026, 5, 31, 23, 59, 59, 0, time.UTC), "before", nil, nil)
	insertClickAt(t, link.ID, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), "at-from", nil, nil)
	insertClickAt(t, link.ID, time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC), "before-to", nil, nil)
	insertClickAt(t, link.ID, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), "at-to", nil, nil)

	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	filter := analytics.ClickFilter{From: &from, To: &to}

	total, err := clickRepository.CountClicks(context.Background(), link.ID, filter)
	if err != nil {
		t.Fatalf("count filtered clicks: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 filtered clicks, got %d", total)
	}

	byDay, err := clickRepository.CountClicksByDay(context.Background(), link.ID, filter)
	if err != nil {
		t.Fatalf("count filtered clicks by day: %v", err)
	}
	assertTimeBucketCounts(t, byDay, "2006-01-02", map[string]int64{
		"2026-06-01": 1,
		"2026-06-30": 1,
	})
}

func TestRepositoryRecentClicksRejectsInvalidLimit(t *testing.T) {
	cleanAnalyticsDB(t)

	clickRepository := analyticspg.NewRepository(analyticsTestDB.Pool)

	_, err := clickRepository.RecentClicks(context.Background(), uuid.New(), 0)
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func createAnalyticsTestLink(t *testing.T, repository *linkspg.Repository, code string) links.Link {
	t.Helper()

	created, err := repository.CreateLink(context.Background(), links.Link{
		ID:          uuid.New(),
		Code:        code,
		OriginalURL: "https://example.com/" + code,
	})
	if err != nil {
		t.Fatalf("create analytics test link: %v", err)
	}

	return created
}

func insertClickAt(
	t *testing.T,
	linkID uuid.UUID,
	clickedAt time.Time,
	userAgent string,
	referer *string,
	ip *string,
) {
	t.Helper()

	_, err := analyticsTestDB.Pool.Exec(
		context.Background(),
		`
		INSERT INTO clicks (id, link_id, clicked_at, user_agent, referer, ip)
		VALUES ($1, $2, $3, $4, $5, CAST($6 AS inet));
		`,
		uuid.New(),
		linkID,
		clickedAt,
		userAgent,
		referer,
		ip,
	)
	if err != nil {
		t.Fatalf("insert click at %s: %v", clickedAt, err)
	}
}

func assertTimeBucketCounts(t *testing.T, got []analytics.TimeBucketCount, layout string, want map[string]int64) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d bucket counts, got %d: %+v", len(want), len(got), got)
	}

	for _, count := range got {
		key := count.Bucket.Format(layout)
		if want[key] != count.Count {
			t.Fatalf("expected bucket %s count %d, got %d", key, want[key], count.Count)
		}
	}
}

func assertUserAgentCounts(t *testing.T, got []analytics.UserAgentCount, want map[string]int64) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d user-agent counts, got %d: %+v", len(want), len(got), got)
	}

	for _, count := range got {
		if want[count.UserAgent] != count.Count {
			t.Fatalf("expected user agent %q count %d, got %d", count.UserAgent, want[count.UserAgent], count.Count)
		}
	}
}

func cleanAnalyticsDB(t *testing.T) {
	t.Helper()

	if err := analyticsTestDB.Clean(context.Background()); err != nil {
		t.Fatalf("clean postgres database: %v", err)
	}
}
