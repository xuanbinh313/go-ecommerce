package local

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/example/medusa-go-clone/internal/modules/analytics/domain/models"
)

const Identifier = "analytics-local"

type Provider struct{ logger *slog.Logger }

func New(logger *slog.Logger) *Provider {
	if logger == nil {
		logger = slog.Default()
	}
	return &Provider{logger: logger}
}
func (*Provider) Identifier() string { return Identifier }
func (p *Provider) Track(_ context.Context, event models.TrackEvent) error {
	props, err := json.Marshal(event.Properties)
	if err != nil {
		return err
	}
	actor, group := event.ActorID, "-"
	if actor == "" {
		actor = "-"
	}
	if event.Group != nil && event.Group.ID != "" {
		group = event.Group.ID
	}
	p.logger.Debug(fmt.Sprintf("Tracking event: '%s', actor_id: '%s', group: '%s', properties: '%s'", event.Event, actor, group, string(props)))
	return nil
}
func (p *Provider) Identify(_ context.Context, event models.IdentifyEvent) error {
	props, err := json.Marshal(event.Properties)
	if err != nil {
		return err
	}
	actor, group := event.ActorID, "-"
	if actor == "" {
		actor = "-"
	}
	if event.Group != nil {
		group = event.Group.ID
	}
	p.logger.Debug(fmt.Sprintf("Identifying user: '%s', group: '%s', properties: '%s'", actor, group, string(props)))
	return nil
}
func (*Provider) Shutdown(context.Context) error { return nil }
