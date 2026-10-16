package models

import "time"

type Event struct {
	Name     string         `json:"name"`
	Data     map[string]any `json:"data"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type Options struct {
	Internal bool          `json:"internal,omitempty"`
	Delay    time.Duration `json:"delay,omitempty"`
}
