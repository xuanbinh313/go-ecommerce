package workflows

import (
	"context"
	"fmt"
	fulfillmentmodels "github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
)

// Workflow and step identifiers match Medusa's shipping option type flows.
const (
	CreateShippingOptionTypesWorkflowID = "create-shipping-option-types"
	CreateShippingOptionTypesStepID     = "create-shipping-option-types"
	UpdateShippingOptionTypesWorkflowID = "update-shipping-option-types"
	UpdateShippingOptionTypesStepID     = "update-shipping-option-types"
	DeleteShippingOptionTypesWorkflowID = "delete-shipping-option-types"
	DeleteShippingOptionTypesStepID     = "delete-shipping-option-types"
)

type ShippingOptionTypeService interface {
	CreateShippingOptionTypes(context.Context, []fulfillmentmodels.ShippingOptionTypeInput) ([]fulfillmentmodels.ShippingOptionType, error)
	UpdateShippingOptionType(context.Context, string, fulfillmentmodels.ShippingOptionTypeUpdate) (fulfillmentmodels.ShippingOptionType, error)
	GetShippingOptionType(context.Context, string) (fulfillmentmodels.ShippingOptionType, error)
	SoftDeleteShippingOptionTypes(context.Context, []string) error
	RestoreShippingOptionTypes(context.Context, []string) error
	HardDeleteShippingOptionTypes(context.Context, []string) error
	ShippingOptionTypeInUse(context.Context, string) (bool, error)
}

func CreateShippingOptionTypes(ctx context.Context, s ShippingOptionTypeService, in []fulfillmentmodels.ShippingOptionTypeInput) ([]fulfillmentmodels.ShippingOptionType, error) {
	v, e := s.CreateShippingOptionTypes(ctx, in)
	if e != nil {
		return nil, e
	}
	return v, nil
}
func UpdateShippingOptionTypes(ctx context.Context, s ShippingOptionTypeService, id string, in fulfillmentmodels.ShippingOptionTypeUpdate) (fulfillmentmodels.ShippingOptionType, error) {
	return s.UpdateShippingOptionType(ctx, id, in)
}
func DeleteShippingOptionTypes(ctx context.Context, s ShippingOptionTypeService, id string) error {
	used, e := s.ShippingOptionTypeInUse(ctx, id)
	if e != nil {
		return e
	}
	if used {
		return fmt.Errorf("cannot delete shipping option type because some shipping options are using it")
	}
	if e = s.SoftDeleteShippingOptionTypes(ctx, []string{id}); e != nil {
		return e
	}
	return nil
}
