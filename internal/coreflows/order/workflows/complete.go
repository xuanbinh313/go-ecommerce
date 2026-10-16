package workflows

import (
	"context"

	eventmodels "github.com/example/medusa-go-clone/internal/modules/event-bus-local/domain/models"
	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
)

const (
	CompleteOrderWorkflowID = "complete-order-workflow"
	CompleteOrdersStepID    = "complete-orders"
	OrderCompletedEvent     = "order.completed"
)

type Completer interface {
	Complete(context.Context, []string) ([]models.Order, map[string]string, error)
	RestoreStatuses(context.Context, map[string]string) error
}

func CompleteOrders(ctx context.Context, service Completer, events EventPublisher, ids []string) ([]models.Order, error) {
	completed, previous, err := service.Complete(ctx, ids)
	if err != nil {
		return nil, err
	}
	groupID := entityid.New("eventgroup")
	eventList := make([]eventmodels.Event, 0, len(ids))
	for _, id := range ids {
		eventList = append(eventList, eventmodels.Event{Name: OrderCompletedEvent, Data: map[string]any{"id": id}, Metadata: map[string]any{"eventGroupId": groupID}})
	}
	if err = events.Emit(ctx, eventList, eventmodels.Options{}); err != nil {
		events.ClearGroupedEvents(groupID, []string{OrderCompletedEvent})
		if restoreErr := service.RestoreStatuses(ctx, previous); restoreErr != nil {
			return nil, restoreErr
		}
		return nil, err
	}
	if err = events.ReleaseGroupedEvents(ctx, groupID); err != nil {
		events.ClearGroupedEvents(groupID, []string{OrderCompletedEvent})
		if restoreErr := service.RestoreStatuses(ctx, previous); restoreErr != nil {
			return nil, restoreErr
		}
		return nil, err
	}
	return completed, nil
}
