package application

import (
	"context"

	"github.com/example/medusa-go-clone/internal/modules/settings/domain"
	"github.com/example/medusa-go-clone/internal/modules/settings/domain/models"
)

type Service struct{ repository domain.Repository }

func NewService(repository domain.Repository) *Service { return &Service{repository: repository} }

func (s *Service) ListPropertyLabels(ctx context.Context, filter models.PropertyLabelFilter, limit, offset int) ([]models.PropertyLabel, int, error) {
	return s.repository.ListPropertyLabels(ctx, filter, limit, offset)
}
func (s *Service) GetPropertyLabel(ctx context.Context, id string) (models.PropertyLabel, error) {
	return s.repository.GetPropertyLabel(ctx, id)
}
func (s *Service) CreatePropertyLabels(ctx context.Context, input []models.PropertyLabelInput) ([]models.PropertyLabel, error) {
	return s.repository.CreatePropertyLabels(ctx, input)
}
func (s *Service) UpdatePropertyLabel(ctx context.Context, id string, fields map[string]any) (models.PropertyLabel, error) {
	return s.repository.UpdatePropertyLabel(ctx, id, fields)
}
func (s *Service) DeletePropertyLabels(ctx context.Context, ids []string) ([]models.PropertyLabel, error) {
	return s.repository.DeletePropertyLabels(ctx, ids)
}

func (s *Service) ListLayoutConfigurations(ctx context.Context, userID string, filter models.LayoutConfigurationFilter, limit, offset int) ([]models.LayoutConfiguration, int, error) {
	return s.repository.ListLayoutConfigurations(ctx, userID, filter, limit, offset)
}

func (s *Service) GetLayoutConfiguration(ctx context.Context, zone, userID string) (models.LayoutConfigurations, error) {
	return s.repository.GetLayoutConfiguration(ctx, zone, userID)
}

func (s *Service) SetLayoutConfiguration(ctx context.Context, zone, userID string, isDefault bool, configuration []byte) error {
	return s.repository.SetLayoutConfiguration(ctx, zone, userID, isDefault, configuration)
}

func (s *Service) ClearLayoutConfiguration(ctx context.Context, zone, userID string) error {
	return s.repository.ClearLayoutConfiguration(ctx, zone, userID)
}
