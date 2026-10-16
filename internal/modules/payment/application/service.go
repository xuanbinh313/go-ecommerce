package application

import (
	"context"

	"github.com/example/medusa-go-clone/internal/modules/payment/domain"
	"github.com/example/medusa-go-clone/internal/modules/payment/domain/models"
)

type Service struct {
	repo      domain.Repository
	providers *ProviderService
}

func New(repo domain.Repository) *Service { return &Service{repo: repo} }

func NewWithProviderService(repo domain.Repository, providers *ProviderService) *Service {
	return &Service{repo: repo, providers: providers}
}

func (s *Service) ProviderService() *ProviderService { return s.providers }

func (s *Service) ListRefundReasons(ctx context.Context, f models.RefundReasonFilter) ([]models.RefundReason, int64, error) {
	return s.repo.ListRefundReasons(ctx, f)
}
func (s *Service) GetRefundReason(ctx context.Context, id string) (*models.RefundReason, error) {
	return s.repo.GetRefundReason(ctx, id)
}
func (s *Service) CreateRefundReasons(ctx context.Context, in []models.RefundReasonInput) ([]models.RefundReason, error) {
	return s.repo.CreateRefundReasons(ctx, in)
}
func (s *Service) UpdateRefundReasons(ctx context.Context, in []models.RefundReason) ([]models.RefundReason, error) {
	return s.repo.UpdateRefundReasons(ctx, in)
}
func (s *Service) SoftDeleteRefundReasons(ctx context.Context, ids []string) error {
	return s.repo.SoftDeleteRefundReasons(ctx, ids)
}
func (s *Service) RestoreRefundReasons(ctx context.Context, ids []string) error {
	return s.repo.RestoreRefundReasons(ctx, ids)
}
func (s *Service) DeleteRefundReasons(ctx context.Context, ids []string) error {
	return s.repo.DeleteRefundReasons(ctx, ids)
}

func (s *Service) CreatePaymentCollections(ctx context.Context, inputs []models.CreatePaymentCollectionInput) ([]models.PaymentCollection, error) {
	created := make([]models.PaymentCollection, 0, len(inputs))
	for _, input := range inputs {
		item, err := s.repo.CreatePaymentCollection(ctx, input)
		if err != nil {
			return nil, err
		}
		created = append(created, *item)
	}
	return created, nil
}
func (s *Service) GetPaymentCollection(ctx context.Context, id string) (*models.PaymentCollection, error) {
	return s.repo.GetPaymentCollection(ctx, id)
}
func (s *Service) DeletePaymentCollections(ctx context.Context, ids []string) error {
	return s.repo.DeletePaymentCollections(ctx, ids)
}
func (s *Service) ListPaymentCollections(ctx context.Context, filters map[string]any, limit, offset int) ([]models.PaymentCollection, int64, error) {
	return s.repo.ListPaymentCollections(ctx, filters, limit, offset)
}
func (s *Service) GetPaymentSession(ctx context.Context, id string) (*models.PaymentSession, error) {
	return s.repo.GetPaymentSession(ctx, id)
}
func (s *Service) GetPayment(ctx context.Context, id string) (*models.Payment, error) {
	return s.repo.GetPayment(ctx, id)
}
func (s *Service) ListPayments(ctx context.Context, f models.PaymentFilter) ([]models.Payment, int64, error) {
	return s.repo.ListPayments(ctx, f)
}
func (s *Service) ListPaymentProviders(ctx context.Context, f models.PaymentProviderFilter) ([]models.PaymentProvider, int64, error) {
	return s.repo.ListPaymentProviders(ctx, f)
}
func (s *Service) CreateCapture(ctx context.Context, input models.Capture) (*models.Capture, error) {
	return s.repo.CreateCapture(ctx, input)
}
func (s *Service) CreateRefund(ctx context.Context, input models.Refund) (*models.Refund, error) {
	return s.repo.CreateRefund(ctx, input)
}
