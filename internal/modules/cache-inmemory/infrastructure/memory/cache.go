package memory

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"
)

type record struct {
	value   any
	expires time.Time
	timer   *time.Timer
}
type Cache struct {
	mu         sync.RWMutex
	records    map[string]record
	defaultTTL time.Duration
}

func New(defaultTTLSeconds int) *Cache {
	if defaultTTLSeconds == 0 {
		defaultTTLSeconds = 30
	}
	return &Cache{records: map[string]record{}, defaultTTL: time.Duration(defaultTTLSeconds) * time.Second}
}
func (c *Cache) Get(ctx context.Context, key string) (any, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	c.mu.RLock()
	r, ok := c.records[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if !r.expires.IsZero() && time.Now().After(r.expires) {
		_ = c.Invalidate(ctx, key)
		return nil, false, nil
	}
	return r.value, true, nil
}
func (c *Cache) Set(ctx context.Context, key string, value any, ttlSeconds int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ttlSeconds == 0 {
		return nil
	}
	ttl := c.defaultTTL
	if ttlSeconds > 0 {
		ttl = time.Duration(ttlSeconds) * time.Second
	}
	expires := time.Now().Add(ttl)
	var timer *time.Timer
	if ttl > 0 {
		timer = time.AfterFunc(ttl, func() {
			c.mu.Lock()
			defer c.mu.Unlock()
			r, ok := c.records[key]
			if ok && !r.expires.After(time.Now()) {
				delete(c.records, key)
			}
		})
	}
	c.mu.Lock()
	if old, ok := c.records[key]; ok && old.timer != nil {
		old.timer.Stop()
	}
	c.records[key] = record{value: value, expires: expires, timer: timer}
	c.mu.Unlock()
	return nil
}
func (c *Cache) Invalidate(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !strings.Contains(key, "*") {
		if r, ok := c.records[key]; ok && r.timer != nil {
			r.timer.Stop()
		}
		delete(c.records, key)
		return nil
	}
	pattern := regexp.QuoteMeta(key)
	pattern = strings.ReplaceAll(pattern, "\\*", ".*")
	re, err := regexp.Compile("^" + pattern + "$")
	if err != nil {
		return err
	}
	for k, r := range c.records {
		if re.MatchString(k) {
			if r.timer != nil {
				r.timer.Stop()
			}
			delete(c.records, k)
		}
	}
	return nil
}
func (c *Cache) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range c.records {
		if r.timer != nil {
			r.timer.Stop()
		}
	}
	clear(c.records)
	return nil
}
