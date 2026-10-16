package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/example/medusa-go-clone/internal/modules/index/domain"
	"github.com/example/medusa-go-clone/internal/modules/index/domain/models"
)

// ReconcileSchema compares the loaded graph index schema with persisted
// metadata, creates entity/pivot partitions, resets changed cursors, and removes
// records for entities that are no longer indexed.
func (s *Service) ReconcileSchema(ctx context.Context, schemas []models.EntitySchema) ([]models.EntitySchema, error) {
	current, err := s.repository.ListMetadata(ctx)
	if err != nil {
		return nil, err
	}
	byEntity := make(map[string]models.IndexMetadata, len(current))
	for _, row := range current {
		byEntity[row.Entity] = row
	}
	present := map[string]models.EntitySchema{}
	changed := make([]models.EntitySchema, 0, len(schemas))
	for _, schema := range schemas {
		if schema.Name == "" {
			return nil, fmt.Errorf("index entity name is required")
		}
		if _, exists := present[schema.Name]; exists {
			return nil, fmt.Errorf("index entity %q appears more than once in schema", schema.Name)
		}
		present[schema.Name] = schema
		if err := s.repository.EnsureEntityPartition(ctx, schema); err != nil {
			return nil, err
		}
		fields := append([]string(nil), schema.Fields...)
		sort.Strings(fields)
		fieldList := strings.Join(fields, ",")
		hash := simpleHash(fieldList)
		metadata, loadErr := s.repository.GetMetadata(ctx, schema.Name)
		if loadErr != nil && !errors.Is(loadErr, domain.ErrNotFound) {
			return nil, loadErr
		}
		isNew := errors.Is(loadErr, domain.ErrNotFound)
		isChanged := isNew || metadata.FieldsHash != hash
		if isChanged {
			metadata.Entity = schema.Name
			metadata.Fields = fieldList
			metadata.FieldsHash = hash
			metadata.Status = models.MetadataPending
			if err := s.repository.SaveMetadata(ctx, metadata); err != nil {
				return nil, err
			}
			if err := s.repository.SaveSync(ctx, models.IndexSync{Entity: schema.Name, LastKey: nil}); err != nil {
				return nil, err
			}
		}
		if isChanged || metadata.Status == models.MetadataPending || metadata.Status == models.MetadataProcessing || metadata.Status == models.MetadataError {
			changed = append(changed, schema)
		}
	}
	removed := []string{}
	for entity := range byEntity {
		if _, ok := present[entity]; !ok {
			removed = append(removed, entity)
		}
	}
	if len(removed) > 0 {
		if err := s.repository.DeleteMetadata(ctx, removed); err != nil {
			return nil, err
		}
		for _, entity := range removed {
			if err := s.repository.DeleteEntity(ctx, entity, false); err != nil {
				return nil, err
			}
		}
	}
	return changed, nil
}

func simpleHash(value string) string {
	hash := int64(5381)
	for _, unit := range utf16.Encode([]rune(value)) {
		hash = int64(int32(hash)<<5) + hash + int64(unit)
	}
	return strconv.FormatInt(hash, 16)
}
