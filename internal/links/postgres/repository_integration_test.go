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
	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
	linkspg "github.com/horizoonn/shortener/internal/links/postgres"
	testpostgres "github.com/horizoonn/shortener/internal/testsupport/postgres"
)

var linksTestDB *testpostgres.Database

func TestMain(m *testing.M) {
	ctx := context.Background()

	db, err := testpostgres.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres test database: %v\n", err)
		os.Exit(1)
	}
	linksTestDB = db

	code := m.Run()

	if err := db.Close(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "close postgres test database: %v\n", err)
		code = 1
	}

	os.Exit(code)
}

func TestRepositoryCreateLinkGeneratedSuccess(t *testing.T) {
	cleanLinksDB(t)

	repository := linkspg.NewRepository(linksTestDB.Pool)
	link := links.Link{
		ID:          uuid.New(),
		Code:        "abc12345",
		OriginalURL: "https://example.com/generated",
		IsCustom:    false,
	}

	created, err := repository.CreateLink(context.Background(), link)
	if err != nil {
		t.Fatalf("create generated link: %v", err)
	}

	if created.ID != link.ID {
		t.Fatalf("expected id %s, got %s", link.ID, created.ID)
	}
	if created.Code != link.Code {
		t.Fatalf("expected code %q, got %q", link.Code, created.Code)
	}
	if created.OriginalURL != link.OriginalURL {
		t.Fatalf("expected original URL %q, got %q", link.OriginalURL, created.OriginalURL)
	}
	if created.IsCustom {
		t.Fatal("expected generated link to not be custom")
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("expected created_at to be filled by database")
	}
	if created.DisabledAt != nil {
		t.Fatalf("expected nil disabled_at, got %v", created.DisabledAt)
	}
}

func TestRepositoryCreateLinkCustomAliasSuccess(t *testing.T) {
	cleanLinksDB(t)

	repository := linkspg.NewRepository(linksTestDB.Pool)
	link := links.Link{
		ID:          uuid.New(),
		Code:        "custom_alias",
		OriginalURL: "https://example.com/custom",
		IsCustom:    true,
	}

	created, err := repository.CreateLink(context.Background(), link)
	if err != nil {
		t.Fatalf("create custom link: %v", err)
	}

	if created.Code != link.Code {
		t.Fatalf("expected code %q, got %q", link.Code, created.Code)
	}
	if !created.IsCustom {
		t.Fatal("expected custom link")
	}
}

func TestRepositoryCreateLinkDuplicateCodeConflict(t *testing.T) {
	cleanLinksDB(t)

	repository := linkspg.NewRepository(linksTestDB.Pool)
	link := links.Link{
		ID:          uuid.New(),
		Code:        "duplicate1",
		OriginalURL: "https://example.com/one",
	}

	if _, err := repository.CreateLink(context.Background(), link); err != nil {
		t.Fatalf("create first link: %v", err)
	}

	duplicate := links.Link{
		ID:          uuid.New(),
		Code:        link.Code,
		OriginalURL: "https://example.com/two",
	}
	_, err := repository.CreateLink(context.Background(), duplicate)
	if !errors.Is(err, core_errors.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestRepositoryGetLinkByCode(t *testing.T) {
	cleanLinksDB(t)

	repository := linkspg.NewRepository(linksTestDB.Pool)
	created := createTestLink(t, repository, "bycode01")

	got, err := repository.GetLinkByCode(context.Background(), created.Code)
	if err != nil {
		t.Fatalf("get link by code: %v", err)
	}

	assertLinksEqual(t, created, got)
}

func TestRepositoryGetLinkByCodeNotFound(t *testing.T) {
	cleanLinksDB(t)

	repository := linkspg.NewRepository(linksTestDB.Pool)

	_, err := repository.GetLinkByCode(context.Background(), "missing1")
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestRepositoryGetLinkByID(t *testing.T) {
	cleanLinksDB(t)

	repository := linkspg.NewRepository(linksTestDB.Pool)
	created := createTestLink(t, repository, "byid001")

	got, err := repository.GetLinkByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get link by id: %v", err)
	}

	assertLinksEqual(t, created, got)
}

func TestRepositoryGetLinkByIDNotFound(t *testing.T) {
	cleanLinksDB(t)

	repository := linkspg.NewRepository(linksTestDB.Pool)

	_, err := repository.GetLinkByID(context.Background(), uuid.New())
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestRepositoryReadsDisabledAt(t *testing.T) {
	cleanLinksDB(t)

	repository := linkspg.NewRepository(linksTestDB.Pool)
	disabledAt := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	link := links.Link{
		ID:          uuid.New(),
		Code:        "disabled",
		OriginalURL: "https://example.com/disabled",
		DisabledAt:  &disabledAt,
	}

	created, err := repository.CreateLink(context.Background(), link)
	if err != nil {
		t.Fatalf("create disabled link: %v", err)
	}
	if created.DisabledAt == nil {
		t.Fatal("expected disabled_at to be returned")
	}
	if !created.DisabledAt.Equal(disabledAt) {
		t.Fatalf("expected disabled_at %s, got %s", disabledAt, *created.DisabledAt)
	}

	got, err := repository.GetLinkByCode(context.Background(), link.Code)
	if err != nil {
		t.Fatalf("get disabled link by code: %v", err)
	}
	if got.DisabledAt == nil || !got.DisabledAt.Equal(disabledAt) {
		t.Fatalf("expected disabled_at %s, got %v", disabledAt, got.DisabledAt)
	}
}

func createTestLink(t *testing.T, repository *linkspg.Repository, code string) links.Link {
	t.Helper()

	link := links.Link{
		ID:          uuid.New(),
		Code:        code,
		OriginalURL: "https://example.com/" + code,
	}

	created, err := repository.CreateLink(context.Background(), link)
	if err != nil {
		t.Fatalf("create test link: %v", err)
	}

	return created
}

func cleanLinksDB(t *testing.T) {
	t.Helper()

	if err := linksTestDB.Clean(context.Background()); err != nil {
		t.Fatalf("clean postgres database: %v", err)
	}
}

func assertLinksEqual(t *testing.T, want links.Link, got links.Link) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("expected id %s, got %s", want.ID, got.ID)
	}
	if got.Code != want.Code {
		t.Fatalf("expected code %q, got %q", want.Code, got.Code)
	}
	if got.OriginalURL != want.OriginalURL {
		t.Fatalf("expected original URL %q, got %q", want.OriginalURL, got.OriginalURL)
	}
	if got.IsCustom != want.IsCustom {
		t.Fatalf("expected is_custom %v, got %v", want.IsCustom, got.IsCustom)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("expected created_at %s, got %s", want.CreatedAt, got.CreatedAt)
	}
	if (got.DisabledAt == nil) != (want.DisabledAt == nil) {
		t.Fatalf("expected disabled_at %v, got %v", want.DisabledAt, got.DisabledAt)
	}
	if got.DisabledAt != nil && !got.DisabledAt.Equal(*want.DisabledAt) {
		t.Fatalf("expected disabled_at %s, got %s", *want.DisabledAt, *got.DisabledAt)
	}
}
