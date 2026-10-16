package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/search/domain/models"
)

type SearchProvider interface {
	Identifier() string
	UpsertIndex(context.Context, models.IndexDefinition) (models.SearchTask, error)
	DeleteIndex(context.Context, string) (models.SearchTask, error)
	ListIndexes(context.Context) ([]models.SearchIndexInfo, error)
	UpsertDocuments(context.Context, string, models.IndexDefinition, []models.Document) (models.SearchTask, error)
	DeleteDocuments(context.Context, string, models.SearchFilters) (models.SearchTask, error)
	ClearIndex(context.Context, string) (models.SearchTask, error)
	Search(context.Context, models.ProviderSearchQuery) (models.SearchResult, error)
}

// OptionalSearchMany and OptionalTaskWait match optional Medusa provider capabilities.
type OptionalSearchMany interface {
	SearchMany(context.Context, []models.ProviderSearchQuery) ([]models.SearchResult, error)
}
type OptionalTaskWait interface {
	WaitForTask(context.Context, models.SearchTask, int) (models.SearchTask, error)
}

type ProviderRegistry interface {
	Retrieve(string) (SearchProvider, error)
	List() []SearchProvider
	GetDefaultIdentifier(string) (string, error)
}

type IndexResolver interface {
	ResolveActiveDefinition(context.Context, string, bool) (models.IndexDefinition, error)
}

type Repository interface {
	GetIndex(context.Context, string) (models.SearchIndex, error)
	ListIndexes(context.Context, []string) ([]models.SearchIndex, error)
	CreateIndex(context.Context, models.SearchIndex) (models.SearchIndex, error)
	UpdateActiveVersion(context.Context, string, *int) error
	CreateVersion(context.Context, models.SearchIndexVersion) (models.SearchIndexVersion, error)
	ListVersions(context.Context, []string) ([]models.SearchIndexVersion, error)
	CreateSync(context.Context, models.SearchIndexSync) (models.SearchIndexSync, error)
	UpdateSync(context.Context, models.SearchIndexSync) error
	ListSyncs(context.Context, []string) ([]models.SearchIndexSync, error)
	DeleteIndexRecords(context.Context, string) (int, error)
}
