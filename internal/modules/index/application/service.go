package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/index/domain"
	"github.com/example/medusa-go-clone/internal/modules/index/domain/models"
)

type Service struct {
	repository domain.Repository
	storage    domain.StorageProvider
	graph      domain.GraphQuery
	locker     domain.Locker
}

func NewService(repository domain.Repository, storage domain.StorageProvider) *Service {
	return &Service{repository: repository, storage: storage}
}
func NewSynchronizingService(repository domain.Repository, storage domain.StorageProvider, graph domain.GraphQuery, locker domain.Locker) *Service {
	return &Service{repository: repository, storage: storage, graph: graph, locker: locker}
}
func (s *Service) Query(ctx context.Context, input models.QueryConfig) (models.QueryResult, error) {
	if s.storage == nil {
		return models.QueryResult{}, fmt.Errorf("index storage provider is not configured")
	}
	return s.storage.Query(ctx, input)
}
func (s *Service) OnApplicationStart(ctx context.Context) error {
	if s.storage != nil {
		return s.storage.OnApplicationStart(ctx)
	}
	return nil
}
func (s *Service) Shutdown(ctx context.Context) error {
	if s.storage != nil {
		return s.storage.Shutdown(ctx)
	}
	return nil
}

// SyncEntities runs entity synchronization in order, under the same per-entity
// lock and status/checkpoint protocol as Medusa's synchronizer.
func (s *Service) SyncEntities(ctx context.Context, entities []models.EntitySchema) error {
	for _, schema := range entities {
		if schema.Name == "" {
			return fmt.Errorf("index entity name is required")
		}
		if s.locker != nil {
			if err := s.locker.Acquire(ctx, schema.Name, time.Minute, fmt.Sprintf("index-sync-%d", os.Getpid())); err != nil {
				continue
			}
		}
		err := s.syncEntity(ctx, schema, models.EntitySyncInput{})
		if s.locker != nil {
			_ = s.locker.Release(ctx, schema.Name, fmt.Sprintf("index-sync-%d", os.Getpid()))
		}
		if err != nil {
			var failure syncFailure
			if errors.As(err, &failure) {
				continue
			}
			return err
		}
	}
	return nil
}

func (s *Service) SyncEntity(ctx context.Context, schema models.EntitySchema, input models.EntitySyncInput) error {
	return s.syncEntity(ctx, schema, input)
}

