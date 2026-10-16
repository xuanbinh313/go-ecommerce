package application

import (
	"context"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/analytics/domain"
	"github.com/example/medusa-go-clone/internal/modules/analytics/domain/models"
)

const RegistrationPrefix = "aly_"
const IdentifierRegistrationName = "analytics_providers_identifier"

type Service struct{ provider domain.Provider }

func New(providers map[string]domain.Provider) (*Service, error) {
	if len(providers) != 1 {
		return nil, fmt.Errorf("analytics module should be initialized with exactly one provider")
	}
	for _, p := range providers {
		return &Service{provider: p}, nil
	}
	return nil, fmt.Errorf("analytics provider missing")
}
func (s *Service) Track(ctx context.Context, event models.TrackEvent) error {
	if err := s.provider.Track(ctx, event); err != nil {
		return fmt.Errorf("error tracking event for %s: %w", event.Event, err)
	}
	return nil
}
func (s *Service) Identify(ctx context.Context, event models.IdentifyEvent) error {
	if err := s.provider.Identify(ctx, event); err != nil {
		id := event.ActorID
		if event.Group != nil {
			id = event.Group.ID
		}
		return fmt.Errorf("error identifying event for %s: %w", id, err)
	}
	return nil
}
func (s *Service) Shutdown(ctx context.Context) error { return s.provider.Shutdown(ctx) }
