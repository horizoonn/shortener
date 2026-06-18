package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	core_errors "github.com/horizoonn/shortener/internal/errors"
	"github.com/horizoonn/shortener/internal/links"
	goredis "github.com/redis/go-redis/v9"
)

const linkKeyPrefix = "shortener:link:"
const linkMissKeyPrefix = "shortener:link-miss:"

type Metrics interface {
	RecordCacheHit(cache string)
	RecordCacheMiss(cache string)
}

type Cache struct {
	client  *goredis.Client
	ttl     time.Duration
	missTTL time.Duration
	metrics Metrics
}

type cachedLink struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	OriginalURL string `json:"original_url"`
	DisabledAt  string `json:"disabled_at,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

func NewCache(client *goredis.Client, ttl time.Duration, missTTL time.Duration) (*Cache, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is required")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("redis cache TTL must be positive")
	}
	if missTTL <= 0 {
		return nil, fmt.Errorf("redis miss TTL must be positive")
	}

	return &Cache{
		client:  client,
		ttl:     ttl,
		missTTL: missTTL,
	}, nil
}

func (c *Cache) WithMetrics(metrics Metrics) *Cache {
	if c == nil {
		return nil
	}
	c.metrics = metrics
	return c
}

func (c *Cache) GetLink(ctx context.Context, code string) (links.Link, error) {
	data, err := c.client.Get(ctx, linkKey(code)).Bytes()
	if err != nil {
		if !errors.Is(err, goredis.Nil) {
			return links.Link{}, fmt.Errorf("get link from redis: %w", err)
		}

		missExists, missErr := c.client.Exists(ctx, linkMissKey(code)).Result()
		if missErr != nil {
			return links.Link{}, fmt.Errorf("get link miss from redis: %w", missErr)
		}
		if missExists > 0 {
			if c.metrics != nil {
				c.metrics.RecordCacheHit("links")
			}
			return links.Link{}, fmt.Errorf("negative link cache hit: %w", core_errors.ErrNotFound)
		}

		if c.metrics != nil {
			c.metrics.RecordCacheMiss("links")
		}
		return links.Link{}, fmt.Errorf("link cache miss: %w", core_errors.ErrCacheMiss)
	}

	if c.metrics != nil {
		c.metrics.RecordCacheHit("links")
	}

	var cached cachedLink
	if err := json.Unmarshal(data, &cached); err != nil {
		_ = c.client.Del(ctx, linkKey(code)).Err()
		return links.Link{}, fmt.Errorf("decode cached link: %w", err)
	}
	if cached.Code != code {
		_ = c.client.Del(ctx, linkKey(code)).Err()
		return links.Link{}, fmt.Errorf("cached link code mismatch: %w", core_errors.ErrNotFound)
	}

	link, err := cached.toLink()
	if err != nil {
		_ = c.client.Del(ctx, linkKey(code)).Err()
		return links.Link{}, fmt.Errorf("convert cached link: %w", err)
	}

	return link, nil
}

func (c *Cache) SetLink(ctx context.Context, link links.Link) error {
	data, err := json.Marshal(newCachedLink(link))
	if err != nil {
		return fmt.Errorf("encode cached link: %w", err)
	}

	ttl := c.ttl
	if link.ExpiresAt != nil {
		remaining := time.Until(*link.ExpiresAt)
		if remaining <= 0 {
			_ = c.client.Del(ctx, linkMissKey(link.Code)).Err()
			return nil
		}
		if remaining < ttl {
			ttl = remaining
		}
	}

	if err := c.client.Set(ctx, linkKey(link.Code), data, ttl).Err(); err != nil {
		return fmt.Errorf("set link in redis: %w", err)
	}
	_ = c.client.Del(ctx, linkMissKey(link.Code)).Err()

	return nil
}

func (c *Cache) SetLinkNotFound(ctx context.Context, code string) error {
	if err := c.client.Set(ctx, linkMissKey(code), "1", c.missTTL).Err(); err != nil {
		return fmt.Errorf("set link miss in redis: %w", err)
	}

	return nil
}

func (c *Cache) DeleteLink(ctx context.Context, code string) error {
	if err := c.client.Del(ctx, linkKey(code), linkMissKey(code)).Err(); err != nil {
		return fmt.Errorf("delete link from redis: %w", err)
	}

	return nil
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func newCachedLink(link links.Link) cachedLink {
	var disabledAt string
	if link.DisabledAt != nil {
		disabledAt = link.DisabledAt.UTC().Format(time.RFC3339Nano)
	}

	var expiresAt string
	if link.ExpiresAt != nil {
		expiresAt = link.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}

	return cachedLink{
		ID:          link.ID.String(),
		Code:        link.Code,
		OriginalURL: link.OriginalURL,
		DisabledAt:  disabledAt,
		ExpiresAt:   expiresAt,
	}
}

func (l cachedLink) toLink() (links.Link, error) {
	id, err := uuid.Parse(l.ID)
	if err != nil {
		return links.Link{}, fmt.Errorf("parse link ID: %w", err)
	}

	var disabledAt *time.Time
	if l.DisabledAt != "" {
		parsedDisabledAt, err := time.Parse(time.RFC3339Nano, l.DisabledAt)
		if err != nil {
			return links.Link{}, fmt.Errorf("parse disabled_at: %w", err)
		}
		disabledAt = &parsedDisabledAt
	}

	var expiresAt *time.Time
	if l.ExpiresAt != "" {
		parsedExpiresAt, err := time.Parse(time.RFC3339Nano, l.ExpiresAt)
		if err != nil {
			return links.Link{}, fmt.Errorf("parse expires_at: %w", err)
		}
		expiresAt = &parsedExpiresAt
	}

	return links.Link{
		ID:          id,
		Code:        l.Code,
		OriginalURL: l.OriginalURL,
		DisabledAt:  disabledAt,
		ExpiresAt:   expiresAt,
	}, nil
}

func linkKey(code string) string {
	return linkKeyPrefix + code
}

func linkMissKey(code string) string {
	return linkMissKeyPrefix + code
}
