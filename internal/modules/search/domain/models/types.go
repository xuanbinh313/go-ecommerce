package models

import (
	"encoding/json"
	"time"
)

type IndexState string

const (
	IndexPending  IndexState = "pending"
	IndexBuilding IndexState = "building"
	IndexReady    IndexState = "ready"
	IndexError    IndexState = "error"
)

type SyncStatus string

const (
	SyncPending    SyncStatus = "pending"
	SyncProcessing SyncStatus = "processing"
	SyncDone       SyncStatus = "done"
	SyncFailed     SyncStatus = "failed"
	SyncCanceled   SyncStatus = "canceled"
)

type SearchIndex struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	ActiveVersion *int                 `json:"active_version"`
	Versions      []SearchIndexVersion `json:"versions,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	DeletedAt     *time.Time           `json:"deleted_at,omitempty"`
}

type SearchIndexVersion struct {
	ID             string            `json:"id"`
	Version        int               `json:"version"`
	Provider       string            `json:"provider"`
	PhysicalName   string            `json:"physical_name"`
	DefinitionHash string            `json:"definition_hash"`
	Status         IndexState        `json:"status"`
	SearchIndexID  string            `json:"search_index_id"`
	Syncs          []SearchIndexSync `json:"syncs,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	DeletedAt      *time.Time        `json:"deleted_at,omitempty"`
}

type SearchIndexSync struct {
	ID                   string          `json:"id"`
	JobID                *string         `json:"job_id,omitempty"`
	Status               SyncStatus      `json:"status"`
	Filters              json.RawMessage `json:"filters,omitempty"`
	LastKey              *string         `json:"last_key,omitempty"`
	Resumable            bool            `json:"resumable"`
	DocumentsSynced      int64           `json:"documents_synced"`
	StartedAt            *time.Time      `json:"started_at,omitempty"`
	CompletedAt          *time.Time      `json:"completed_at,omitempty"`
	Error                *string         `json:"error,omitempty"`
	SearchIndexVersionID string          `json:"search_index_version_id"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	DeletedAt            *time.Time      `json:"deleted_at,omitempty"`
}

type FieldDefinition struct {
	Type            string                     `json:"type,omitempty"`
	Array           bool                       `json:"array,omitempty"`
	Searchable      any                        `json:"searchable,omitempty"`
	Retrievable     *bool                      `json:"retrievable,omitempty"`
	Filterable      bool                       `json:"filterable,omitempty"`
	Sortable        bool                       `json:"sortable,omitempty"`
	Facetable       any                        `json:"facetable,omitempty"`
	Fields          map[string]FieldDefinition `json:"fields,omitempty"`
	Dimensions      int                        `json:"dimensions,omitempty"`
	Embed           bool                       `json:"embed,omitempty"`
	ProviderOptions map[string]map[string]any  `json:"provider_options,omitempty"`
}

type IndexDefinition struct {
	Name           string                     `json:"name"`
	Entity         string                     `json:"entity"`
	PrimaryKey     string                     `json:"primary_key"`
	Provider       string                     `json:"provider"`
	DefinitionHash string                     `json:"definition_hash"`
	PhysicalName   string                     `json:"physical_name"`
	Fields         map[string]FieldDefinition `json:"fields"`
	Settings       map[string]any             `json:"settings,omitempty"`
}

type SearchFilters map[string]any
type SearchQuery struct {
	Entity        string         `json:"entity"`
	Fields        []string       `json:"fields,omitempty"`
	Filters       SearchFilters  `json:"filters,omitempty"`
	Pagination    Pagination     `json:"pagination,omitempty"`
	SearchOptions map[string]any `json:"search_options,omitempty"`
}
type Pagination struct {
	Skip   *int              `json:"skip,omitempty"`
	Take   *int              `json:"take,omitempty"`
	Order  map[string]string `json:"order,omitempty"`
	Cursor string            `json:"cursor,omitempty"`
}
type ProviderSearchQuery struct {
	Index                IndexDefinition `json:"index"`
	AttributesToRetrieve []string        `json:"attributes_to_retrieve"`
	Filters              SearchFilters   `json:"filters,omitempty"`
	Pagination           Pagination      `json:"pagination"`
	SearchOptions        map[string]any  `json:"search_options,omitempty"`
	Q                    string          `json:"q,omitempty"`
}
type Document map[string]any
type SearchTask struct {
	ID     string            `json:"id,omitempty"`
	Index  string            `json:"index,omitempty"`
	Status string            `json:"status"`
	Error  map[string]string `json:"error,omitempty"`
}
type SearchHit struct {
	ID         string              `json:"id"`
	Score      *float64            `json:"score,omitempty"`
	Document   Document            `json:"document"`
	Highlights map[string][]string `json:"highlights,omitempty"`
}
type FacetResult map[string]any
type ResultMetadata struct {
	Skip             int      `json:"skip"`
	Take             int      `json:"take"`
	Count            *int64   `json:"count"`
	NextCursor       string   `json:"next_cursor,omitempty"`
	Query            string   `json:"query,omitempty"`
	ProcessingTimeMS *float64 `json:"processing_time_ms,omitempty"`
}
type SearchResult struct {
	Hits             []SearchHit            `json:"hits"`
	Facets           map[string]FacetResult `json:"facets,omitempty"`
	Metadata         ResultMetadata         `json:"metadata"`
	ProviderMetadata map[string]any         `json:"provider_metadata,omitempty"`
}
type SearchIndexInfo struct {
	Name          string     `json:"name"`
	Provider      string     `json:"provider"`
	DocumentCount int64      `json:"document_count"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type SearchIndexDetails struct {
	Name     string             `json:"name"`
	Entity   string             `json:"entity"`
	Provider string             `json:"provider"`
	Status   IndexState         `json:"status"`
	Fields   []SearchIndexField `json:"fields"`
}

type SearchIndexField struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Searchable bool   `json:"searchable"`
	Filterable bool   `json:"filterable"`
	Sortable   bool   `json:"sortable"`
	Facetable  bool   `json:"facetable"`
}

type SearchIndexDeleteResult struct {
	Index           string `json:"index"`
	DeletedVersions int    `json:"deleted_versions"`
}
