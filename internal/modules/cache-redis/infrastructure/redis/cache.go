package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/cache-redis/domain"
)

// Client is the subset of Redis operations required by Medusa's Redis cache.
type Client interface {
	Get(context.Context, string) (string, error)
	Set(context.Context, string, string, time.Duration) error
	Scan(context.Context, uint64, string, int64) (uint64, []string, error)
	Unlink(context.Context, ...string) error
}
type Cache struct {
	client     Client
	namespace  string
	defaultTTL time.Duration
}

func New(client Client, namespace string, ttlSeconds int) *Cache {
	if namespace == "" {
		namespace = "medusa"
	}
	if ttlSeconds == 0 {
		ttlSeconds = 30
	}
	return &Cache{client: client, namespace: namespace, defaultTTL: time.Duration(ttlSeconds) * time.Second}
}
func (c *Cache) key(k string) string { return c.namespace + ":" + k }
func (c *Cache) Get(ctx context.Context, key string) (any, bool, error) {
	k := c.key(key)
	raw, err := c.client.Get(ctx, k)
	if err != nil {
		if unlinkErr := c.client.Unlink(ctx, k); unlinkErr != nil {
			return nil, false, unlinkErr
		}
		return nil, false, nil
	}
	var value any
	if err = json.Unmarshal([]byte(raw), &value); err != nil {
		if unlinkErr := c.client.Unlink(ctx, k); unlinkErr != nil {
			return nil, false, unlinkErr
		}
		return nil, false, nil
	}
	return value, true, nil
}
func (c *Cache) Set(ctx context.Context, key string, value any, ttlSeconds int) error {
	if ttlSeconds == 0 {
		return nil
	}
	ttl := c.defaultTTL
	if ttlSeconds > 0 {
		ttl = time.Duration(ttlSeconds) * time.Second
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(key), string(raw), ttl)
}
func (c *Cache) Invalidate(ctx context.Context, key string) error {
	pattern := c.key(key)
	var cursor uint64
	for {
		next, keys, err := c.client.Scan(ctx, cursor, pattern, 100)
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err = c.client.Unlink(ctx, keys...); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}
func (c *Cache) Clear(ctx context.Context) error { return c.Invalidate(ctx, "*") }
func (c *Cache) String() string {
	return fmt.Sprintf("redis cache namespace=%s", strings.TrimSuffix(c.namespace, ":"))
}

var _ domain.Cache = (*Cache)(nil)
