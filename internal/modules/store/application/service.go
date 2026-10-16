package application

import (
	"context"
	"errors"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/store/domain"
	"github.com/example/medusa-go-clone/internal/modules/store/domain/models"
)

type Service struct{ repository domain.Repository }

func NewService(repository domain.Repository) *Service { return &Service{repository: repository} }

func (s *Service) Create(ctx context.Context, in models.StoreInput) (models.Store, error) {
	normalize(&in)
	if err := validateCurrencies(in.SupportedCurrencies); err != nil {
		return models.Store{}, err
	}
	return s.repository.Create(ctx, in)
}
func (s *Service) Get(ctx context.Context, id string) (models.Store, error) {
	return s.repository.Get(ctx, id)
}
func (s *Service) List(ctx context.Context) ([]models.Store, error) { return s.repository.List(ctx) }
func (s *Service) Update(ctx context.Context, id string, in models.StoreInput) (models.Store, error) {
	normalize(&in)
	if err := validateCurrencies(in.SupportedCurrencies); err != nil {
		return models.Store{}, err
	}
	return s.repository.Update(ctx, id, in)
}
func (s *Service) Delete(ctx context.Context, id string) error { return s.repository.Delete(ctx, id) }
func (s *Service) DeleteStores(ctx context.Context, ids []string) error {
	return s.repository.DeleteStores(ctx, ids)
}

func normalize(in *models.StoreInput) {
	if in.Name != nil {
		value := strings.TrimSpace(*in.Name)
		in.Name = &value
	}
	for i := range in.SupportedCurrencies {
		in.SupportedCurrencies[i].CurrencyCode = strings.ToLower(in.SupportedCurrencies[i].CurrencyCode)
	}
}
func validateCurrencies(currencies []models.StoreCurrencyInput) error {
	if len(currencies) == 0 {
		return nil
	}
	seen := map[string]bool{}
	defaults := 0
	for _, currency := range currencies {
		if seen[currency.CurrencyCode] {
			return errors.New("duplicate currency codes")
		}
		seen[currency.CurrencyCode] = true
		if currency.IsDefault {
			defaults++
		}
	}
	if defaults != 1 {
		return errors.New("store requires exactly one default currency")
	}
	return nil
}
