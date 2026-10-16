package workflows

import (
	"context"

	eventmodels "github.com/example/medusa-go-clone/internal/modules/event-bus-local/domain/models"
	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
)

const (
	ArchiveOrderWorkflowID = "archive-order-workflow"
	ArchiveOrdersStepID    = "archive-orders"
	OrderArchivedEvent     = "order.archived"
)

type Archiver interface {
	Archive(context.Context, []string) ([]models.Order, error)
}
type EventPublisher interface {
	Emit(context.Context, []eventmodels.Event, eventmodels.Options) error
	ReleaseGroupedEvents(context.Context, string) error
	ClearGroupedEvents(string, []string)
}

func ArchiveOrder(ctx context.Context, service Archiver, events EventPublisher, id string) (models.Order, error) {
	groupID := entityid.New("eventgroup")
	if err := events.Emit(ctx, []eventmodels.Event{{Name: OrderArchivedEvent, Data: map[string]any{"id": id}, Metadata: map[string]any{"eventGroupId": groupID}}}, eventmodels.Options{}); err != nil {
		return models.Order{}, err
	}
	orders, err := service.Archive(ctx, []string{id})
	if err != nil {
		events.ClearGroupedEvents(groupID, []string{OrderArchivedEvent})
		return models.Order{}, err
	}
	if err := events.ReleaseGroupedEvents(ctx, groupID); err != nil {
		return models.Order{}, err
	}
	if len(orders) == 0 {
		return models.Order{}, nil
	}
	return orders[0], nil
}
