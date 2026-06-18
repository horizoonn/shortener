package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	core_errors "github.com/horizoonn/shortener/internal/errors"
)

type fakeClicksRepository struct {
	saveClick func(ctx context.Context, click analytics.Click) (analytics.Click, error)
}

func (r fakeClicksRepository) SaveClick(ctx context.Context, click analytics.Click) (analytics.Click, error) {
	if r.saveClick == nil {
		return analytics.Click{}, fmt.Errorf("save click not implemented")
	}

	return r.saveClick(ctx, click)
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
