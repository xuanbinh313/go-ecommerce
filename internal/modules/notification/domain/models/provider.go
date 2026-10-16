package models

type ProviderSendNotificationInput struct {
	To           string                   `json:"to"`
	From         string                   `json:"from,omitempty"`
	Channel      string                   `json:"channel"`
	Template     string                   `json:"template,omitempty"`
	Data         any                      `json:"data,omitempty"`
	Content      *NotificationContent     `json:"content,omitempty"`
	Attachments  []NotificationAttachment `json:"attachments,omitempty"`
	ProviderData map[string]any           `json:"provider_data,omitempty"`
}

type NotificationContent struct {
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type NotificationAttachment struct {
	Content     string `json:"content"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Disposition string `json:"disposition,omitempty"`
	ID          string `json:"id,omitempty"`
}
