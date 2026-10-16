package application

import (
	"context"
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/notification/domain"
	"github.com/example/medusa-go-clone/internal/modules/notification/domain/models"
)

type Service struct {
	repository domain.Repository
	providers  *ProviderService
}

func NewService(repository domain.Repository) *Service { return &Service{repository: repository} }
func NewWithProviderService(repository domain.Repository, providers *ProviderService) *Service {
	return &Service{repository: repository, providers: providers}
}

func (s *Service) List(ctx context.Context, filter models.NotificationFilter, limit, offset int, order string) ([]models.Notification, int, error) {
	return s.repository.List(ctx, filter, limit, offset, order)
}

func (s *Service) Get(ctx context.Context, id string) (models.Notification, error) {
	return s.repository.Get(ctx, id)
}

func (s *Service) Send(ctx context.Context, providerID string, input *models.ProviderSendNotificationInput) (map[string]any, error) {
	if s.providers == nil {
		return nil, fmt.Errorf("notification provider service is not configured")
	}
	return s.providers.Send(ctx, providerID, input)
}

type ProviderService struct{ providers map[string]domain.Provider }

func NewProviderService(providers ...domain.Provider) (*ProviderService, error) {
	registry := &ProviderService{providers: map[string]domain.Provider{}}
	for _, p := range providers {
		if p == nil || p.Identifier() == "" {
			continue
		}
		id := p.Identifier()
		if registry.providers[id] != nil {
			return nil, fmt.Errorf("notification provider %q is registered twice", id)
		}
		registry.providers[id] = p
	}
	return registry, nil
}
func (s *ProviderService) Send(ctx context.Context, id string, input *models.ProviderSendNotificationInput) (map[string]any, error) {
	p := s.providers[id]
	if p == nil {
		return nil, fmt.Errorf("unable to retrieve notification provider with id: %s", id)
	}
	return p.Send(ctx, input)
}
