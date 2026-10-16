package application

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/pricing/domain"
	"github.com/example/medusa-go-clone/internal/modules/pricing/domain/models"
)

type Service struct{ repo domain.Repository }

func NewService(repo domain.Repository) *Service { return &Service{repo: repo} }
func (s *Service) ListPreferences(ctx context.Context, attributes, values []string, limit, offset int) ([]models.PricePreference, int, error) {
	return s.repo.ListPreferences(ctx, attributes, values, limit, offset)
}
func (s *Service) GetPreference(ctx context.Context, id string) (models.PricePreference, error) {
	return s.repo.GetPreference(ctx, id)
}
func (s *Service) CreatePreference(ctx context.Context, in models.PreferenceInput) (models.PricePreference, error) {
	return s.repo.CreatePreference(ctx, in)
}
func (s *Service) UpdatePreference(ctx context.Context, id string, in models.PreferenceUpdate) (models.PricePreference, error) {
	return s.repo.UpdatePreference(ctx, id, in)
}
func (s *Service) DeletePreference(ctx context.Context, id string) error {
	return s.repo.DeletePreference(ctx, id)
}
func (s *Service) ListPriceLists(ctx context.Context, ids, statuses, types []string, limit, offset int) ([]models.PriceList, int, error) {
	return s.repo.ListPriceLists(ctx, ids, statuses, types, limit, offset)
}
func (s *Service) GetPriceList(ctx context.Context, id string) (models.PriceList, error) {
	return s.repo.GetPriceList(ctx, id)
}
func (s *Service) CreatePriceList(ctx context.Context, in models.PriceList) (models.PriceList, error) {
	return s.repo.CreatePriceList(ctx, in)
}
func (s *Service) UpdatePriceList(ctx context.Context, id string, in models.PriceList) (models.PriceList, error) {
	return s.repo.UpdatePriceList(ctx, id, in)
}
func (s *Service) DeletePriceList(ctx context.Context, id string) error {
	return s.repo.DeletePriceList(ctx, id)
}
