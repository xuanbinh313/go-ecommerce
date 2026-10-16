package application

import (
	"context"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/order/domain"
	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
)

type Service struct{ repo domain.Repository }

func New(repo domain.Repository) *Service { return &Service{repo: repo} }
func (s *Service) ListClaims(ctx context.Context, f models.ClaimFilter) ([]models.Claim, int, error) {
	return s.repo.ListClaims(ctx, f)
}
func (s *Service) GetClaim(ctx context.Context, id string) (models.Claim, error) {
	return s.repo.GetClaim(ctx, id)
}
func (s *Service) CreateClaim(ctx context.Context, in models.CreateClaimInput) (models.Claim, error) {
	if in.OrderID == "" || (in.Type != "refund" && in.Type != "replace") {
		return models.Claim{}, fmt.Errorf("order_id and a valid claim type are required")
	}
	claim, err := s.repo.CreateClaim(ctx, in)
	if err != nil {
		return models.Claim{}, err
	}
	if err = s.repo.CreateOrderChangeForClaim(ctx, in, claim.ID); err != nil {
		if compensationErr := s.repo.SoftDeleteClaims(ctx, []string{claim.ID}); compensationErr != nil {
			return models.Claim{}, fmt.Errorf("create order change: %w; compensate claim: %v", err, compensationErr)
		}
		return models.Claim{}, err
	}
	return claim, nil
}
func (s *Service) CancelBeginClaim(ctx context.Context, id string) error {
	return s.repo.CancelBeginClaim(ctx, id)
}
func (s *Service) ListReturnReasons(ctx context.Context, f models.ReturnReasonFilter, limit, offset int) ([]models.ReturnReason, int, error) {
	return s.repo.ListReturnReasons(ctx, f, limit, offset)
}
func (s *Service) GetReturnReason(ctx context.Context, id string) (models.ReturnReason, error) {
	return s.repo.GetReturnReason(ctx, id)
}
func (s *Service) CreateReturnReasons(ctx context.Context, in []models.ReturnReasonInput) ([]models.ReturnReason, error) {
	return s.repo.CreateReturnReasons(ctx, in)
}
func (s *Service) UpdateReturnReasons(ctx context.Context, ids []string, fields map[string]any) ([]models.ReturnReason, error) {
	return s.repo.UpdateReturnReasons(ctx, ids, fields)
}
func (s *Service) SoftDeleteReturnReasons(ctx context.Context, ids []string) error {
	return s.repo.SoftDeleteReturnReasons(ctx, ids)
}
func (s *Service) RestoreReturnReasons(ctx context.Context, ids []string) error {
	return s.repo.RestoreReturnReasons(ctx, ids)
}
func (s *Service) HardDeleteReturnReasons(ctx context.Context, ids []string) error {
	return s.repo.HardDeleteReturnReasons(ctx, ids)
}
func (s *Service) List(ctx context.Context, f models.Filter) (models.ListResult, error) {
	return s.repo.List(ctx, f)
}
func (s *Service) Get(ctx context.Context, id string, version *int) (models.Order, error) {
	return s.repo.Get(ctx, id, version)
}
func (s *Service) Archive(ctx context.Context, ids []string) ([]models.Order, error) {
	return s.repo.Archive(ctx, ids)
}
func (s *Service) Complete(ctx context.Context, ids []string) ([]models.Order, map[string]string, error) {
	return s.repo.Complete(ctx, ids)
}
func (s *Service) RestoreStatuses(ctx context.Context, statuses map[string]string) error {
	return s.repo.RestoreStatuses(ctx, statuses)
}
func (s *Service) Update(ctx context.Context, input models.UpdateInput) (models.Order, error) {
	return s.repo.Update(ctx, input)
}
func (s *Service) DeleteDraftOrders(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.repo.DeleteDraftOrders(ctx, ids)
}
