package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/example/medusa-go-clone/internal/modules/notification/domain/models"
)

const Identifier = "notification-local"

type Provider struct{ logger *slog.Logger }

func New(logger *slog.Logger) *Provider {
	if logger == nil {
		logger = slog.Default()
	}
	return &Provider{logger: logger}
}
func (*Provider) Identifier() string { return Identifier }
func (p *Provider) Send(_ context.Context, notification *models.ProviderSendNotificationInput) (map[string]any, error) {
	if notification == nil {
		return nil, errors.New("No notification information provided")
	}
	data, err := json.Marshal(notification.Data)
	if err != nil {
		return nil, fmt.Errorf("serialize notification data: %w", err)
	}
	message := fmt.Sprintf("Attempting to send a notification to: '%s' on the channel: '%s' with template: '%s' and data: '%s'", notification.To, notification.Channel, notification.Template, string(data))
	p.logger.Info(message)
	return map[string]any{}, nil
}
