package domain

import (
	"context"

	"github.com/example/medusa-go-clone/internal/modules/notification/domain/models"
)

type Repository interface {
	List(context.Context, models.NotificationFilter, int, int, string) ([]models.Notification, int, error)
	Get(context.Context, string) (models.Notification, error)
}

type Provider interface {
	Identifier() string
	Send(context.Context, *models.ProviderSendNotificationInput) (map[string]any, error)
}
