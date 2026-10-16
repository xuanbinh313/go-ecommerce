package models

import "time"

type Event struct {
	Name     string         `json:"name"`
	Data     any            `json:"data"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
}

type JobOptions struct {
	Attempts         int            `json:"attempts"`
	Priority         int            `json:"priority"`
	RemoveOnComplete any            `json:"removeOnComplete,omitempty"`
	Delay            time.Duration  `json:"delay,omitempty"`
	Extra            map[string]any `json:"extra,omitempty"`
}

type Job struct {
	Name                   string         `json:"name"`
	Data                   map[string]any `json:"data"`
	Opts                   JobOptions     `json:"opts"`
	CompletedSubscriberIDs []string       `json:"completedSubscriberIds,omitempty"`
}

type EmitOptions struct {
	JobOptions
	Internal         bool `json:"internal,omitempty"`
	GroupedEventsTTL int  `json:"groupedEventsTTL,omitempty"`
}
