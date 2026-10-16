package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/search/domain"
	"github.com/example/medusa-go-clone/internal/modules/search/domain/models"
)

type Service struct {
	providers  domain.ProviderRegistry
	indexes    domain.IndexResolver
	repository domain.Repository
}

func NewService(providers domain.ProviderRegistry, indexes domain.IndexResolver, repository domain.Repository) *Service {
	return &Service{providers: providers, indexes: indexes, repository: repository}
}

func (s *Service) Search(ctx context.Context, query models.SearchQuery) (models.SearchResult, error) {
	results, err := s.SearchMany(ctx, []models.SearchQuery{query})
	if err != nil {
		return models.SearchResult{}, err
	}
	return results[0], nil
}

func (s *Service) ListIndexes(ctx context.Context) ([]models.SearchIndexDetails, error) {
	if definitions, ok := s.indexes.(interface {
		ListIndexes(context.Context) ([]models.SearchIndexDetails, error)
	}); ok {
		return definitions.ListIndexes(ctx)
	}
	return nil, fmt.Errorf("search index resolver does not support listing indexes")
}
func (s *Service) SearchMany(ctx context.Context, queries []models.SearchQuery) ([]models.SearchResult, error) {
	if len(queries) == 0 {
		return []models.SearchResult{}, nil
	}
	results := make([]models.SearchResult, len(queries))
	// Group by provider, retaining the source query order when joining responses.
	type pending struct {
		index int
		query models.ProviderSearchQuery
	}
	groups := map[string][]pending{}
	providers := map[string]domain.SearchProvider{}
	for i, query := range queries {
		definition, err := s.indexes.ResolveActiveDefinition(ctx, query.Entity, false)
		if err != nil {
			return nil, err
		}
		provider, err := s.providers.Retrieve(definition.Provider)
		if err != nil {
			return nil, err
		}
		if query.Pagination.Take == nil {
			take := 20
			query.Pagination.Take = &take
		}
		if query.Pagination.Skip == nil {
			skip := 0
			query.Pagination.Skip = &skip
		}
		if *query.Pagination.Skip < 0 || *query.Pagination.Take < 0 {
			return nil, fmt.Errorf("search pagination values cannot be negative")
		}
		fields := query.Fields
		if fields == nil {
			fields = retrievableFields(definition.Fields, "")
		}
		filters := make(models.SearchFilters, len(query.Filters))
		for key, value := range query.Filters {
			if key != "q" {
				filters[key] = value
			}
		}
		q, _ := query.Filters["q"].(string)
		if len(filters) == 0 {
			filters = nil
		}
		searchOptions := normalizeSearchOptions(query.SearchOptions, definition.Fields)
		groups[definition.Provider] = append(groups[definition.Provider], pending{i, models.ProviderSearchQuery{Index: definition, AttributesToRetrieve: fields, Filters: filters, Pagination: query.Pagination, SearchOptions: searchOptions, Q: q}})
		providers[definition.Provider] = provider
	}
	for id, group := range groups {
		provider := providers[id]
		inputs := make([]models.ProviderSearchQuery, len(group))
		for i, item := range group {
			inputs[i] = item.query
		}
		var batch []models.SearchResult
		if many, ok := provider.(domain.OptionalSearchMany); ok {
			var err error
			batch, err = many.SearchMany(ctx, inputs)
			if err != nil {
				return nil, err
			}
		}
		if batch == nil {
			batch = make([]models.SearchResult, len(inputs))
			for i, input := range inputs {
				result, err := provider.Search(ctx, input)
				if err != nil {
					return nil, err
				}
				batch[i] = result
			}
		}
		if len(batch) != len(group) {
			return nil, fmt.Errorf("search provider %q returned %d results for %d queries", id, len(batch), len(group))
		}
		for i, item := range group {
			results[item.index] = batch[i]
		}
	}
	return results, nil
}

func retrievableFields(fields map[string]models.FieldDefinition, prefix string) []string {
	out := []string{}
	for name, field := range fields {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		if field.Type == "object" {
			out = append(out, retrievableFields(field.Fields, path)...)
			continue
		}
		if field.Type == "vector" {
			if field.Retrievable != nil && *field.Retrievable {
				out = append(out, path)
			}
			continue
		}
		if field.Retrievable == nil || *field.Retrievable {
			out = append(out, path)
		}
	}
	return out
}

