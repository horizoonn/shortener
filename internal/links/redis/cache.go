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

type Cache struct {
	client *goredis.Client
	ttl    time.Duration
}

type cachedLink struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	OriginalURL string `json:"original_url"`
}

func NewCache(client *goredis.Client, ttl time.Duration) (*Cache, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is required")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("redis cache TTL must be positive")
	}

	return &Cache{
		client: client,
		ttl:    ttl,
	}, nil
}

func (c *Cache) GetLink(ctx context.Context, code string) (links.Link, error) {
	data, err := c.client.Get(ctx, linkKey(code)).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return links.Link{}, fmt.Errorf("link cache miss: %w", core_errors.ErrNotFound)
		}

		return links.Link{}, fmt.Errorf("get link from redis: %w", err)
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

	if err := c.client.Set(ctx, linkKey(link.Code), data, c.ttl).Err(); err != nil {
		return fmt.Errorf("set link in redis: %w", err)
	}

	return nil
}

func (c *Cache) DeleteLink(ctx context.Context, code string) error {
	if err := c.client.Del(ctx, linkKey(code)).Err(); err != nil {
		return fmt.Errorf("delete link from redis: %w", err)
	}

	return nil
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func newCachedLink(link links.Link) cachedLink {
	return cachedLink{
		ID:          link.ID.String(),
		Code:        link.Code,
		OriginalURL: link.OriginalURL,
	}
}

func (l cachedLink) toLink() (links.Link, error) {
	id, err := uuid.Parse(l.ID)
	if err != nil {
		return links.Link{}, fmt.Errorf("parse link ID: %w", err)
	}

	return links.Link{
		ID:          id,
		Code:        l.Code,
		OriginalURL: l.OriginalURL,
	}, nil
}

func linkKey(code string) string {
	return linkKeyPrefix + code
}
