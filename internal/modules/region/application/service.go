package application

import (
	"context"
	"errors"
	"github.com/example/medusa-go-clone/internal/modules/region/domain"
	"github.com/example/medusa-go-clone/internal/modules/region/domain/models"
	"strings"
)

type Service struct{ repository domain.Repository }

func NewService(repository domain.Repository) *Service { return &Service{repository: repository} }
func (s *Service) Create(ctx context.Context, in models.RegionInput) (models.Region, error) {
	normalize(&in)
	if err := s.validateCountries(ctx, in.Countries, ""); err != nil {
		return models.Region{}, err
	}
	return s.repository.Create(ctx, in)
}
func (s *Service) Get(ctx context.Context, id string) (models.Region, error) {
	return s.repository.Get(ctx, id)
}
func (s *Service) List(ctx context.Context) ([]models.Region, error) { return s.repository.List(ctx) }
func (s *Service) Update(ctx context.Context, id string, in models.RegionInput) (models.Region, error) {
	normalize(&in)
	if in.Countries != nil {
		if err := s.validateCountries(ctx, in.Countries, id); err != nil {
			return models.Region{}, err
		}
	}
	return s.repository.Update(ctx, id, in)
}
func (s *Service) SoftDelete(ctx context.Context, ids []string) error {
	return s.repository.SoftDelete(ctx, ids)
}
func (s *Service) RestoreRegions(ctx context.Context, ids []string) error {
	return s.repository.RestoreRegions(ctx, ids)
}
func (s *Service) DeleteRegions(ctx context.Context, ids []string) error {
	return s.repository.DeleteRegions(ctx, ids)
}
func (s *Service) validateCountries(ctx context.Context, codes []string, regionID string) error {
	if len(codes) == 0 {
		return nil
	}
	seen := map[string]bool{}
	for _, code := range codes {
		if seen[code] {
			return errors.New("countries are already assigned to a region")
		}
		seen[code] = true
	}
	countries, err := s.repository.Countries(ctx, codes, regionID)
	if err != nil {
		return err
	}
	if len(countries) != len(codes) {
		return errors.New("one or more countries do not exist")
	}
	for _, country := range countries {
		if country.RegionID != nil && *country.RegionID != "" && *country.RegionID != regionID {
			return errors.New("one or more countries are already assigned to a region")
		}
	}
	return nil
}
func normalize(in *models.RegionInput) {
	if in.Name != nil {
		value := strings.TrimSpace(*in.Name)
		in.Name = &value
	}
	if in.CurrencyCode != nil {
		value := strings.ToLower(strings.TrimSpace(*in.CurrencyCode))
		in.CurrencyCode = &value
	}
	for i := range in.Countries {
		in.Countries[i] = strings.ToLower(in.Countries[i])
	}
}
