package application

import (
	"context"
	"fmt"
	"sort"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain"
	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
)

type ProviderService struct{ providers map[string]domain.Provider }

func NewProviderService(providers ...domain.Provider) (*ProviderService, error) {
	registry := &ProviderService{providers: map[string]domain.Provider{}}
	for _, provider := range providers {
		if provider == nil || provider.Identifier() == "" {
			continue
		}
		id := provider.Identifier()
		if _, exists := registry.providers[id]; exists {
			return nil, fmt.Errorf("fulfillment provider %q is registered twice", id)
		}
		registry.providers[id] = provider
	}
	return registry, nil
}
func (s *ProviderService) Retrieve(id string) (domain.Provider, error) {
	provider := s.providers[id]
	if provider == nil {
		return nil, fmt.Errorf("unable to retrieve fulfillment provider with id: %s", id)
	}
	return provider, nil
}
func (s *ProviderService) List() []models.FulfillmentProvider {
	ids := make([]string, 0, len(s.providers))
	for id := range s.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]models.FulfillmentProvider, 0, len(ids))
	for _, id := range ids {
		out = append(out, models.FulfillmentProvider{ID: id, IsEnabled: true})
	}
	return out
}
func (s *ProviderService) GetFulfillmentOptions(ctx context.Context, id string) ([]models.FulfillmentOption, error) {
	p, e := s.Retrieve(id)
	if e != nil {
		return nil, e
	}
	return p.GetFulfillmentOptions(ctx)
}
func (s *ProviderService) ValidateFulfillmentData(ctx context.Context, id string, optionData, data, contextData map[string]any) (map[string]any, error) {
	p, e := s.Retrieve(id)
	if e != nil {
		return nil, e
	}
	return p.ValidateFulfillmentData(ctx, optionData, data, contextData)
}
func (s *ProviderService) CalculatePrice(ctx context.Context, id string, optionData, data, contextData map[string]any) (models.ShippingPriceResult, error) {
	p, e := s.Retrieve(id)
	if e != nil {
		return models.ShippingPriceResult{}, e
	}
	return p.CalculatePrice(ctx, optionData, data, contextData)
}
func (s *ProviderService) CanCalculate(ctx context.Context, id string, data map[string]any) (bool, error) {
	p, e := s.Retrieve(id)
	if e != nil {
		return false, e
	}
	return p.CanCalculate(ctx, data)
}
func (s *ProviderService) ValidateOption(ctx context.Context, id string, data map[string]any) (bool, error) {
	p, e := s.Retrieve(id)
	if e != nil {
		return false, e
	}
	return p.ValidateOption(ctx, data)
}
func (s *ProviderService) CreateFulfillment(ctx context.Context, id string, input models.CreateFulfillmentInput) (models.FulfillmentProviderResult, error) {
	p, e := s.Retrieve(id)
	if e != nil {
		return models.FulfillmentProviderResult{}, e
	}
	return p.CreateFulfillment(ctx, input)
}
func (s *ProviderService) CancelFulfillment(ctx context.Context, id string, fulfillment map[string]any) (map[string]any, error) {
	p, e := s.Retrieve(id)
	if e != nil {
		return nil, e
	}
	return p.CancelFulfillment(ctx, fulfillment)
}
func (s *ProviderService) CreateReturnFulfillment(ctx context.Context, id string, fulfillment map[string]any) (models.FulfillmentProviderResult, error) {
	p, e := s.Retrieve(id)
	if e != nil {
		return models.FulfillmentProviderResult{}, e
	}
	return p.CreateReturnFulfillment(ctx, fulfillment)
}
