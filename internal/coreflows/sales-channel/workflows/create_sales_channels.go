package workflows

import (
	"context"
	"errors"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/sales-channel/domain/models"
)

const CreateSalesChannelsWorkflowID = "create-sales-channels"
const CreateSalesChannelsStepID = "create-sales-channels"
const EmitEventStepID = "emit-event-step"
const SalesChannelsCreatedEvent = "sales-channel.created"

type SalesChannelCreator interface {
	Create(context.Context, models.Input) (models.SalesChannel, error)
}
type SalesChannelDeleter interface {
	DeleteMany(context.Context, []string) error
}
type EventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}

func CreateSalesChannels(ctx context.Context, channels SalesChannelCreator, events EventPublisher, inputs []models.Input) ([]models.SalesChannel, error) {
	if events == nil {
		return nil, errors.New("sales channel event publisher is required")
	}
	created := make([]models.SalesChannel, 0, len(inputs))
	for _, input := range inputs {
		channel, err := channels.Create(ctx, input)
		if err != nil {
			return nil, compensateSalesChannels(ctx, channels, created, err)
		}
		created = append(created, channel)
	}
	eventData := make([]map[string]any, len(created))
	for i, channel := range created {
		eventData[i] = map[string]any{"id": channel.ID}
	}
	if err := events.Publish(ctx, SalesChannelsCreatedEvent, eventData); err != nil {
		return nil, compensateSalesChannels(ctx, channels, created, fmt.Errorf("emit %s: %w", SalesChannelsCreatedEvent, err))
	}
	return created, nil
}

func compensateSalesChannels(ctx context.Context, channels SalesChannelCreator, created []models.SalesChannel, cause error) error {
	deleter, ok := channels.(SalesChannelDeleter)
	if !ok || len(created) == 0 {
		return cause
	}
	ids := make([]string, len(created))
	for i, c := range created {
		ids[i] = c.ID
	}
	if err := deleter.DeleteMany(ctx, ids); err != nil {
		return errors.Join(cause, fmt.Errorf("delete created sales channels: %w", err))
	}
	return cause
}