func (s *Service) syncEntity(ctx context.Context, schema models.EntitySchema, input models.EntitySyncInput) error {
	if schema.Alias == "" {
		return nil
	}
	metadata, err := s.repository.GetMetadata(ctx, schema.Name)
	if err != nil && err != domain.ErrNotFound {
		return err
	}
	metadata.Entity = schema.Name
	metadata.Fields = strings.Join(schema.Fields, ",")
	metadata.FieldsHash = schema.FieldsHash
	metadata.Status = models.MetadataProcessing
	if err = s.repository.SaveMetadata(ctx, metadata); err != nil {
		return err
	}
	if err = s.repository.MarkEntityStale(ctx, schema.Name); err != nil {
		return err
	}
	if !schema.HasID {
		return s.failSync(ctx, schema, fmt.Errorf("Entity %s does not have a property 'id'. The 'id' must be provided and must be orderable (e.g ulid)", schema.Name))
	}
	if s.graph == nil {
		return s.failSync(ctx, schema, fmt.Errorf("index graph query is not configured"))
	}
	syncRecord, err := s.repository.GetSync(ctx, schema.Name)
	if err != nil && err != domain.ErrNotFound {
		return err
	}
	if input.Cursor != nil {
		syncRecord.LastKey = input.Cursor
	}
	cursor := syncRecord.LastKey
	batchSize := input.BatchSize
	if batchSize <= 0 || batchSize > 100 {
		batchSize = 100
	}
	limit := input.Limit
	if limit <= 0 {
		limit = math.MaxInt
	}
	processed := 0
	for processed < limit {
		filters := map[string]any{}
		if cursor != nil && *cursor != "" {
			filters["id"] = map[string]any{"$gt": *cursor}
		}
		if input.UpdatedAt != nil {
			filters["updated_at"] = map[string]any{"$gt": input.UpdatedAt}
		}
		take := batchSize
		if remaining := limit - processed; remaining < take {
			take = remaining
		}
		page, queryErr := s.graph.QueryGraph(ctx, schema.Alias, filters, append([]string{"id"}, schema.Fields...), models.Pagination{Take: &take, Order: map[string]any{"id": "ASC"}}, false)
		if queryErr != nil {
			return s.failSync(ctx, schema, queryErr)
		}
		if len(page) == 0 {
			break
		}
		action := "created"
		if schema.IsLink {
			action = "attached"
		}
		change := models.EntityChange{Action: action, Entity: schema.Name, Data: page, Schema: schema, Relationships: schema.LinkRelationships}
		if consumeErr := s.ConsumeEntityEvent(ctx, change); consumeErr != nil {
			return s.failSync(ctx, schema, consumeErr)
		}
		last, ok := stringValue(page[len(page)-1]["id"])
		if !ok {
			return s.failSync(ctx, schema, fmt.Errorf("index sync %s result missing id", schema.Name))
		}
		cursor = &last
		syncRecord.Entity = schema.Name
		syncRecord.LastKey = cursor
		if err = s.repository.SaveSync(ctx, syncRecord); err != nil {
			return s.failSync(ctx, schema, err)
		}
		processed += len(page)
		if s.locker != nil {
			_ = s.locker.Acquire(ctx, schema.Name, time.Minute, fmt.Sprintf("index-sync-%d", os.Getpid()))
		}
	}
	metadata.Status = models.MetadataDone
	if err = s.repository.SaveMetadata(ctx, metadata); err != nil {
		return err
	}
	syncRecord.Entity = schema.Name
	syncRecord.LastKey = cursor
	if err = s.repository.SaveSync(ctx, syncRecord); err != nil {
		return err
	}
	return s.repository.DeleteEntity(ctx, schema.Name, true)
}

func (s *Service) failSync(ctx context.Context, schema models.EntitySchema, cause error) error {
	metadata, err := s.repository.GetMetadata(ctx, schema.Name)
	if err == nil {
		metadata.Status = models.MetadataError
		_ = s.repository.SaveMetadata(ctx, metadata)
	}
	return syncFailure{cause: cause}
}

type syncFailure struct{ cause error }

func (e syncFailure) Error() string { return e.cause.Error() }
func (e syncFailure) Unwrap() error { return e.cause }

// ConsumeEntityEvent mirrors the Postgres provider's create/update/delete and
// link attach/detach persistence actions. Graph projection and event subscription
// are host responsibilities, matching the source module's injected boundaries.
func (s *Service) ConsumeEntityEvent(ctx context.Context, event models.EntityChange) error {
	if event.Entity == "" {
		return fmt.Errorf("index event entity is required")
	}
	if len(event.Data) == 0 {
		return nil
	}
	switch event.Action {
	case "created", "restored", "updated":
		rows, relations, err := projectRows(event)
		if err != nil {
			return err
		}
		return s.repository.UpsertEntity(ctx, rows, relations)
	case "deleted", "detached":
		ids := make([]string, 0, len(event.Data))
		for _, row := range event.Data {
			id, ok := stringValue(row["id"])
			if !ok {
				return fmt.Errorf("index event %s record is missing id", event.Entity)
			}
			ids = append(ids, id)
		}
		return s.repository.DeleteEntityRecords(ctx, event.Entity, ids)
	case "attached":
		rows, _, err := projectRows(event)
		if err != nil {
			return err
		}
		relations, err := projectLinkRelations(event)
		if err != nil {
			return err
		}
		return s.repository.UpsertEntity(ctx, rows, relations)
	default:
		return fmt.Errorf("index module does not handle event action %q", event.Action)
	}
}

