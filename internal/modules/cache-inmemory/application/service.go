package application

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/cache-inmemory/domain"
)

type Service struct{ cache domain.Cache }

func New(cache domain.Cache) *Service { return &Service{cache: cache} }
func (s *Service) Get(ctx context.Context, key string) (any, bool, error) {
	return s.cache.Get(ctx, key)
}
func (s *Service) Set(ctx context.Context, key string, value any, ttl int) error {
	return s.cache.Set(ctx, key, value, ttl)
}
func (s *Service) Invalidate(ctx context.Context, key string) error {
	return s.cache.Invalidate(ctx, key)
}
func (s *Service) Clear(ctx context.Context) error { return s.cache.Clear(ctx) }
