package workflows

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
)

const (
	CreateShippingOptionsWorkflowID      = "create-shipping-options-workflow"
	ValidateFulfillmentProvidersStepID   = "validate-fulfillment-providers-step"
	ValidateShippingOptionPricesStepID   = "validate-shipping-option-prices"
	UpsertShippingOptionsStepID          = "create-shipping-options-step"
	CreateShippingOptionsPriceSetsStepID = "add-shipping-options-prices-step"
	SetShippingOptionsPriceSetsStepID    = "set-shipping-options-price-sets-step"
	EmitShippingOptionEventStepID        = "emit-event-step"
	ShippingOptionCreatedEvent           = "shipping-option.created"
	DeleteShippingOptionsWorkflowID      = "delete-shipping-options-workflow"
	DeleteShippingOptionsStepID          = "delete-shipping-options-step"
	ShippingOptionDeletedEvent           = "shipping-option.deleted"
)

type ShippingOptionCreator interface {
	CreateShippingOption(context.Context, models.ShippingOptionInput) (models.ShippingOption, error)
	HardDeleteShippingOption(context.Context, string) error
}
type ShippingOptionEventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}

type ShippingOptionDeleter interface {
	SoftDeleteShippingOptions(context.Context, []string) error
	RestoreShippingOptions(context.Context, []string) error
}

func DeleteShippingOptions(ctx context.Context, service ShippingOptionDeleter, events ShippingOptionEventPublisher, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := service.SoftDeleteShippingOptions(ctx, ids); err != nil {
		return err
	}
	eventData := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		eventData = append(eventData, map[string]any{"id": id})
	}
	var err error
	if events == nil {
		err = errors.New("shipping option event publisher is required")
	} else {
		err = events.Publish(ctx, ShippingOptionDeletedEvent, eventData)
	}
	if err != nil {
		if compensationErr := service.RestoreShippingOptions(ctx, ids); compensationErr != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", DeleteShippingOptionsStepID, compensationErr))
		}
		return err
	}
	return nil
}

// CreateShippingOption executes the source validation and persistence sequence.
// The repository transaction writes the option, rules, Price Set and link as one unit.
func CreateShippingOption(ctx context.Context, service ShippingOptionCreator, events ShippingOptionEventPublisher, input models.ShippingOptionInput) (models.ShippingOption, error) {
	created, err := service.CreateShippingOption(ctx, input)
	if err != nil {
		return models.ShippingOption{}, err
	}
	if events == nil {
		err = errors.New("shipping option event publisher is required")
	} else {
		err = events.Publish(ctx, ShippingOptionCreatedEvent, []map[string]any{{"id": created.ID}})
	}
	if err != nil {
		if compensationErr := service.HardDeleteShippingOption(ctx, created.ID); compensationErr != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", UpsertShippingOptionsStepID, compensationErr))
		}
		return models.ShippingOption{}, err
	}
	return created, nil
}