func projectRows(event models.EntityChange) ([]models.IndexData, []models.IndexRelation, error) {
	properties := map[string]bool{"id": true}
	parents := map[string]string{}
	for _, field := range event.Schema.Fields {
		if field != "" && !strings.Contains(field, ".") {
			properties[field] = true
		}
	}
	for _, parent := range event.Schema.ParentProperties {
		parents[parent.Alias] = parent.Entity
	}
	rows := make([]models.IndexData, 0, len(event.Data))
	relations := []models.IndexRelation{}
	for _, item := range event.Data {
		id, ok := stringValue(item["id"])
		if !ok {
			return nil, nil, fmt.Errorf("index event %s record is missing id", event.Entity)
		}
		clean := map[string]any{}
		for field := range properties {
			if value, exists := item[field]; exists {
				clean[field] = value
			}
		}
		encoded, err := json.Marshal(clean)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, models.IndexData{ID: id, Name: event.Entity, Data: encoded})
		for alias, parentEntity := range parents {
			parentValue, exists := item[alias]
			if !exists || parentValue == nil {
				continue
			}
			for _, parent := range asObjects(parentValue) {
				parentID, ok := stringValue(parent["id"])
				if !ok {
					continue
				}
				relations = append(relations, models.IndexRelation{Pivot: parentEntity + "-" + event.Entity, ParentName: parentEntity, ParentID: parentID, ChildName: event.Entity, ChildID: id})
			}
		}
	}
	return rows, relations, nil
}

func projectLinkRelations(event models.EntityChange) ([]models.IndexRelation, error) {
	if len(event.Relationships) < 2 {
		return nil, fmt.Errorf("attached index link %s requires parent and child relationships", event.Entity)
	}
	parent, child := event.Relationships[0], event.Relationships[1]
	if parent.ForeignKey == "" || child.ForeignKey == "" || parent.Entity == "" || child.Entity == "" {
		return nil, fmt.Errorf("attached index link %s has incomplete relationship metadata", event.Entity)
	}
	out := make([]models.IndexRelation, 0, len(event.Data)*2)
	for _, row := range event.Data {
		id, ok := stringValue(row["id"])
		if !ok {
			return nil, fmt.Errorf("attached index link %s is missing id", event.Entity)
		}
		parentID, ok := stringValue(row[parent.ForeignKey])
		if !ok {
			return nil, fmt.Errorf("attached index link %s is missing %s", event.Entity, parent.ForeignKey)
		}
		childID, ok := stringValue(row[child.ForeignKey])
		if !ok {
			return nil, fmt.Errorf("attached index link %s is missing %s", event.Entity, child.ForeignKey)
		}
		out = append(out, models.IndexRelation{Pivot: parent.Entity + "-" + event.Entity, ParentName: parent.Entity, ParentID: parentID, ChildName: event.Entity, ChildID: id}, models.IndexRelation{Pivot: event.Entity + "-" + child.Entity, ParentName: event.Entity, ParentID: id, ChildName: child.Entity, ChildID: childID})
	}
	return out, nil
}

func asObjects(value any) []map[string]any {
	switch v := value.(type) {
	case map[string]any:
		return []map[string]any{v}
	case []map[string]any:
		return v
	case []any:
		out := []map[string]any{}
		for _, item := range v {
			if object, ok := item.(map[string]any); ok {
				out = append(out, object)
			}
		}
		return out
	default:
		return nil
	}
}
func stringValue(value any) (string, bool) {
	if value == nil {
		return "", false
	}
	switch v := value.(type) {
	case string:
		return v, v != ""
	case fmt.Stringer:
		return v.String(), true
	default:
		return fmt.Sprint(v), fmt.Sprint(v) != ""
	}
}
