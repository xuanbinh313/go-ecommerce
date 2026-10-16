package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	searchmodels "github.com/example/medusa-go-clone/internal/modules/search/domain/models"
	"github.com/gin-gonic/gin"
)

type StoreSearchService interface {
	SearchMany(context.Context, []searchmodels.SearchQuery) ([]searchmodels.SearchResult, error)
	GetIndexDefinition(string) (searchmodels.IndexDefinition, error)
}

type PublishableKeyScopes interface {
	PublishableKeySalesChannels(context.Context, string) ([]string, error)
}

// StoreSearchIndexConfig is the JSON-compatible subset of configureStoreSearch.
// Medusa's callback-based filters must be implemented by Go callers separately.
type StoreSearchIndexConfig struct {
	Allowed bool                       `json:"-"`
	Filters searchmodels.SearchFilters `json:"filters,omitempty"`
}

func (config *StoreSearchIndexConfig) UnmarshalJSON(data []byte) error {
	switch string(bytes.TrimSpace(data)) {
	case "true":
		config.Allowed = true
		return nil
	case "false":
		return errors.New("store search indexes cannot be explicitly disabled in the allowlist")
	}
	type plain StoreSearchIndexConfig
	var parsed plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&parsed); err != nil {
		return err
	}
	*config = StoreSearchIndexConfig(parsed)
	config.Allowed = true
	return nil
}

type SearchHTTP struct {
	Service   StoreSearchService
	APIKeys   PublishableKeyScopes
	Allowlist map[string]StoreSearchIndexConfig
}

func (h *SearchHTTP) Register(router *gin.RouterGroup, authenticate gin.HandlerFunc) {
	router.POST("/store/search", authenticate, h.search)
}

func (h *SearchHTTP) search(c *gin.Context) {
	if h.Service == nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "not_allowed", "message": "The Search Module is not installed, so nothing can be searched."})
		return
	}
	queries, err := decodeStoreSearch(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	salesChannels := []string{}
	if h.APIKeys != nil {
		salesChannels, err = h.APIKeys.PublishableKeySalesChannels(c, c.GetHeader("x-publishable-api-key"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"type": "database_error", "message": err.Error()})
			return
		}
	}
	for i := range queries {
		if strings.TrimSpace(queries[i].Entity) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "entity is required"})
			return
		}
		access, ok := h.Allowlist[queries[i].Entity]
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"type": "not_found", "message": fmt.Sprintf("No search index named %q", queries[i].Entity)})
			return
		}
		definition, err := h.Service.GetIndexDefinition(queries[i].Entity)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"type": "not_found", "message": fmt.Sprintf("No search index named %q", queries[i].Entity)})
			return
		}
		if len(queries[i].Fields) > 0 {
			retrievable := map[string]bool{}
			collectSearchFields(definition.Fields, "", retrievable)
			for _, field := range queries[i].Fields {
				if !retrievable[field] {
					c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": fmt.Sprintf("Search index %q can't return the field: %s", queries[i].Entity, field)})
					return
				}
			}
		}
		productFilters := searchmodels.SearchFilters{}
		if definition.Entity == "product" {
			if searchFieldFilterable(definition.Fields, "status") {
				productFilters["status"] = "published"
			}
			if searchFieldFilterable(definition.Fields, "sales_channel_ids") {
				if len(salesChannels) == 0 {
					c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "Publishable key needs to have a sales channel configured"})
					return
				}
				productFilters["sales_channel_ids"] = salesChannels
			}
		}
		queries[i].Filters = mergeStoreSearchFilters(access.Filters, productFilters, queries[i].Filters)
		if err := validateStoreSearchPagination(queries[i].Pagination); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
			return
		}
	}
	results, err := h.Service.SearchMany(c, queries)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"type": "database_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

func decodeStoreSearch(body io.Reader) ([]searchmodels.SearchQuery, error) {
	data, err := io.ReadAll(io.LimitReader(body, 2<<20))
	if err != nil {
		return nil, err
	}
	var shape map[string]json.RawMessage
	if err := json.Unmarshal(data, &shape); err != nil {
		return nil, err
	}
	if _, ok := shape["queries"]; ok {
		var request struct {
			Queries []searchmodels.SearchQuery `json:"queries"`
		}
		if err := decodeStrict(data, &request); err != nil {
			return nil, err
		}
		if len(request.Queries) == 0 || len(request.Queries) > 20 {
			return nil, errors.New("queries must contain between 1 and 20 entries")
		}
		return request.Queries, nil
	}
	var query searchmodels.SearchQuery
	if err := decodeStrict(data, &query); err != nil {
		return nil, err
	}
	if strings.TrimSpace(query.Entity) == "" {
		return nil, errors.New("entity is required")
	}
	return []searchmodels.SearchQuery{query}, nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("request body must contain a single JSON value")
	}
	return nil
}

func collectSearchFields(fields map[string]searchmodels.FieldDefinition, prefix string, out map[string]bool) {
	for name, definition := range fields {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		if definition.Type == "object" {
			collectSearchFields(definition.Fields, path, out)
			continue
		}
		out[path] = definition.Retrievable == nil && definition.Type != "vector" || definition.Retrievable != nil && *definition.Retrievable
	}
}

func validateStoreSearchPagination(pagination searchmodels.Pagination) error {
	if pagination.Skip != nil && *pagination.Skip < 0 {
		return errors.New("pagination.skip must be nonnegative")
	}
	if pagination.Take != nil && (*pagination.Take < 0 || *pagination.Take > 100) {
		return errors.New("pagination.take must be between 0 and 100")
	}
	for _, direction := range pagination.Order {
		if direction != "ASC" && direction != "DESC" {
			return errors.New("pagination.order directions must be ASC or DESC")
		}
	}
	return nil
}

func searchFieldFilterable(fields map[string]searchmodels.FieldDefinition, name string) bool {
	field, ok := fields[name]
	return ok && field.Filterable
}

func cloneSearchFilters(filters searchmodels.SearchFilters) searchmodels.SearchFilters {
	cloned := searchmodels.SearchFilters{}
	for key, value := range filters {
		cloned[key] = value
	}
	return cloned
}

func mergeStoreSearchFilters(filters ...searchmodels.SearchFilters) searchmodels.SearchFilters {
	branches := []searchmodels.SearchFilters{}
	var query string
	for _, filter := range filters {
		if len(filter) == 0 {
			continue
		}
		branch := searchmodels.SearchFilters{}
		if value, ok := filter["q"].(string); ok {
			query = value
		}
		for key, value := range filter {
			if key != "q" {
				branch[key] = value
			}
		}
		if len(branch) > 0 {
			branches = append(branches, branch)
		}
	}
	merged := searchmodels.SearchFilters{}
	if query != "" {
		merged["q"] = query
	}
	if len(branches) == 1 {
		for key, value := range branches[0] {
			merged[key] = value
		}
	} else if len(branches) > 1 {
		and := make([]any, 0, len(branches))
		for _, branch := range branches {
			and = append(and, map[string]any(branch))
		}
		merged["$and"] = and
	}
	return merged
}
