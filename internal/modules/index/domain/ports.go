package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/index/domain/models"
	"time"
)

type Repository interface {
	UpsertData(context.Context, []models.IndexData) error
	UpsertEntity(context.Context, []models.IndexData, []models.IndexRelation) error
	DeleteData(context.Context, string, []string) error
	DeleteEntityRecords(context.Context, string, []string) error
	DeleteRelationsForEntityIDs(context.Context, string, []string) error
	DeleteEntity(context.Context, string, bool) error
	MarkEntityStale(context.Context, string) error
	UpsertRelations(context.Context, []models.IndexRelation) error
	DeleteRelations(context.Context, []models.IndexRelation) error
	GetSync(context.Context, string) (models.IndexSync, error)
	SaveSync(context.Context, models.IndexSync) error
	GetMetadata(context.Context, string) (models.IndexMetadata, error)
	ListMetadata(context.Context) ([]models.IndexMetadata, error)
	SaveMetadata(context.Context, models.IndexMetadata) error
	DeleteMetadata(context.Context, []string) error
	EnsureEntityPartition(context.Context, models.EntitySchema) error
}

type GraphQuery interface {
	QueryGraph(context.Context, string, map[string]any, []string, models.Pagination, bool) ([]map[string]any, error)
}
type StorageProvider interface {
	OnApplicationStart(context.Context) error
	Query(context.Context, models.QueryConfig) (models.QueryResult, error)
	ConsumeEntityEvent(context.Context, string, map[string]any) error
	Shutdown(context.Context) error
}
type EventBus interface {
	Subscribe(string, func(context.Context, map[string]any) error)
}

type Locker interface {
	Acquire(context.Context, string, time.Duration, string) error
	Release(context.Context, string, string) error
}
