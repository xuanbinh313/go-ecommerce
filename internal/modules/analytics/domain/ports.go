package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/analytics/domain/models"
)

type Provider interface {
	Track(context.Context, models.TrackEvent) error
	Identify(context.Context, models.IdentifyEvent) error
	Shutdown(context.Context) error
}
