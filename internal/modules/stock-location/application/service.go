package application

import (
	"context"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/stock-location/domain"
	"github.com/example/medusa-go-clone/internal/modules/stock-location/domain/models"
	"strings"
)

type Service struct{ repository domain.Repository }

func NewService(r domain.Repository) *Service { return &Service{repository: r} }
func (s *Service) List(c context.Context, f models.Filter) ([]models.StockLocation, int, error) {
	return s.repository.List(c, f)
}
func (s *Service) Get(c context.Context, id string) (models.StockLocation, error) {
	return s.repository.Get(c, id)
}
func (s *Service) Create(c context.Context, in models.Input) (models.StockLocation, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return models.StockLocation{}, fmt.Errorf("name is required")
	}
	if in.Address != nil && in.Address.CountryCode == "" {
		return models.StockLocation{}, fmt.Errorf("address.country_code is required")
	}
	return s.repository.Create(c, in)
}
func (s *Service) Update(c context.Context, id string, in models.UpdateInput) (models.StockLocation, error) {
	if in.Name != nil {
		*in.Name = strings.TrimSpace(*in.Name)
		if *in.Name == "" {
			return models.StockLocation{}, fmt.Errorf("name must not be empty")
		}
	}
	if in.Address != nil && in.Address.CountryCode == "" {
		return models.StockLocation{}, fmt.Errorf("address.country_code is required")
	}
	return s.repository.Update(c, id, in)
}
func (s *Service) Delete(c context.Context, id string) error { return s.repository.Delete(c, id) }
func (s *Service) LinkSalesChannels(c context.Context, id string, add, remove []string) error {
	return s.repository.LinkSalesChannels(c, id, add, remove)
}
