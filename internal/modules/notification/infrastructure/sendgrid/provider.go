package sendgrid

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/notification/domain/models"
)

const Identifier = "notification-sendgrid"
const endpoint = "https://api.sendgrid.com/v3/mail/send"

type Provider struct {
	apiKey string
	from   string
	client *http.Client
}

func New(apiKey, from string, client *http.Client) *Provider {
	if client == nil {
		client = http.DefaultClient
	}
	return &Provider{apiKey: apiKey, from: from, client: client}
}
func (*Provider) Identifier() string { return Identifier }

func (p *Provider) Send(ctx context.Context, notification *models.ProviderSendNotificationInput) (map[string]any, error) {
	if notification == nil {
		return nil, errors.New("No notification information provided")
	}
	if strings.TrimSpace(p.apiKey) == "" {
		return nil, errors.New("SendGrid api_key is required")
	}
	from := strings.TrimSpace(notification.From)
	if from == "" {
		from = p.from
	}
	message := map[string]any{"from": map[string]string{"email": from}}
	if notification.Data != nil {
		message["dynamic_template_data"] = notification.Data
	}
	if len(notification.Attachments) > 0 {
		attachments := make([]map[string]any, 0, len(notification.Attachments))
		for _, a := range notification.Attachments {
			disposition := a.Disposition
			if disposition == "" {
				disposition = "attachment"
			}
			item := map[string]any{"content": a.Content, "filename": a.Filename, "content_type": a.ContentType, "disposition": disposition}
			if a.ID != "" {
				item["id"] = a.ID
			}
			attachments = append(attachments, item)
		}
		message["attachments"] = attachments
	}
	if personalizations, ok := notification.ProviderData["personalizations"].([]any); ok && len(personalizations) > 0 {
		message["personalizations"] = personalizations
	} else if personalizations, ok := notification.ProviderData["personalizations"].([]map[string]any); ok && len(personalizations) > 0 {
		message["personalizations"] = personalizations
	} else {
		message["to"] = []map[string]string{{"email": notification.To}}
	}
	if notification.Content != nil {
		message["subject"] = notification.Content.Subject
		message["content"] = []map[string]string{{"type": "text/html", "value": notification.Content.HTML}}
	} else {
		message["template_id"] = notification.Template
	}
	body, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("encode SendGrid message: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to send email: request - %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		var payload struct {
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		_ = json.Unmarshal(responseBody, &payload)
		detail := "unknown error"
		if len(payload.Errors) > 0 && payload.Errors[0].Message != "" {
			detail = payload.Errors[0].Message
		}
		return nil, fmt.Errorf("Failed to send email: %d - %s", resp.StatusCode, detail)
	}
	return map[string]any{}, nil
}
