package application

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/sales-channel/domain"
	"github.com/example/medusa-go-clone/internal/modules/sales-channel/domain/models"
	"strings"
)

type Service struct{ repository domain.Repository }

func NewService(repository domain.Repository) *Service { return &Service{repository: repository} }
func (s *Service) Create(ctx context.Context, in models.Input) (models.SalesChannel, error) {
	normalize(&in)
	return s.repository.Create(ctx, in)
}
func (s *Service) Get(ctx context.Context, id string) (models.SalesChannel, error) {
	return s.repository.Get(ctx, id)
}
func (s *Service) List(ctx context.Context) ([]models.SalesChannel, error) {
	return s.repository.List(ctx)
}
func (s *Service) Update(ctx context.Context, id string, in models.Input) (models.SalesChannel, error) {
	normalize(&in)
	return s.repository.Update(ctx, id, in)
}
func (s *Service) Delete(ctx context.Context, id string) error { return s.repository.Delete(ctx, id) }
func (s *Service) Restore(ctx context.Context, ids []string) error {
	return s.repository.Restore(ctx, ids)
}
func (s *Service) DeleteMany(ctx context.Context, ids []string) error {
	return s.repository.DeleteMany(ctx, ids)
}
func normalize(in *models.Input) {
	if in.Name != nil {
		value := strings.TrimSpace(*in.Name)
		in.Name = &value
	}
}
