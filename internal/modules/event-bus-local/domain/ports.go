package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/event-bus-local/domain/models"
)

type Subscriber func(context.Context, models.Event) error

type EventBus interface {
	Emit(context.Context, []models.Event, models.Options) error
	Subscribe(string, string, Subscriber)
	Unsubscribe(string, string)
	ReleaseGroupedEvents(context.Context, string) error
	ClearGroupedEvents(string, []string)
}
