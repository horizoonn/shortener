//go:build integration

package redis

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/links"
	goredis "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestCacheSetGetDeleteLink(t *testing.T) {
	ctx := context.Background()
	client := startRedisClient(t, ctx)
	cache, err := NewCache(client, time.Minute, 10*time.Second)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}
	t.Cleanup(func() {
		_ = cache.Close()
	})

	link := links.Link{
		ID:          uuid.New(),
		Code:        "abc12345",
		OriginalURL: "https://example.com/path",
	}
	if err := cache.SetLink(ctx, link); err != nil {
		t.Fatalf("set link: %v", err)
	}

	ttl, err := client.TTL(ctx, linkKey(link.Code)).Result()
	if err != nil {
		t.Fatalf("read ttl: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("expected positive ttl, got %s", ttl)
	}

	got, err := cache.GetLink(ctx, link.Code)
	if err != nil {
		t.Fatalf("get link: %v", err)
	}
	if got.ID != link.ID || got.Code != link.Code || got.OriginalURL != link.OriginalURL {
		t.Fatalf("unexpected cached link: %+v", got)
	}

	if err := cache.DeleteLink(ctx, link.Code); err != nil {
		t.Fatalf("delete link: %v", err)
	}
	if _, err := cache.GetLink(ctx, link.Code); err == nil {
		t.Fatal("expected cache miss after delete")
	}
}

func TestCachePreservesDisabledAt(t *testing.T) {
	ctx := context.Background()
	client := startRedisClient(t, ctx)
	cache, err := NewCache(client, time.Minute, 10*time.Second)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}
	t.Cleanup(func() {
		_ = cache.Close()
	})

	disabledAt := time.Now().UTC().Round(time.Nanosecond)
	link := links.Link{
		ID:          uuid.New(),
		Code:        "disabled1",
		OriginalURL: "https://example.com/path",
		DisabledAt:  &disabledAt,
	}
	if err := cache.SetLink(ctx, link); err != nil {
		t.Fatalf("set link: %v", err)
	}

	got, err := cache.GetLink(ctx, link.Code)
	if err != nil {
		t.Fatalf("get link: %v", err)
	}
	if got.DisabledAt == nil || !got.DisabledAt.Equal(disabledAt) {
		t.Fatalf("expected disabled_at %s, got %v", disabledAt, got.DisabledAt)
	}
}

func TestCacheSetLinkNotFound(t *testing.T) {
	ctx := context.Background()
	client := startRedisClient(t, ctx)
	cache, err := NewCache(client, time.Minute, 10*time.Second)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}
	t.Cleanup(func() {
		_ = cache.Close()
	})

	code := "missing1"
	if err := cache.SetLinkNotFound(ctx, code); err != nil {
		t.Fatalf("set link not found: %v", err)
	}

	ttl, err := client.TTL(ctx, linkMissKey(code)).Result()
	if err != nil {
		t.Fatalf("read miss ttl: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("expected positive miss ttl, got %s", ttl)
	}

	if _, err := cache.GetLink(ctx, code); err == nil {
		t.Fatal("expected negative cache hit")
	}
}

func TestCacheSetAndDeleteLinkClearNegativeCache(t *testing.T) {
	ctx := context.Background()
	client := startRedisClient(t, ctx)
	cache, err := NewCache(client, time.Minute, 10*time.Second)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}
	t.Cleanup(func() {
		_ = cache.Close()
	})

	link := links.Link{
		ID:          uuid.New(),
		Code:        "clearneg",
		OriginalURL: "https://example.com/path",
	}
	if err := cache.SetLinkNotFound(ctx, link.Code); err != nil {
		t.Fatalf("set link not found: %v", err)
	}
	if err := cache.SetLink(ctx, link); err != nil {
		t.Fatalf("set link: %v", err)
	}
	if exists, err := client.Exists(ctx, linkMissKey(link.Code)).Result(); err != nil {
		t.Fatalf("check miss key: %v", err)
	} else if exists != 0 {
		t.Fatalf("expected miss key to be deleted, exists=%d", exists)
	}

	if err := cache.SetLinkNotFound(ctx, link.Code); err != nil {
		t.Fatalf("set link not found again: %v", err)
	}
	if err := cache.DeleteLink(ctx, link.Code); err != nil {
		t.Fatalf("delete link: %v", err)
	}
	if exists, err := client.Exists(ctx, linkKey(link.Code), linkMissKey(link.Code)).Result(); err != nil {
		t.Fatalf("check deleted keys: %v", err)
	} else if exists != 0 {
		t.Fatalf("expected both cache keys to be deleted, exists=%d", exists)
	}
}

func TestCacheDeletesCorruptLinkPayload(t *testing.T) {
	ctx := context.Background()
	client := startRedisClient(t, ctx)
	cache, err := NewCache(client, time.Minute, 10*time.Second)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}
	t.Cleanup(func() {
		_ = cache.Close()
	})

	code := "badjson1"
	if err := client.Set(ctx, linkKey(code), "{", time.Minute).Err(); err != nil {
		t.Fatalf("seed corrupt cache value: %v", err)
	}

	if _, err := cache.GetLink(ctx, code); err == nil {
		t.Fatal("expected corrupt cache value error")
	}

	exists, err := client.Exists(ctx, linkKey(code)).Result()
	if err != nil {
		t.Fatalf("check corrupt cache value deletion: %v", err)
	}
	if exists != 0 {
		t.Fatalf("expected corrupt cache value to be deleted, exists=%d", exists)
	}
}

func startRedisClient(t *testing.T, ctx context.Context) *goredis.Client {
	t.Helper()

	container, err := tcredis.Run(ctx, "redis:8-alpine")
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Fatalf("terminate redis container: %v", err)
		}
	})

	connectionString, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("get redis connection string: %v", err)
	}

	redisURL, err := url.Parse(connectionString)
	if err != nil {
		t.Fatalf("parse redis connection string: %v", err)
	}
	password, _ := redisURL.User.Password()

	client := goredis.NewClient(&goredis.Options{
		Addr:     net.JoinHostPort(redisURL.Hostname(), redisURL.Port()),
		Password: password,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("ping redis container: %v", err)
	}

	return client
}
