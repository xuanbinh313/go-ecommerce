package application

import (
	"context"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/tax/domain"
	"github.com/example/medusa-go-clone/internal/modules/tax/domain/models"
)

type Service struct{ repo domain.Repository }

func NewService(repo domain.Repository) *Service { return &Service{repo: repo} }
func (s *Service) ListProviders(c context.Context, ids []string, enabled *bool, limit, offset int) ([]models.TaxProvider, int, error) {
	return s.repo.ListProviders(c, ids, enabled, limit, offset)
}
func (s *Service) ListRegions(c context.Context, ids, countries, parents []string, limit, offset int) ([]models.TaxRegion, int, error) {
	return s.repo.ListRegions(c, ids, countries, parents, limit, offset)
}
func (s *Service) GetRegion(c context.Context, id string) (models.TaxRegion, error) {
	return s.repo.GetRegion(c, id)
}
func (s *Service) CreateRegion(c context.Context, in models.RegionInput) (models.TaxRegion, error) {
	if in.ParentID == nil || *in.ParentID == "" {
		if in.ProviderID == nil || *in.ProviderID == "" {
			return models.TaxRegion{}, fmt.Errorf("provider is required when creating a non-province tax region")
		}
	} else if in.ProvinceCode == nil || *in.ProvinceCode == "" {
		return models.TaxRegion{}, fmt.Errorf("province_code is required for a province tax region")
	}
	return s.repo.CreateRegion(c, in)
}
func (s *Service) UpdateRegion(c context.Context, id string, in models.RegionUpdate) (models.TaxRegion, error) {
	return s.repo.UpdateRegion(c, id, in)
}
func (s *Service) DeleteRegion(c context.Context, id string) error { return s.repo.DeleteRegion(c, id) }
func (s *Service) ListRates(c context.Context, ids, regions, codes []string, limit, offset int) ([]models.TaxRate, int, error) {
	return s.repo.ListRates(c, ids, regions, codes, limit, offset)
}
func (s *Service) GetRate(c context.Context, id string) (models.TaxRate, error) {
	return s.repo.GetRate(c, id)
}
func (s *Service) CreateRate(c context.Context, in models.RateInput, regionID string, actor *string) (models.TaxRate, error) {
	if in.Code == "" || in.Name == "" {
		return models.TaxRate{}, fmt.Errorf("code and name are required")
	}
	return s.repo.CreateRate(c, in, regionID, actor)
}
func (s *Service) UpdateRate(c context.Context, id string, in models.RateUpdate, actor *string) (models.TaxRate, error) {
	return s.repo.UpdateRate(c, id, in, actor)
}
func (s *Service) DeleteRate(c context.Context, id string) error { return s.repo.DeleteRate(c, id) }
func (s *Service) CreateRateRules(c context.Context, rules []models.RuleCreateInput) error {
	for _, rule := range rules {
		if rule.TaxRateID == "" || rule.Reference == "" || rule.ReferenceID == "" {
			return fmt.Errorf("tax_rate_id, reference, and reference_id are required")
		}
	}
	return s.repo.CreateRateRules(c, rules)
}
func (s *Service) DeleteRateRules(c context.Context, ids []string) error {
	return s.repo.DeleteRateRules(c, ids)
}
