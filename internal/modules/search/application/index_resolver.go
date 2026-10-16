package application

import (
	"context"
	"fmt"
	"sort"

	"github.com/example/medusa-go-clone/internal/modules/search/domain"
	"github.com/example/medusa-go-clone/internal/modules/search/domain/models"
)

// ConfiguredIndexResolver joins immutable process definitions to the active
// version records Medusa stores in search_index/search_index_version.
type ConfiguredIndexResolver struct {
	repository  domain.Repository
	definitions map[string]models.IndexDefinition
}

func NewConfiguredIndexResolver(repository domain.Repository, definitions []models.IndexDefinition) *ConfiguredIndexResolver {
	byName := make(map[string]models.IndexDefinition, len(definitions))
	for _, definition := range definitions {
		byName[definition.Name] = definition
	}
	return &ConfiguredIndexResolver{repository: repository, definitions: byName}
}

func (r *ConfiguredIndexResolver) GetDefinition(name string) (models.IndexDefinition, error) {
	definition, ok := r.definitions[name]
	if !ok {
		return models.IndexDefinition{}, fmt.Errorf("no search index named %q", name)
	}
	return definition, nil
}

func (r *ConfiguredIndexResolver) ListIndexes(ctx context.Context) ([]models.SearchIndexDetails, error) {
	definitions := make([]models.IndexDefinition, 0, len(r.definitions))
	for _, definition := range r.definitions {
		definitions = append(definitions, definition)
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Name < definitions[j].Name })
	records, err := r.repository.ListIndexes(ctx, nil)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]models.SearchIndex, len(records))
	ids := make([]string, 0, len(records))
	for _, record := range records {
		byName[record.Name] = record
		ids = append(ids, record.ID)
	}
	versions, err := r.repository.ListVersions(ctx, ids)
	if err != nil {
		return nil, err
	}
	versionsByIndex := map[string][]models.SearchIndexVersion{}
	for _, version := range versions {
		versionsByIndex[version.SearchIndexID] = append(versionsByIndex[version.SearchIndexID], version)
	}
	out := make([]models.SearchIndexDetails, 0, len(definitions))
	for _, definition := range definitions {
		item := models.SearchIndexDetails{Name: definition.Name, Entity: definition.Entity, Provider: definition.Provider, Status: models.IndexPending, Fields: indexedFields(definition.Fields, "")}
		if record, ok := byName[definition.Name]; ok && record.ActiveVersion != nil {
			for _, version := range versionsByIndex[record.ID] {
				if version.Version == *record.ActiveVersion {
					item.Provider, item.Status = version.Provider, version.Status
					break
				}
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func indexedFields(fields map[string]models.FieldDefinition, prefix string) []models.SearchIndexField {
	out := []models.SearchIndexField{}
	for name, field := range fields {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		if field.Type == "object" {
			out = append(out, indexedFields(field.Fields, path)...)
			continue
		}
		out = append(out, models.SearchIndexField{Name: path, Type: field.Type, Searchable: enabled(field.Searchable), Filterable: field.Filterable, Sortable: field.Sortable, Facetable: enabled(field.Facetable)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func enabled(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case map[string]any:
		return true
	default:
		return false
	}
}

func (r *ConfiguredIndexResolver) ResolveActiveDefinition(ctx context.Context, name string, fresh bool) (models.IndexDefinition, error) {
	definition, ok := r.definitions[name]
	if !ok {
		return models.IndexDefinition{}, fmt.Errorf("no search index named %q", name)
	}
	index, err := r.repository.GetIndex(ctx, name)
	if err != nil {
		return models.IndexDefinition{}, err
	}
	if index.ActiveVersion == nil {
		return models.IndexDefinition{}, fmt.Errorf("search index %q has no active version", name)
	}
	versions, err := r.repository.ListVersions(ctx, []string{index.ID})
	if err != nil {
		return models.IndexDefinition{}, err
	}
	for _, version := range versions {
		if version.SearchIndexID == index.ID && version.Version == *index.ActiveVersion && version.DeletedAt == nil {
			definition.Provider, definition.PhysicalName = version.Provider, version.PhysicalName
			return definition, nil
		}
	}
	return models.IndexDefinition{}, fmt.Errorf("active search index version %d for %q was not found", *index.ActiveVersion, name)
}
