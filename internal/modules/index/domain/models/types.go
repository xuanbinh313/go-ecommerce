package models

import (
	"encoding/json"
	"time"
)

type MetadataStatus string

const (
	MetadataPending    MetadataStatus = "pending"
	MetadataProcessing MetadataStatus = "processing"
	MetadataDone       MetadataStatus = "done"
	MetadataError      MetadataStatus = "error"
)

type IndexData struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Data        json.RawMessage `json:"data"`
	DocumentTSV string          `json:"-"`
	StaledAt    *time.Time      `json:"staled_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty"`
}

type IndexRelation struct {
	ID         int64      `json:"id"`
	Pivot      string     `json:"pivot"`
	ParentName string     `json:"parent_name"`
	ParentID   string     `json:"parent_id"`
	ChildName  string     `json:"child_name"`
	ChildID    string     `json:"child_id"`
	StaledAt   *time.Time `json:"staled_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

type IndexSync struct {
	ID        string     `json:"id"`
	Entity    string     `json:"entity"`
	LastKey   *string    `json:"last_key,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type IndexMetadata struct {
	ID         string         `json:"id"`
	Entity     string         `json:"entity"`
	Fields     string         `json:"fields"`
	FieldsHash string         `json:"fields_hash"`
	Status     MetadataStatus `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  *time.Time     `json:"deleted_at,omitempty"`
}

type QueryConfig struct {
	Entity      string         `json:"entity"`
	Fields      []string       `json:"fields,omitempty"`
	Filters     map[string]any `json:"filters,omitempty"`
	JoinFilters map[string]any `json:"join_filters,omitempty"`
	IDsOnly     bool           `json:"ids_only,omitempty"`
	Pagination  Pagination     `json:"pagination,omitempty"`
}
type Pagination struct {
	Skip  *int           `json:"skip,omitempty"`
	Take  *int           `json:"take,omitempty"`
	Order map[string]any `json:"order,omitempty"`
}
type QueryResult struct {
	Data     []map[string]any `json:"data"`
	Metadata QueryMetadata    `json:"metadata"`
}
type QueryMetadata struct {
	Count *int64 `json:"count,omitempty"`
	Skip  *int   `json:"skip,omitempty"`
	Take  *int   `json:"take,omitempty"`
}

type RelationPartition struct {
	Pivot     string
	TableName string
}
type ParentProperty struct {
	Alias  string
	Entity string
}
type LinkRelationship struct {
	ForeignKey string
	Entity     string
}
type EntitySchema struct {
	Name              string
	Alias             string
	TableName         string
	IsLink            bool
	HasID             bool
	Fields            []string
	FieldsHash        string
	ParentProperties  []ParentProperty
	Relations         []RelationPartition
	LinkRelationships []LinkRelationship
}
type EntityChange struct {
	Action        string
	Entity        string
	Data          []map[string]any
	Schema        EntitySchema
	Relationships []LinkRelationship
}

type EntitySyncInput struct {
	Cursor    *string
	UpdatedAt *time.Time
	Limit     int
	BatchSize int
}