func normalizeSearchOptions(options map[string]any, fields map[string]models.FieldDefinition) map[string]any {
	if options == nil {
		return nil
	}
	copy := make(map[string]any, len(options))
	for key, value := range options {
		copy[key] = value
	}
	if facets, ok := options["facets"].([]any); ok {
		normalized := make([]any, 0, len(facets))
		for _, facet := range facets {
			if field, ok := facet.(string); ok {
				normalized = append(normalized, map[string]any{"field": field, "type": "value"})
			} else {
				normalized = append(normalized, facet)
			}
		}
		copy["facets"] = normalized
	}
	if vector, ok := options["vector"].(map[string]any); ok {
		if _, hasField := vector["field"]; !hasField {
			vectorPaths := []string{}
			collectVectorFields(fields, "", &vectorPaths)
			if len(vectorPaths) == 1 {
				updated := make(map[string]any, len(vector)+1)
				for key, value := range vector {
					updated[key] = value
				}
				updated["field"] = vectorPaths[0]
				copy["vector"] = updated
			}
		}
	}
	return copy
}

func collectVectorFields(fields map[string]models.FieldDefinition, prefix string, into *[]string) {
	for name, field := range fields {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		if field.Type == "vector" {
			*into = append(*into, path)
		} else if field.Type == "object" {
			collectVectorFields(field.Fields, path, into)
		}
	}
}

func (s *Service) UpsertDocuments(ctx context.Context, index string, documents []models.Document) (models.SearchTask, error) {
	if len(documents) == 0 {
		return models.SearchTask{Index: index, Status: "succeeded"}, nil
	}
	definition, err := s.indexes.ResolveActiveDefinition(ctx, index, true)
	if err != nil {
		return models.SearchTask{}, err
	}
	provider, err := s.providers.Retrieve(definition.Provider)
	if err != nil {
		return models.SearchTask{}, err
	}
	task, err := provider.UpsertDocuments(ctx, definition.PhysicalName, definition, documents)
	if err != nil {
		return models.SearchTask{}, err
	}
	return acceptedTask(index, task)
}
func (s *Service) DeleteDocuments(ctx context.Context, index string, filters models.SearchFilters) (models.SearchTask, error) {
	if len(filters) == 0 {
		return models.SearchTask{}, fmt.Errorf("deleting from search index %q requires filters; pass {id: [...]} or clear the index through a reindex", index)
	}
	definition, err := s.indexes.ResolveActiveDefinition(ctx, index, true)
	if err != nil {
		return models.SearchTask{}, err
	}
	provider, err := s.providers.Retrieve(definition.Provider)
	if err != nil {
		return models.SearchTask{}, err
	}
	task, err := provider.DeleteDocuments(ctx, definition.PhysicalName, filters)
	if err != nil {
		return models.SearchTask{}, err
	}
	return acceptedTask(index, task)
}

func (s *Service) DeleteIndex(ctx context.Context, index string) (models.SearchIndexDeleteResult, error) {
	if definitions, ok := s.indexes.(interface {
		GetDefinition(string) (models.IndexDefinition, error)
	}); ok {
		if _, err := definitions.GetDefinition(index); err != nil {
			return models.SearchIndexDeleteResult{}, err
		}
	}
	record, err := s.repository.GetIndex(ctx, index)
	if err != nil {
		// Medusa treats an index with no bookkeeping row as an idempotent delete.
		if errors.Is(err, domain.ErrNotFound) {
			return models.SearchIndexDeleteResult{Index: index}, nil
		}
		return models.SearchIndexDeleteResult{}, err
	}
	versions, err := s.repository.ListVersions(ctx, []string{record.ID})
	if err != nil {
		return models.SearchIndexDeleteResult{}, err
	}
	for _, version := range versions {
		provider, providerErr := s.providers.Retrieve(version.Provider)
		if providerErr == nil {
			_, _ = provider.DeleteIndex(ctx, version.PhysicalName)
		}
	}
	deleted, err := s.repository.DeleteIndexRecords(ctx, index)
	if err != nil {
		return models.SearchIndexDeleteResult{}, err
	}
	return models.SearchIndexDeleteResult{Index: index, DeletedVersions: deleted}, nil
}
func acceptedTask(index string, task models.SearchTask) (models.SearchTask, error) {
	if task.Status == "failed" {
		return models.SearchTask{}, fmt.Errorf("search provider rejected task for index %q: %v", index, task.Error)
	}
	return task, nil
}
