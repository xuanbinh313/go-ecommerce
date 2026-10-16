package models

import (
	"encoding/json"
	"time"
)

type User struct {
	ID        string          `json:"id"`
	FirstName *string         `json:"first_name"`
	LastName  *string         `json:"last_name"`
	Email     string          `json:"email"`
	AvatarURL *string         `json:"avatar_url"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt *time.Time      `json:"deleted_at,omitempty"`
}

type Invite struct {
	ID        string          `json:"id"`
	Email     string          `json:"email"`
	Accepted  bool            `json:"accepted"`
	Token     string          `json:"token"`
	ExpiresAt time.Time       `json:"expires_at"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt *time.Time      `json:"deleted_at,omitempty"`
}

type UserInput struct {
	ID           *string         `json:"id,omitempty"`
	FirstName    *string         `json:"first_name,omitempty"`
	LastName     *string         `json:"last_name,omitempty"`
	Email        *string         `json:"email,omitempty"`
	AvatarURL    *string         `json:"avatar_url,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	FirstNameSet bool            `json:"-"`
	LastNameSet  bool            `json:"-"`
	AvatarURLSet bool            `json:"-"`
	MetadataSet  bool            `json:"-"`
}
type InviteInput struct {
	ID       *string         `json:"id,omitempty"`
	Email    string          `json:"email"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}
