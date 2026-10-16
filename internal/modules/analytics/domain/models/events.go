package models

type TrackEvent struct {
	Event       string         `json:"event"`
	ActorID     string         `json:"actor_id,omitempty"`
	AnonymousID string         `json:"anonymous_id,omitempty"`
	UserID      string         `json:"user_id,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
	Context     map[string]any `json:"context,omitempty"`
	Group       *Group         `json:"group,omitempty"`
}
type IdentifyEvent struct {
	ActorID    string         `json:"actor_id,omitempty"`
	UserID     string         `json:"user_id,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	Group      *Group         `json:"group,omitempty"`
}
type Group struct {
	ID     string         `json:"id"`
	Type   string         `json:"type,omitempty"`
	Traits map[string]any `json:"traits,omitempty"`
}
