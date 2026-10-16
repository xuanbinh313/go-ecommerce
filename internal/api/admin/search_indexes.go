package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	searchmodels "github.com/example/medusa-go-clone/internal/modules/search/domain/models"
	"github.com/gin-gonic/gin"
)

type SearchIndexService interface {
	ListIndexes(context.Context) ([]searchmodels.SearchIndexDetails, error)
	DeleteIndex(context.Context, string) (searchmodels.SearchIndexDeleteResult, error)
}
type SearchIndexReindexer interface {
	Reindex(context.Context, string, map[string]any, string, string) (string, error)
}

type SearchIndexHTTP struct{ Service SearchIndexService }

func (h *SearchIndexHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	a := router.Group("/admin", auth)
	a.GET("/search-indexes", h.list)
	a.DELETE("/search-indexes/:id", h.delete)
	// Reindex is exposed only when an implementation with the source background
	// job lifecycle is configured. No synchronous fake job is returned.
	a.POST("/search-indexes/:id/reindex", h.reindexUnavailable)
}

func (h *SearchIndexHTTP) list(c *gin.Context) {
	if h.Service == nil {
		c.JSON(http.StatusOK, gin.H{"search_indexes": []any{}, "enabled": false})
		return
	}
	indexes, err := h.Service.ListIndexes(c)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"search_indexes": indexes, "enabled": true})
}

func (h *SearchIndexHTTP) delete(c *gin.Context) {
	if h.Service == nil {
		c.JSON(http.StatusNotFound, gin.H{"type": "not_found", "message": "The Search Module is not enabled"})
		return
	}
	result, err := h.Service.DeleteIndex(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": result.Index, "object": "search_index", "deleted": true, "deleted_versions": result.DeletedVersions})
}

func (h *SearchIndexHTTP) reindexUnavailable(c *gin.Context) {
	var body map[string]json.RawMessage
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	for key := range body {
		if key != "since" && key != "filters" && key != "strategy" {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "unknown field: " + key})
			return
		}
	}
	var since, strategy string
	var filters map[string]any
	if raw := body["since"]; raw != nil {
		if err := json.Unmarshal(raw, &since); err != nil || since == "" {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "since must be an ISO 8601 datetime"})
			return
		}
		if _, err := time.Parse(time.RFC3339, since); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "since must be an ISO 8601 datetime"})
			return
		}
	}
	if raw := body["strategy"]; raw != nil {
		if err := json.Unmarshal(raw, &strategy); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "strategy must be swap or in_place"})
			return
		}
	}
	if raw := body["filters"]; raw != nil {
		if err := json.Unmarshal(raw, &filters); err != nil || filters == nil {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "filters must be an object"})
			return
		}
	}
	if strategy != "" && strategy != "swap" && strategy != "in_place" {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "strategy must be swap or in_place"})
		return
	}
	if h.Service == nil {
		c.JSON(http.StatusNotFound, gin.H{"type": "not_found", "message": "The Search Module is not enabled"})
		return
	}
	reindexer, ok := h.Service.(SearchIndexReindexer)
	if !ok {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "Search reindex background execution is not configured"})
		return
	}
	jobID, err := reindexer.Reindex(c, c.Param("id"), filters, since, strategy)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID, "indexes": []string{c.Param("id")}})
}
