package application

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/currency/domain"
	"github.com/example/medusa-go-clone/internal/modules/currency/domain/models"
	"strings"
)

type Service struct{ repository domain.Repository }

func NewService(repository domain.Repository) *Service { return &Service{repository: repository} }
func (s *Service) RetrieveCurrency(ctx context.Context, code string) (models.Currency, error) {
	return s.repository.Retrieve(ctx, strings.ToLower(code))
}
func (s *Service) ListCurrencies(ctx context.Context, codes []string) ([]models.Currency, error) {
	normalized := make([]string, len(codes))
	for i, code := range codes {
		normalized[i] = strings.ToLower(code)
	}
	return s.repository.List(ctx, normalized)
}
func (s *Service) UpsertCurrencies(ctx context.Context, currencies []models.Currency) ([]models.Currency, error) {
	for i := range currencies {
		currencies[i].Code = strings.ToLower(currencies[i].Code)
	}
	return s.repository.Upsert(ctx, currencies)
}
