package workflows

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
)

const (
	CreateFulfillmentWorkflowID = "create-fulfillment-workflow"
	CreateFulfillmentStepID     = "create-fulfillment"
	CancelFulfillmentWorkflowID = "cancel-fulfillment-workflow"
	CancelFulfillmentStepID     = "cancel-fulfillment"
	CreateShipmentWorkflowID    = "create-shipment-workflow"
	ValidateShipmentStepID      = "validate-shipment"
	UpdateFulfillmentWorkflowID = "update-fulfillment-workflow"
	UpdateFulfillmentStepID     = "update-fulfillment"
	FulfillmentCreatedEvent     = "fulfillment.created"
	FulfillmentUpdatedEvent     = "fulfillment.updated"
)

type FulfillmentMutations interface {
	CreateFulfillmentRecord(context.Context, models.FulfillmentCreateInput) (models.FulfillmentRecord, error)
	GetFulfillment(context.Context, string) (models.FulfillmentRecord, error)
	CancelFulfillmentRecord(context.Context, string) (models.FulfillmentRecord, error)
	CreateFulfillmentShipment(context.Context, string, models.CreateShipmentInput) (models.FulfillmentRecord, models.FulfillmentRecord, error)
	RestoreFulfillment(context.Context, models.FulfillmentRecord) error
	HardDeleteFulfillment(context.Context, string) error
}

type FulfillmentEventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}

func CreateFulfillment(ctx context.Context, service FulfillmentMutations, events FulfillmentEventPublisher, input models.FulfillmentCreateInput) (models.FulfillmentRecord, error) {
	created, err := service.CreateFulfillmentRecord(ctx, input)
	if err != nil {
		return models.FulfillmentRecord{}, err
	}
	if events == nil {
		err = errors.New("fulfillment event publisher is required")
	} else {
		err = events.Publish(ctx, FulfillmentCreatedEvent, []map[string]any{{"id": created.ID}})
	}
	if err != nil {
		if created.ProviderID != nil {
			_, _ = service.CancelFulfillmentRecord(ctx, created.ID)
		}
		if compensationErr := service.HardDeleteFulfillment(ctx, created.ID); compensationErr != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", CreateFulfillmentStepID, compensationErr))
		}
		return models.FulfillmentRecord{}, err
	}
	return created, nil
}

func CancelFulfillment(ctx context.Context, service FulfillmentMutations, events FulfillmentEventPublisher, id string) (models.FulfillmentRecord, error) {
	updated, err := service.CancelFulfillmentRecord(ctx, id)
	if err != nil {
		return models.FulfillmentRecord{}, err
	}
	if events != nil {
		if err = events.Publish(ctx, FulfillmentUpdatedEvent, []map[string]any{{"id": id}}); err != nil {
			return models.FulfillmentRecord{}, err
		}
	}
	return updated, nil
}

func CreateFulfillmentShipment(ctx context.Context, service FulfillmentMutations, events FulfillmentEventPublisher, id string, input models.CreateShipmentInput) (models.FulfillmentRecord, error) {
	previous, updated, err := service.CreateFulfillmentShipment(ctx, id, input)
	if err != nil {
		return models.FulfillmentRecord{}, err
	}
	if events == nil {
		err = errors.New("fulfillment event publisher is required")
	} else {
		err = events.Publish(ctx, FulfillmentUpdatedEvent, []map[string]any{{"id": id}})
	}
	if err != nil {
		if compensationErr := service.RestoreFulfillment(ctx, previous); compensationErr != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", UpdateFulfillmentStepID, compensationErr))
		}
		return models.FulfillmentRecord{}, err
	}
	return updated, nil
}
