package models

import (
	"encoding/json"
	"time"
)

type Role struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty"`
	Policies    []Policy        `json:"policies,omitempty"`
}
type Policy struct {
	ID                  string          `json:"id"`
	Key                 string          `json:"key"`
	Resource            string          `json:"resource"`
	Operation           string          `json:"operation"`
	Name                *string         `json:"name"`
	Description         *string         `json:"description"`
	Metadata            json.RawMessage `json:"metadata"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	DeletedAt           *time.Time      `json:"deleted_at,omitempty"`
	InheritedFromRoleID *string         `json:"inherited_from_role_id,omitempty"`
}
type RolePolicy struct {
	ID        string          `json:"id"`
	RoleID    string          `json:"role_id"`
	PolicyID  string          `json:"policy_id"`
	Policy    *Policy         `json:"policy,omitempty"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt *time.Time      `json:"deleted_at,omitempty"`
}
type RoleParent struct {
	ID        string          `json:"id"`
	RoleID    string          `json:"role_id"`
	ParentID  string          `json:"parent_id"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
type PermissionAction struct {
	Resource  string `json:"resource"`
	Operation string `json:"operation"`
}
type RoleInput struct {
	ID          *string         `json:"id,omitempty"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
}
type PolicyInput struct {
	ID          *string         `json:"id,omitempty"`
	Key         string          `json:"key,omitempty"`
	Resource    string          `json:"resource"`
	Operation   string          `json:"operation"`
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
}
type RoleUpdate struct {
	Name           *string         `json:"name"`
	Description    *string         `json:"description"`
	DescriptionSet bool            `json:"-"`
	Metadata       json.RawMessage `json:"metadata"`
	MetadataSet    bool            `json:"-"`
}
type PolicyUpdate struct {
	Key            *string         `json:"key"`
	Resource       *string         `json:"resource"`
	Operation      *string         `json:"operation"`
	Name           *string         `json:"name"`
	NameSet        bool            `json:"-"`
	Description    *string         `json:"description"`
	DescriptionSet bool            `json:"-"`
	Metadata       json.RawMessage `json:"metadata"`
	MetadataSet    bool            `json:"-"`
}
type RolePolicyInput struct {
	ID       *string         `json:"id,omitempty"`
	RoleID   string          `json:"role_id"`
	PolicyID string          `json:"policy_id"`
	Metadata json.RawMessage `json:"metadata"`
}
type RoleParentInput struct {
	ID       *string         `json:"id,omitempty"`
	RoleID   string          `json:"role_id"`
	ParentID string          `json:"parent_id"`
	Metadata json.RawMessage `json:"metadata"`
}
