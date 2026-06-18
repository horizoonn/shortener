package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
)

type fakeLinksRepository struct {
	createLink func(ctx context.Context, link links.Link) (links.Link, error)
	getLink    func(ctx context.Context, code string) (links.Link, error)
}

func (r fakeLinksRepository) CreateLink(ctx context.Context, link links.Link) (links.Link, error) {
	if r.createLink == nil {
		return links.Link{}, fmt.Errorf("create link not implemented")
	}

	return r.createLink(ctx, link)
}

func (r fakeLinksRepository) GetLinkByCode(ctx context.Context, code string) (links.Link, error) {
	if r.getLink == nil {
		return links.Link{}, fmt.Errorf("get link not implemented")
	}

	return r.getLink(ctx, code)
}

type fakeCodeGenerator struct {
	codes []string
	err   error
	calls int
}

func (g *fakeCodeGenerator) Generate() (string, error) {
	g.calls++
	if g.err != nil {
		return "", g.err
	}
	if len(g.codes) == 0 {
		return "", fmt.Errorf("no generated codes")
	}

	code := g.codes[0]
	g.codes = g.codes[1:]
	return code, nil
}

func TestServiceCreateLinkGeneratedSuccess(t *testing.T) {
	t.Parallel()

	generator := &fakeCodeGenerator{codes: []string{"abc1234"}}
	repository := fakeLinksRepository{
		createLink: func(_ context.Context, link links.Link) (links.Link, error) {
			if link.ID == uuid.Nil {
				t.Fatal("expected generated link ID")
			}
			if link.Code != "abc1234" {
				t.Fatalf("expected code abc1234, got %q", link.Code)
			}
			if link.OriginalURL != "https://example.com" {
				t.Fatalf("expected original URL to be preserved, got %q", link.OriginalURL)
			}
			if link.IsCustom {
				t.Fatal("expected generated link to not be custom")
			}
			return link, nil
		},
	}

	service := NewService(repository, generator)

	link, err := service.CreateLink(context.Background(), "https://example.com", nil)
	if err != nil {
		t.Fatalf("create link: %v", err)
	}
	if link.Code != "abc1234" {
		t.Fatalf("expected returned code abc1234, got %q", link.Code)
	}
	if generator.calls != 1 {
		t.Fatalf("expected one generator call, got %d", generator.calls)
	}
}

func TestServiceCreateLinkCustomAliasSuccess(t *testing.T) {
	t.Parallel()

	customAlias := "custom_alias-123"
	generator := &fakeCodeGenerator{codes: []string{"unused"}}
	repository := fakeLinksRepository{
		createLink: func(_ context.Context, link links.Link) (links.Link, error) {
			if link.Code != customAlias {
				t.Fatalf("expected custom alias code %q, got %q", customAlias, link.Code)
			}
			if !link.IsCustom {
				t.Fatal("expected custom link")
			}
			return link, nil
		},
	}

	service := NewService(repository, generator)

	link, err := service.CreateLink(context.Background(), "https://example.com", &customAlias)
	if err != nil {
		t.Fatalf("create custom link: %v", err)
	}
	if link.Code != customAlias {
		t.Fatalf("expected returned custom alias %q, got %q", customAlias, link.Code)
	}
	if generator.calls != 0 {
		t.Fatalf("expected generator not to be called, got %d calls", generator.calls)
	}
}

