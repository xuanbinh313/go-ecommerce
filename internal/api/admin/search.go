package admin

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	searchmodels "github.com/example/medusa-go-clone/internal/modules/search/domain/models"
	"github.com/gin-gonic/gin"
)

type GlobalSearchService interface {
	ListIndexes(context.Context) ([]searchmodels.SearchIndexDetails, error)
	SearchMany(context.Context, []searchmodels.SearchQuery) ([]searchmodels.SearchResult, error)
}

type SearchHTTP struct{ Service GlobalSearchService }

func (h *SearchHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	router.Group("/admin", auth).GET("/search", h.search)
}

func (h *SearchHTTP) search(c *gin.Context) {
	if !validSearchQueryKeys(c) {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "only q, entity, limit, and offset are supported"})
		return
	}
	if h.Service == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "Admin graph fallback search is not configured"})
		return
	}
	limit, err := parseSearchInt(c.Query("limit"), 20, 1, 100)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "limit must be an integer between 1 and 100"})
		return
	}
	offset, err := parseSearchInt(c.Query("offset"), 0, 0, int(^uint(0)>>1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "offset must be a nonnegative integer"})
		return
	}
	indexes, err := h.Service.ListIndexes(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"type": "database_error", "message": err.Error()})
		return
	}
	entities := parseSearchEntities(c)
	if len(entities) == 0 {
		for _, index := range indexes {
			entities = append(entities, index.Name)
		}
	}
	if len(entities) == 0 {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "Admin graph fallback search is not configured"})
		return
	}
	indexed := make(map[string]bool, len(indexes))
	for _, index := range indexes {
		indexed[index.Name] = true
	}
	queries := make([]searchmodels.SearchQuery, len(entities))
	for i, entity := range entities {
		if !indexed[entity] {
			c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": fmt.Sprintf("Admin graph fallback for search entity %q is not configured", entity)})
			return
		}
		take, skip := limit, offset
		filters := searchmodels.SearchFilters{}
		options := map[string]any{}
		if query := c.Query("q"); query != "" {
			filters["q"] = query
			options["match_strategy"] = "last"
		}
		queries[i] = searchmodels.SearchQuery{Entity: entity, Filters: filters, Pagination: searchmodels.Pagination{Skip: &skip, Take: &take}, SearchOptions: options}
	}
	results, err := h.Service.SearchMany(c, queries)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"type": "database_error", "message": err.Error()})
		return
	}
	groups := make([]gin.H, 0, len(results))
	for i, result := range results {
		count := int64(0)
		if result.Metadata.Count != nil {
			count = *result.Metadata.Count
		}
		data := make([]searchmodels.Document, len(result.Hits))
		for j, hit := range result.Hits {
			data[j] = hit.Document
		}
		groups = append(groups, gin.H{"entity": entities[i], "data": data, "count": count, "offset": result.Metadata.Skip, "limit": result.Metadata.Take})
	}
	c.JSON(http.StatusOK, gin.H{"results": groups})
}

func parseSearchEntities(c *gin.Context) []string {
	values := append([]string{}, c.QueryArray("entity")...)
	values = append(values, c.QueryArray("entity[]")...)
	entities := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		for _, entity := range strings.Split(value, ",") {
			entity = strings.TrimSpace(entity)
			if entity != "" && !seen[entity] {
				entities = append(entities, entity)
				seen[entity] = true
			}
		}
	}
	return entities
}

func parseSearchInt(value string, fallback, minimum, maximum int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum || parsed > maximum {
		return 0, fmt.Errorf("out of range")
	}
	return parsed, nil
}

func validSearchQueryKeys(c *gin.Context) bool {
	for key := range c.Request.URL.Query() {
		switch key {
		case "q", "entity", "entity[]", "limit", "offset":
		default:
			return false
		}
	}
	return true
}
