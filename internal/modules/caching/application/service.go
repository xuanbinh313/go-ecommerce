package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/example/medusa-go-clone/internal/modules/caching/domain"
)

const DefaultTTLSeconds = 3600

type Service struct {
	providers domain.ProviderRegistry
	strategy  domain.Strategy
	defaultID string
	ttl       int
	logger    domain.Logger
	mu        sync.Mutex
	pending   map[string]*call
}

type call struct {
	done chan struct{}
	val  any
	err  error
}

func New(providers domain.ProviderRegistry, strategy domain.Strategy, defaultProviderID string, ttlOption *int, logger domain.Logger) *Service {
	ttlSeconds := DefaultTTLSeconds
	if ttlOption != nil {
		ttlSeconds = *ttlOption
	}
	return &Service{providers: providers, strategy: strategy, defaultID: defaultProviderID, ttl: ttlSeconds, logger: logger, pending: map[string]*call{}}
}

func requestKey(key string, tags, providers []string, defaultID string, clear bool) string {
	tagCopy := append([]string(nil), tags...)
	sort.Strings(tagCopy)
	providerCopy := append([]string(nil), providers...)
	if len(providerCopy) == 0 {
		providerCopy = []string{defaultID}
	}
	prefix := ""
	if clear {
		prefix = "clear:"
	}
	return prefix + key + "|" + strings.Join(tagCopy, ",") + "|" + strings.Join(providerCopy, ",")
}

func (s *Service) do(ctx context.Context, key string, fn func() (any, error)) (any, error) {
	s.mu.Lock()
	if active := s.pending[key]; active != nil {
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-active.done:
			return active.val, active.err
		}
	}
	active := &call{done: make(chan struct{})}
	s.pending[key] = active
	s.mu.Unlock()
	func() {
		defer func() {
			s.mu.Lock()
			delete(s.pending, key)
			close(active.done)
			s.mu.Unlock()
		}()
		active.val, active.err = fn()
	}()
	return active.val, active.err
}

func (s *Service) Get(ctx context.Context, key string, tags, providers []string) (any, error) {
	if key == "" && tags == nil {
		return nil, fmt.Errorf("either key or tags must be provided")
	}
	return s.do(ctx, requestKey(key, tags, providers, s.defaultID, false), func() (any, error) {
		ids := providers
		if len(ids) == 0 {
			ids = []string{s.defaultID}
		}
		for _, id := range ids {
			p, err := s.providers.RetrieveProvider(id)
			if err != nil {
				if s.logger != nil {
					s.logger.Warn("cache provider %s failed: %v", id, err)
				}
				continue
			}
			v, err := p.Get(ctx, key, tags)
			if err != nil {
				if s.logger != nil {
					s.logger.Warn("cache provider %s failed: %v", id, err)
				}
				continue
			}
			if v != nil {
				return v, nil
			}
		}
		return nil, nil
	})
}

type ProviderOption struct {
	ID  string
	TTL *int
}
type SetOptions struct {
	Key                  string
	Data                 any
	TTL                  *int
	Tags                 []string
	ComputeAutomaticTags bool
	Providers            []ProviderOption
	AutoInvalidate       *bool
}

func (s *Service) Set(ctx context.Context, in SetOptions) error {
	if in.Key == "" {
		return fmt.Errorf("[CachingModuleService] Key must be provided")
	}
	tags := append([]string(nil), in.Tags...)
	if in.Tags == nil || in.ComputeAutomaticTags {
		computed, err := s.strategy.ComputeTags(ctx, in.Data, nil)
		if err != nil {
			return err
		}
		tags = append(tags, computed...)
	}
	seen := map[string]bool{}
	uniq := tags[:0]
	for _, tag := range tags {
		if !seen[tag] {
			seen[tag] = true
			uniq = append(uniq, tag)
		}
	}
	tags = uniq
	providers := in.Providers
	if len(providers) == 0 {
		providers = []ProviderOption{{ID: s.defaultID}}
	}
	ids := make([]string, len(providers))
	for i, p := range providers {
		ids[i] = p.ID
	}
	_, err := s.do(ctx, requestKey(in.Key, tags, ids, s.defaultID, false), func() (any, error) {
		providerOptions := map[string]any(nil)
		if in.AutoInvalidate != nil {
			providerOptions = map[string]any{"autoInvalidate": *in.AutoInvalidate}
		}
		for _, opt := range providers {
			p, e := s.providers.RetrieveProvider(opt.ID)
			if e != nil {
				return nil, e
			}
			ttl := s.ttl
			if in.TTL != nil {
				ttl = *in.TTL
			}
			if opt.TTL != nil {
				ttl = *opt.TTL
			}
			if e = p.Set(ctx, in.Key, tags, in.Data, ttl, providerOptions); e != nil {
				return nil, e
			}
		}
		return nil, nil
	})
	return err
}

func (s *Service) Clear(ctx context.Context, key string, tags []string, providers []string, autoInvalidate bool) error {
	if key == "" && tags == nil {
		return fmt.Errorf("either key or tags must be provided")
	}
	_, err := s.do(ctx, requestKey(key, tags, providers, s.defaultID, true), func() (any, error) {
		ids := providers
		if len(ids) == 0 {
			ids = []string{s.defaultID}
		}
		providerOptions := map[string]any(nil)
		if autoInvalidate {
			providerOptions = map[string]any{"autoInvalidate": true}
		}
		for _, id := range ids {
			p, e := s.providers.RetrieveProvider(id)
			if e != nil {
				return nil, e
			}
			if e = p.Clear(ctx, key, tags, providerOptions); e != nil {
				return nil, e
			}
		}
		return nil, nil
	})
	return err
}

func (s *Service) ComputeKey(ctx context.Context, input any) (string, error) {
	return s.strategy.ComputeKey(ctx, input)
}
func (s *Service) ComputeTags(ctx context.Context, input any, options map[string]any) ([]string, error) {
	return s.strategy.ComputeTags(ctx, input, options)
}
