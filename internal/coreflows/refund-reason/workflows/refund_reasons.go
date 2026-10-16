package workflows

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/payment/domain/models"
)

const (
	CreateRefundReasonsWorkflowID = "create-refund-reasons-workflow"
	UpdateRefundReasonsWorkflowID = "update-refund-reasons"
	DeleteRefundReasonsWorkflowID = "delete-refund-reasons-workflow"
	CreateRefundReasonStepID      = "create-refund-reason"
	UpdateRefundReasonsStepID     = "update-refund-reasons"
	DeleteRefundReasonsStepID     = "delete-refund-reasons"
)

type RefundReasonService interface {
	ListRefundReasons(context.Context, models.RefundReasonFilter) ([]models.RefundReason, int64, error)
	GetRefundReason(context.Context, string) (*models.RefundReason, error)
	CreateRefundReasons(context.Context, []models.RefundReasonInput) ([]models.RefundReason, error)
	UpdateRefundReasons(context.Context, []models.RefundReason) ([]models.RefundReason, error)
	SoftDeleteRefundReasons(context.Context, []string) error
	RestoreRefundReasons(context.Context, []string) error
	DeleteRefundReasons(context.Context, []string) error
}

func CreateRefundReasons(ctx context.Context, s RefundReasonService, in []models.RefundReasonInput) ([]models.RefundReason, error) {
	return s.CreateRefundReasons(ctx, in)
}
func UpdateRefundReasons(ctx context.Context, s RefundReasonService, in []models.RefundReason) ([]models.RefundReason, error) {
	previous := make([]models.RefundReason, 0, len(in))
	for _, v := range in {
		old, err := s.GetRefundReason(ctx, v.ID)
		if err != nil {
			return nil, err
		}
		previous = append(previous, *old)
	}
	updated, err := s.UpdateRefundReasons(ctx, in)
	if err != nil {
		_, _ = s.UpdateRefundReasons(ctx, previous)
		return nil, err
	}
	return updated, nil
}
func DeleteRefundReasons(ctx context.Context, s RefundReasonService, ids []string) error {
	return s.SoftDeleteRefundReasons(ctx, ids)
}
