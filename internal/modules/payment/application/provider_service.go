package application

import (
	"context"

	"github.com/example/medusa-go-clone/internal/modules/payment/domain"
)

type ProviderLogger interface{ Warn(string, ...any) }

type PaymentMethodProvider interface {
	ListPaymentMethods(context.Context, map[string]any) ([]map[string]any, error)
	SavePaymentMethod(context.Context, map[string]any) (map[string]any, error)
	DeletePaymentMethod(context.Context, map[string]any) (map[string]any, error)
}
type AccountHolderProvider interface {
	RetrieveAccountHolder(context.Context, map[string]any) (map[string]any, error)
	CreateAccountHolder(context.Context, map[string]any) (map[string]any, error)
	UpdateAccountHolder(context.Context, map[string]any) (map[string]any, error)
	DeleteAccountHolder(context.Context, map[string]any) (map[string]any, error)
}

// ProviderService delegates each operation to the registered provider, matching
// Medusa's payment-provider service boundary. Optional capabilities return the
// source-compatible empty values when the provider doesn't implement them.
type ProviderService struct {
	registry domain.ProviderRegistry
	logger   ProviderLogger
}

func NewProviderService(registry domain.ProviderRegistry, logger ProviderLogger) *ProviderService {
	return &ProviderService{registry: registry, logger: logger}
}
func (s *ProviderService) provider(id string) (domain.Provider, error) {
	return s.registry.RetrieveProvider(id)
}
func (s *ProviderService) CreateSession(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	return p.CreateSession(ctx, input)
}
func (s *ProviderService) UpdateSession(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	return p.UpdateSession(ctx, input)
}
func (s *ProviderService) DeleteSession(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	return p.DeleteSession(ctx, input)
}
func (s *ProviderService) AuthorizePayment(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	return p.AuthorizePayment(ctx, input)
}
func (s *ProviderService) GetStatus(ctx context.Context, id string, input map[string]any) (string, error) {
	p, e := s.provider(id)
	if e != nil {
		return "", e
	}
	return p.GetStatus(ctx, input)
}
func (s *ProviderService) CapturePayment(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	return p.CapturePayment(ctx, input)
}
func (s *ProviderService) CancelPayment(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	return p.CancelPayment(ctx, input)
}
func (s *ProviderService) RefundPayment(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	return p.RefundPayment(ctx, input)
}
func (s *ProviderService) WebhookAction(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	return p.GetWebhookActionAndData(ctx, input)
}

func (s *ProviderService) RetrieveAccountHolder(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	optional, ok := p.(AccountHolderProvider)
	if !ok {
		s.warn(id, "retrieving account holders")
		return map[string]any{}, nil
	}
	return optional.RetrieveAccountHolder(ctx, input)
}
func (s *ProviderService) CreateAccountHolder(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	optional, ok := p.(AccountHolderProvider)
	if !ok {
		s.warn(id, "creating account holders")
		return map[string]any{}, nil
	}
	return optional.CreateAccountHolder(ctx, input)
}
func (s *ProviderService) UpdateAccountHolder(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	optional, ok := p.(AccountHolderProvider)
	if !ok {
		s.warn(id, "updating account holders")
		return map[string]any{}, nil
	}
	return optional.UpdateAccountHolder(ctx, input)
}
func (s *ProviderService) DeleteAccountHolder(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	optional, ok := p.(AccountHolderProvider)
	if !ok {
		s.warn(id, "deleting account holders")
		return map[string]any{}, nil
	}
	return optional.DeleteAccountHolder(ctx, input)
}
func (s *ProviderService) ListPaymentMethods(ctx context.Context, id string, input map[string]any) ([]map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	optional, ok := p.(PaymentMethodProvider)
	if !ok {
		s.warn(id, "listing payment methods")
		return []map[string]any{}, nil
	}
	return optional.ListPaymentMethods(ctx, input)
}
func (s *ProviderService) SavePaymentMethod(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	optional, ok := p.(PaymentMethodProvider)
	if !ok {
		s.warn(id, "saving payment methods")
		return map[string]any{}, nil
	}
	return optional.SavePaymentMethod(ctx, input)
}
func (s *ProviderService) DeletePaymentMethod(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	p, e := s.provider(id)
	if e != nil {
		return nil, e
	}
	optional, ok := p.(PaymentMethodProvider)
	if !ok {
		s.warn(id, "deleting payment methods")
		return map[string]any{}, nil
	}
	return optional.DeletePaymentMethod(ctx, input)
}
func (s *ProviderService) warn(id, capability string) {
	if s.logger != nil {
		s.logger.Warn("provider %s does not support %s", id, capability)
	}
}
