package models

import "time"

type APIKey struct {
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	Token         string         `json:"token"`
	Redacted      string         `json:"redacted"`
	Type          string         `json:"type"`
	LastUsedAt    *time.Time     `json:"last_used_at"`
	CreatedBy     string         `json:"created_by"`
	RevokedBy     *string        `json:"revoked_by"`
	RevokedAt     *time.Time     `json:"revoked_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	SalesChannels []SalesChannel `json:"sales_channels,omitempty"`
	Salt          string         `json:"-"`
}

type SalesChannel struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}
type CreateInput struct {
	Title     string `json:"title"`
	Type      string `json:"type"`
	CreatedBy string `json:"created_by"`
}
type UpdateInput struct {
	Title string `json:"title"`
}
type RevokeInput struct {
	RevokeIn  *int64 `json:"revoke_in"`
	RevokedBy string `json:"revoked_by"`
}
type Filter struct {
	IDs    []string
	Titles []string
	Types  []string
	Query  string
	Limit  int
	Offset int
}
type Created struct {
	APIKey   APIKey
	RawToken string
}
