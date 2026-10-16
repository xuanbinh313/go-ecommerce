package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/payment/domain/models"
)

type Repository interface {
	ListRefundReasons(context.Context, models.RefundReasonFilter) ([]models.RefundReason, int64, error)
	GetRefundReason(context.Context, string) (*models.RefundReason, error)
	CreateRefundReasons(context.Context, []models.RefundReasonInput) ([]models.RefundReason, error)
	UpdateRefundReasons(context.Context, []models.RefundReason) ([]models.RefundReason, error)
	SoftDeleteRefundReasons(context.Context, []string) error
	RestoreRefundReasons(context.Context, []string) error
	DeleteRefundReasons(context.Context, []string) error
	CreatePaymentCollection(context.Context, models.CreatePaymentCollectionInput) (*models.PaymentCollection, error)
	DeletePaymentCollections(context.Context, []string) error
	GetPaymentCollection(context.Context, string) (*models.PaymentCollection, error)
	ListPaymentCollections(context.Context, map[string]any, int, int) ([]models.PaymentCollection, int64, error)
	GetPaymentSession(context.Context, string) (*models.PaymentSession, error)
	GetPayment(context.Context, string) (*models.Payment, error)
	ListPayments(context.Context, models.PaymentFilter) ([]models.Payment, int64, error)
	ListPaymentProviders(context.Context, models.PaymentProviderFilter) ([]models.PaymentProvider, int64, error)
	CreateCapture(context.Context, models.Capture) (*models.Capture, error)
	CreateRefund(context.Context, models.Refund) (*models.Refund, error)
}

// Provider preserves Medusa's payment-provider operations without hard-wiring
// the Payment module to a concrete gateway.
type Provider interface {
	CreateSession(context.Context, map[string]any) (map[string]any, error)
	UpdateSession(context.Context, map[string]any) (map[string]any, error)
	DeleteSession(context.Context, map[string]any) (map[string]any, error)
	AuthorizePayment(context.Context, map[string]any) (map[string]any, error)
	GetStatus(context.Context, map[string]any) (string, error)
	CapturePayment(context.Context, map[string]any) (map[string]any, error)
	CancelPayment(context.Context, map[string]any) (map[string]any, error)
	RefundPayment(context.Context, map[string]any) (map[string]any, error)
	GetWebhookActionAndData(context.Context, map[string]any) (map[string]any, error)
}

type ProviderRegistry interface {
	RetrieveProvider(string) (Provider, error)
}
