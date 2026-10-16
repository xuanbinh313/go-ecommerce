package memory

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/caching/domain"
)

const (
	DefaultTTLSeconds   = 3600
	DefaultMaxKeys      = 25000
	DefaultMaxSizeBytes = 300 * 1024 * 1024
)

type HashFunc func(string) string
type Logger interface{ Warn(string, ...any) }

type Options struct {
	TTLSeconds   int
	MaxKeys      int
	CheckPeriod  time.Duration
	MaxSizeBytes int64
}

type entry struct {
	value   any
	expires time.Time
	tags    map[string]struct{}
	options map[string]any
	bytes   int64
}

type Provider struct {
	mu         sync.Mutex
	entries    map[string]entry
	tagKeys    map[string]map[string]struct{}
	hash       HashFunc
	logger     Logger
	defaultTTL time.Duration
	maxKeys    int
	maxSize    int64
	used       int64
	stop       chan struct{}
	closed     chan struct{}
	stopOnce   sync.Once
}

func New(options Options, hasher HashFunc, logger Logger) *Provider {
	if options.TTLSeconds == 0 {
		options.TTLSeconds = DefaultTTLSeconds
	}
	if options.MaxKeys == 0 {
		options.MaxKeys = DefaultMaxKeys
	}
	if options.MaxSizeBytes == 0 {
		options.MaxSizeBytes = DefaultMaxSizeBytes
	}
	if options.CheckPeriod <= 0 {
		options.CheckPeriod = 60 * time.Second
	}
	if hasher == nil {
		hasher = Hash
	}
	p := &Provider{entries: map[string]entry{}, tagKeys: map[string]map[string]struct{}{}, hash: hasher, logger: logger, defaultTTL: time.Duration(options.TTLSeconds) * time.Second, maxKeys: options.MaxKeys, maxSize: options.MaxSizeBytes, stop: make(chan struct{}), closed: make(chan struct{})}
	go p.cleanupLoop(options.CheckPeriod)
	return p
}

func (p *Provider) cleanupLoop(period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	defer close(p.closed)
	for {
		select {
		case <-ticker.C:
			p.mu.Lock()
			now := time.Now()
			for key, e := range p.entries {
				if !e.expires.IsZero() && !e.expires.After(now) {
					p.deleteLocked(key)
				}
			}
			p.mu.Unlock()
		case <-p.stop:
			return
		}
	}
}

func (p *Provider) deleteLocked(key string) {
	e, ok := p.entries[key]
	if !ok {
		return
	}
	delete(p.entries, key)
	p.used -= e.bytes
	for tag := range e.tags {
		keys := p.tagKeys[tag]
		delete(keys, key)
		if len(keys) == 0 {
			delete(p.tagKeys, tag)
		}
	}
}

func (p *Provider) Get(ctx context.Context, key string, tags []string) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	if key != "" {
		hashed := p.hash(key)
		e, ok := p.entries[hashed]
		if ok && !e.expires.IsZero() && !e.expires.After(now) {
			p.deleteLocked(hashed)
			ok = false
		}
		if !ok {
			return nil, nil
		}
		return e.value, nil
	}
	keys := map[string]struct{}{}
	for _, tag := range tags {
		for k := range p.tagKeys[p.hash(tag)] {
			keys[k] = struct{}{}
		}
	}
	result := make([]any, 0, len(keys))
	for k := range keys {
		e, ok := p.entries[k]
		if ok && !e.expires.IsZero() && !e.expires.After(now) {
			p.deleteLocked(k)
			continue
		}
		if ok {
			result = append(result, e.value)
		}
	}
	return result, nil
}

func (p *Provider) Set(ctx context.Context, key string, tags []string, value any, ttl int, options map[string]any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	hashedKey := p.hash(key)
	if _, exists := p.entries[hashedKey]; !exists && len(p.entries) >= p.maxKeys {
		if p.logger != nil {
			p.logger.Warn("cache key limit reached; skipping entry")
		}
		return nil
	}
	var body []byte
	body, _ = json.Marshal(value)
	hashedTags := map[string]struct{}{}
	for _, tag := range tags {
		hashedTags[p.hash(tag)] = struct{}{}
	}
	var tagBytes []byte
	if len(hashedTags) > 0 {
		tagBytes, _ = json.Marshal(hashedTags)
	}
	size := int64(len(body) + len(hashedKey) + len(tagBytes))
	if p.used > p.maxSize {
		if p.logger != nil {
			p.logger.Warn("cache size limit exceeded; skipping entry")
		}
		return nil
	}
	expirySeconds := ttl
	var expires time.Time
	if expirySeconds > 0 {
		expires = time.Now().Add(time.Duration(expirySeconds) * time.Second)
	}
	entryOptions := map[string]any{}
	for k, v := range options {
		entryOptions[k] = v
	}
	p.deleteLocked(hashedKey)
	// Replacing an existing key does not consume another node-cache slot.
	p.entries[hashedKey] = entry{value: value, expires: expires, tags: hashedTags, options: entryOptions, bytes: size}
	p.used += size
	for tag := range hashedTags {
		if p.tagKeys[tag] == nil {
			p.tagKeys[tag] = map[string]struct{}{}
		}
		p.tagKeys[tag][hashedKey] = struct{}{}
	}
	return nil
}

func (p *Provider) Clear(ctx context.Context, key string, tags []string, options map[string]any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if key != "" {
		p.deleteLocked(p.hash(key))
		return nil
	}
	for _, tag := range tags {
		if tag == "*" {
			p.entries = map[string]entry{}
			p.tagKeys = map[string]map[string]struct{}{}
			p.used = 0
			return nil
		}
	}
	keys := map[string]struct{}{}
	for _, tag := range tags {
		for k := range p.tagKeys[p.hash(tag)] {
			keys[k] = struct{}{}
		}
	}
	for k := range keys {
		e, ok := p.entries[k]
		if !ok {
			continue
		}
		if auto, provided := options["autoInvalidate"]; provided && auto == true {
			if configured, ok := e.options["autoInvalidate"].(bool); ok && !configured {
				continue
			}
		}
		p.deleteLocked(k)
	}
	return nil
}

func (p *Provider) Close() { p.stopOnce.Do(func() { close(p.stop) }); <-p.closed }

var _ domain.Provider = (*Provider)(nil)
