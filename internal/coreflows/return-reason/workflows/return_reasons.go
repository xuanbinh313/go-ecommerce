package workflows

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
)

const (
	CreateReturnReasonsWorkflowID = "create-return-reasons"
	UpdateReturnReasonsWorkflowID = "update-return-reasons"
	DeleteReturnReasonsWorkflowID = "delete-return-reasons"
	CreateReturnReasonsStepID     = "create-return-reasons"
	UpdateReturnReasonsStepID     = "update-return-reasons"
	DeleteReturnReasonsStepID     = "delete-return-reasons"
)

type ReturnReasonService interface {
	ListReturnReasons(context.Context, models.ReturnReasonFilter, int, int) ([]models.ReturnReason, int, error)
	GetReturnReason(context.Context, string) (models.ReturnReason, error)
	CreateReturnReasons(context.Context, []models.ReturnReasonInput) ([]models.ReturnReason, error)
	UpdateReturnReasons(context.Context, []string, map[string]any) ([]models.ReturnReason, error)
	SoftDeleteReturnReasons(context.Context, []string) error
	RestoreReturnReasons(context.Context, []string) error
	HardDeleteReturnReasons(context.Context, []string) error
}

func CreateReturnReasons(ctx context.Context, s ReturnReasonService, in []models.ReturnReasonInput) ([]models.ReturnReason, error) {
	return s.CreateReturnReasons(ctx, in)
}
func UpdateReturnReasons(ctx context.Context, s ReturnReasonService, id string, fields map[string]any) ([]models.ReturnReason, error) {
	previous, err := s.GetReturnReason(ctx, id)
	if err != nil {
		return nil, err
	}
	updated, err := s.UpdateReturnReasons(ctx, []string{id}, fields)
	if err != nil {
		_, _ = s.UpdateReturnReasons(ctx, []string{id}, map[string]any{"value": previous.Value, "label": previous.Label, "description": previous.Description, "metadata": previous.Metadata})
		return nil, err
	}
	return updated, nil
}
func DeleteReturnReasons(ctx context.Context, s ReturnReasonService, ids []string) error {
	return s.SoftDeleteReturnReasons(ctx, ids)
}
