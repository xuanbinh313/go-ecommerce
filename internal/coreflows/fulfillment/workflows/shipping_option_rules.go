package workflows

import (
	"context"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
)

const (
	BatchShippingOptionRulesWorkflowID = "batch-shipping-option-rules"
	CreateShippingOptionRulesStepID    = "create-shipping-option-rules"
	UpdateShippingOptionRulesStepID    = "update-shipping-option-rules"
	DeleteShippingOptionRulesStepID    = "delete-shipping-option-rules"
)

type ShippingOptionRuleBatcher interface {
	BatchShippingOptionRules(context.Context, string, models.ShippingOptionRuleBatchInput) (models.ShippingOptionRuleBatchResult, error)
}

// BatchShippingOptionRules applies the source create, update, and delete groups.
func BatchShippingOptionRules(ctx context.Context, service ShippingOptionRuleBatcher, optionID string, input models.ShippingOptionRuleBatchInput) (models.ShippingOptionRuleBatchResult, error) {
	return service.BatchShippingOptionRules(ctx, optionID, input)
}