func TestServiceCreateLinkInvalidOriginalURL(t *testing.T) {
	t.Parallel()

	service := NewService(fakeLinksRepository{}, &fakeCodeGenerator{})

	_, err := service.CreateLink(context.Background(), "ftp://example.com", nil)
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestServiceCreateLinkInvalidCustomAlias(t *testing.T) {
	t.Parallel()

	customAlias := "bad alias"
	service := NewService(fakeLinksRepository{}, &fakeCodeGenerator{})

	_, err := service.CreateLink(context.Background(), "https://example.com", &customAlias)
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestServiceCreateLinkCustomAliasConflict(t *testing.T) {
	t.Parallel()

	customAlias := "taken-alias"
	repository := fakeLinksRepository{
		createLink: func(_ context.Context, _ links.Link) (links.Link, error) {
			return links.Link{}, fmt.Errorf("repository conflict: %w", core_errors.ErrConflict)
		},
	}
	service := NewService(repository, &fakeCodeGenerator{})

	_, err := service.CreateLink(context.Background(), "https://example.com", &customAlias)
	if !errors.Is(err, core_errors.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestServiceCreateLinkGeneratedCollisionThenSuccess(t *testing.T) {
	t.Parallel()

	generator := &fakeCodeGenerator{codes: []string{"collide1", "success1"}}
	attempts := 0
	repository := fakeLinksRepository{
		createLink: func(_ context.Context, link links.Link) (links.Link, error) {
			attempts++
			if attempts == 1 {
				return links.Link{}, fmt.Errorf("repository conflict: %w", core_errors.ErrConflict)
			}

			return link, nil
		},
	}
	service := NewService(repository, generator)

	link, err := service.CreateLink(context.Background(), "https://example.com", nil)
	if err != nil {
		t.Fatalf("create generated link after collision: %v", err)
	}
	if link.Code != "success1" {
		t.Fatalf("expected success code, got %q", link.Code)
	}
	if generator.calls != 2 {
		t.Fatalf("expected two generator calls, got %d", generator.calls)
	}
}

func TestServiceCreateLinkGeneratedCollisionMaxAttemptsExceeded(t *testing.T) {
	t.Parallel()

	generator := &fakeCodeGenerator{codes: []string{"code001", "code002", "code003", "code004", "code005"}}
	repository := fakeLinksRepository{
		createLink: func(_ context.Context, _ links.Link) (links.Link, error) {
			return links.Link{}, fmt.Errorf("repository conflict: %w", core_errors.ErrConflict)
		},
	}
	service := NewService(repository, generator)

	_, err := service.CreateLink(context.Background(), "https://example.com", nil)
	if !errors.Is(err, core_errors.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if generator.calls != MaxCodeGenerationAttempts {
		t.Fatalf("expected %d generator calls, got %d", MaxCodeGenerationAttempts, generator.calls)
	}
}

func TestServiceResolveLinkSuccess(t *testing.T) {
	t.Parallel()

	expectedLink := links.Link{
		ID:          uuid.New(),
		Code:        "abc1234",
		OriginalURL: "https://example.com",
	}
	repository := fakeLinksRepository{
		getLink: func(_ context.Context, code string) (links.Link, error) {
			if code != expectedLink.Code {
				t.Fatalf("expected code %q, got %q", expectedLink.Code, code)
			}
			return expectedLink, nil
		},
	}
	service := NewService(repository, &fakeCodeGenerator{})

	link, err := service.ResolveLink(context.Background(), expectedLink.Code)
	if err != nil {
		t.Fatalf("resolve link: %v", err)
	}
	if link != expectedLink {
		t.Fatalf("expected link %+v, got %+v", expectedLink, link)
	}
}

func TestServiceResolveLinkNotFound(t *testing.T) {
	t.Parallel()

	repository := fakeLinksRepository{
		getLink: func(_ context.Context, _ string) (links.Link, error) {
			return links.Link{}, fmt.Errorf("repository not found: %w", core_errors.ErrNotFound)
		},
	}
	service := NewService(repository, &fakeCodeGenerator{})

	_, err := service.ResolveLink(context.Background(), "missing1")
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceResolveLinkDisabled(t *testing.T) {
	t.Parallel()

	disabledAt := time.Now()
	repository := fakeLinksRepository{
		getLink: func(_ context.Context, code string) (links.Link, error) {
			return links.Link{
				ID:          uuid.New(),
				Code:        code,
				OriginalURL: "https://example.com",
				DisabledAt:  &disabledAt,
			}, nil
		},
	}
	service := NewService(repository, &fakeCodeGenerator{})

	_, err := service.ResolveLink(context.Background(), "disabled1")
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
