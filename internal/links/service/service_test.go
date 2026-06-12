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
	createLink  func(ctx context.Context, link links.Link) (links.Link, error)
	getLink     func(ctx context.Context, code string) (links.Link, error)
	disableLink func(ctx context.Context, code string) (links.Link, error)
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

func (r fakeLinksRepository) DisableLink(ctx context.Context, code string) (links.Link, error) {
	if r.disableLink == nil {
		return links.Link{}, fmt.Errorf("disable link not implemented")
	}

	return r.disableLink(ctx, code)
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

type fakeLinkCache struct {
	getLink    func(ctx context.Context, code string) (links.Link, error)
	setLink    func(ctx context.Context, link links.Link) error
	deleteLink func(ctx context.Context, code string) error

	getCalls    int
	setCalls    int
	deleteCalls int
}

func (c *fakeLinkCache) GetLink(ctx context.Context, code string) (links.Link, error) {
	c.getCalls++
	if c.getLink == nil {
		return links.Link{}, fmt.Errorf("cache miss: %w", core_errors.ErrNotFound)
	}

	return c.getLink(ctx, code)
}

func (c *fakeLinkCache) SetLink(ctx context.Context, link links.Link) error {
	c.setCalls++
	if c.setLink == nil {
		return nil
	}

	return c.setLink(ctx, link)
}

func (c *fakeLinkCache) DeleteLink(ctx context.Context, code string) error {
	c.deleteCalls++
	if c.deleteLink == nil {
		return nil
	}

	return c.deleteLink(ctx, code)
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

func TestServiceResolveLinkCacheHit(t *testing.T) {
	t.Parallel()

	expectedLink := links.Link{
		ID:          uuid.New(),
		Code:        "cached1",
		OriginalURL: "https://example.com",
	}
	cache := &fakeLinkCache{
		getLink: func(_ context.Context, code string) (links.Link, error) {
			if code != expectedLink.Code {
				t.Fatalf("expected cache code %q, got %q", expectedLink.Code, code)
			}

			return expectedLink, nil
		},
	}
	repository := fakeLinksRepository{
		getLink: func(_ context.Context, _ string) (links.Link, error) {
			t.Fatal("repository must not be called on cache hit")
			return links.Link{}, nil
		},
	}
	service := NewServiceWithCache(repository, &fakeCodeGenerator{}, cache)

	link, err := service.ResolveLink(context.Background(), expectedLink.Code)
	if err != nil {
		t.Fatalf("resolve cached link: %v", err)
	}
	if link != expectedLink {
		t.Fatalf("expected cached link %+v, got %+v", expectedLink, link)
	}
	if cache.getCalls != 1 {
		t.Fatalf("expected one cache get call, got %d", cache.getCalls)
	}
	if cache.setCalls != 0 {
		t.Fatalf("expected no cache set calls, got %d", cache.setCalls)
	}
}

func TestServiceResolveLinkCacheMissCachesRepositoryLink(t *testing.T) {
	t.Parallel()

	expectedLink := links.Link{
		ID:          uuid.New(),
		Code:        "miss001",
		OriginalURL: "https://example.com",
	}
	cache := &fakeLinkCache{}
	repositoryCalls := 0
	repository := fakeLinksRepository{
		getLink: func(_ context.Context, code string) (links.Link, error) {
			repositoryCalls++
			if code != expectedLink.Code {
				t.Fatalf("expected repository code %q, got %q", expectedLink.Code, code)
			}

			return expectedLink, nil
		},
	}
	service := NewServiceWithCache(repository, &fakeCodeGenerator{}, cache)

	link, err := service.ResolveLink(context.Background(), expectedLink.Code)
	if err != nil {
		t.Fatalf("resolve link after cache miss: %v", err)
	}
	if link != expectedLink {
		t.Fatalf("expected repository link %+v, got %+v", expectedLink, link)
	}
	if repositoryCalls != 1 {
		t.Fatalf("expected one repository call, got %d", repositoryCalls)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected one cache set call, got %d", cache.setCalls)
	}
}

func TestServiceResolveLinkCacheGetErrorFallsBackToRepository(t *testing.T) {
	t.Parallel()

	expectedLink := links.Link{
		ID:          uuid.New(),
		Code:        "cacheerr",
		OriginalURL: "https://example.com",
	}
	cache := &fakeLinkCache{
		getLink: func(_ context.Context, _ string) (links.Link, error) {
			return links.Link{}, fmt.Errorf("redis unavailable")
		},
	}
	repositoryCalls := 0
	repository := fakeLinksRepository{
		getLink: func(_ context.Context, _ string) (links.Link, error) {
			repositoryCalls++
			return expectedLink, nil
		},
	}
	service := NewServiceWithCache(repository, &fakeCodeGenerator{}, cache)

	link, err := service.ResolveLink(context.Background(), expectedLink.Code)
	if err != nil {
		t.Fatalf("resolve link after cache get error: %v", err)
	}
	if link != expectedLink {
		t.Fatalf("expected repository link %+v, got %+v", expectedLink, link)
	}
	if repositoryCalls != 1 {
		t.Fatalf("expected one repository call, got %d", repositoryCalls)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected one cache set call, got %d", cache.setCalls)
	}
}

func TestServiceResolveLinkCacheSetErrorReturnsRepositoryLink(t *testing.T) {
	t.Parallel()

	expectedLink := links.Link{
		ID:          uuid.New(),
		Code:        "seterr1",
		OriginalURL: "https://example.com",
	}
	cache := &fakeLinkCache{
		setLink: func(_ context.Context, _ links.Link) error {
			return fmt.Errorf("redis set failed")
		},
	}
	repository := fakeLinksRepository{
		getLink: func(_ context.Context, _ string) (links.Link, error) {
			return expectedLink, nil
		},
	}
	service := NewServiceWithCache(repository, &fakeCodeGenerator{}, cache)

	link, err := service.ResolveLink(context.Background(), expectedLink.Code)
	if err != nil {
		t.Fatalf("resolve link after cache set error: %v", err)
	}
	if link != expectedLink {
		t.Fatalf("expected repository link %+v, got %+v", expectedLink, link)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected one cache set call, got %d", cache.setCalls)
	}
}

func TestServiceResolveLinkNotFoundDoesNotSetCache(t *testing.T) {
	t.Parallel()

	cache := &fakeLinkCache{}
	repository := fakeLinksRepository{
		getLink: func(_ context.Context, _ string) (links.Link, error) {
			return links.Link{}, fmt.Errorf("repository not found: %w", core_errors.ErrNotFound)
		},
	}
	service := NewServiceWithCache(repository, &fakeCodeGenerator{}, cache)

	_, err := service.ResolveLink(context.Background(), "missing1")
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if cache.setCalls != 0 {
		t.Fatalf("expected no cache set calls, got %d", cache.setCalls)
	}
}

func TestServiceResolveLinkDisabledDoesNotSetCache(t *testing.T) {
	t.Parallel()

	disabledAt := time.Now()
	cache := &fakeLinkCache{}
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
	service := NewServiceWithCache(repository, &fakeCodeGenerator{}, cache)

	_, err := service.ResolveLink(context.Background(), "disabled1")
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if cache.setCalls != 0 {
		t.Fatalf("expected no cache set calls, got %d", cache.setCalls)
	}
	if cache.deleteCalls != 1 {
		t.Fatalf("expected one cache delete call, got %d", cache.deleteCalls)
	}
}

func TestServiceResolveLinkCachedDisabledDeletesCache(t *testing.T) {
	t.Parallel()

	disabledAt := time.Now()
	cache := &fakeLinkCache{
		getLink: func(_ context.Context, code string) (links.Link, error) {
			return links.Link{
				ID:          uuid.New(),
				Code:        code,
				OriginalURL: "https://example.com",
				DisabledAt:  &disabledAt,
			}, nil
		},
	}
	repository := fakeLinksRepository{
		getLink: func(_ context.Context, _ string) (links.Link, error) {
			t.Fatal("repository must not be called for disabled cached link")
			return links.Link{}, nil
		},
	}
	service := NewServiceWithCache(repository, &fakeCodeGenerator{}, cache)

	_, err := service.ResolveLink(context.Background(), "disabled1")
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if cache.setCalls != 0 {
		t.Fatalf("expected no cache set calls, got %d", cache.setCalls)
	}
	if cache.deleteCalls != 1 {
		t.Fatalf("expected one cache delete call, got %d", cache.deleteCalls)
	}
}

func TestServiceGetLinkReturnsDisabledLink(t *testing.T) {
	t.Parallel()

	disabledAt := time.Now().UTC()
	expectedLink := links.Link{
		ID:          uuid.New(),
		Code:        "disabled1",
		OriginalURL: "https://example.com",
		DisabledAt:  &disabledAt,
	}
	cache := &fakeLinkCache{
		getLink: func(_ context.Context, _ string) (links.Link, error) {
			t.Fatal("cache must not be called when getting link metadata")
			return links.Link{}, nil
		},
	}
	repository := fakeLinksRepository{
		getLink: func(_ context.Context, code string) (links.Link, error) {
			if code != expectedLink.Code {
				t.Fatalf("expected code %q, got %q", expectedLink.Code, code)
			}

			return expectedLink, nil
		},
	}
	service := NewServiceWithCache(repository, &fakeCodeGenerator{}, cache)

	link, err := service.GetLink(context.Background(), expectedLink.Code)
	if err != nil {
		t.Fatalf("get link: %v", err)
	}
	if link != expectedLink {
		t.Fatalf("expected link %+v, got %+v", expectedLink, link)
	}
}

func TestServiceGetLinkNotFound(t *testing.T) {
	t.Parallel()

	repository := fakeLinksRepository{
		getLink: func(_ context.Context, _ string) (links.Link, error) {
			return links.Link{}, fmt.Errorf("repository not found: %w", core_errors.ErrNotFound)
		},
	}
	service := NewService(repository, &fakeCodeGenerator{})

	_, err := service.GetLink(context.Background(), "missing1")
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceDisableLinkSuccessDeletesCache(t *testing.T) {
	t.Parallel()

	disabledAt := time.Now().UTC()
	expectedLink := links.Link{
		ID:          uuid.New(),
		Code:        "disable1",
		OriginalURL: "https://example.com",
		DisabledAt:  &disabledAt,
	}
	cache := &fakeLinkCache{}
	repository := fakeLinksRepository{
		disableLink: func(_ context.Context, code string) (links.Link, error) {
			if code != expectedLink.Code {
				t.Fatalf("expected code %q, got %q", expectedLink.Code, code)
			}

			return expectedLink, nil
		},
	}
	service := NewServiceWithCache(repository, &fakeCodeGenerator{}, cache)

	link, err := service.DisableLink(context.Background(), expectedLink.Code)
	if err != nil {
		t.Fatalf("disable link: %v", err)
	}
	if link != expectedLink {
		t.Fatalf("expected link %+v, got %+v", expectedLink, link)
	}
	if cache.deleteCalls != 1 {
		t.Fatalf("expected one cache delete call, got %d", cache.deleteCalls)
	}
}

func TestServiceDisableLinkIgnoresCacheDeleteError(t *testing.T) {
	t.Parallel()

	cache := &fakeLinkCache{
		deleteLink: func(_ context.Context, _ string) error {
			return fmt.Errorf("redis delete failed")
		},
	}
	repository := fakeLinksRepository{
		disableLink: func(_ context.Context, code string) (links.Link, error) {
			disabledAt := time.Now().UTC()
			return links.Link{
				ID:          uuid.New(),
				Code:        code,
				OriginalURL: "https://example.com",
				DisabledAt:  &disabledAt,
			}, nil
		},
	}
	service := NewServiceWithCache(repository, &fakeCodeGenerator{}, cache)

	if _, err := service.DisableLink(context.Background(), "disable1"); err != nil {
		t.Fatalf("disable link must ignore cache delete error: %v", err)
	}
	if cache.deleteCalls != 1 {
		t.Fatalf("expected one cache delete call, got %d", cache.deleteCalls)
	}
}

func TestServiceDisableLinkNotFound(t *testing.T) {
	t.Parallel()

	cache := &fakeLinkCache{}
	repository := fakeLinksRepository{
		disableLink: func(_ context.Context, _ string) (links.Link, error) {
			return links.Link{}, fmt.Errorf("repository not found: %w", core_errors.ErrNotFound)
		},
	}
	service := NewServiceWithCache(repository, &fakeCodeGenerator{}, cache)

	_, err := service.DisableLink(context.Background(), "missing1")
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if cache.deleteCalls != 0 {
		t.Fatalf("expected no cache delete calls, got %d", cache.deleteCalls)
	}
}
