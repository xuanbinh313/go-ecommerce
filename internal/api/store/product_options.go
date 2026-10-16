package store

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
	"github.com/gin-gonic/gin"
)

type StoreProductOptionService interface {
	ListProductOptions(context.Context, models.ProductOptionFilter, int, int) ([]models.ProductOption, int, error)
	GetProductOption(context.Context, string) (models.ProductOption, error)
}

type StoreProductOptionHTTP struct{ Service StoreProductOptionService }

var storeProductOptionDefaults = []string{"id", "title", "is_exclusive", "values", "created_at", "updated_at", "metadata"}

func (h *StoreProductOptionHTTP) Register(router *gin.RouterGroup) {
	router.GET("/store/product-options", h.list)
	router.GET("/store/product-options/:id", h.get)
}

func (h *StoreProductOptionHTTP) list(c *gin.Context) {
	limit, offset, ok := productCatalogPage(c)
	if !ok {
		return
	}
	filter, valid := storeProductOptionFilter(c)
	if !valid {
		return
	}
	items, count, err := h.Service.ListProductOptions(c, filter, limit, offset)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	fields, err := storeProductOptionFields(c.Query("fields"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, projectStoreProductOption(item, fields))
	}
	c.JSON(http.StatusOK, gin.H{"product_options": result, "count": count, "offset": offset, "limit": limit})
}

func (h *StoreProductOptionHTTP) get(c *gin.Context) {
	item, err := h.Service.GetProductOption(c, c.Param("id"))
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	fields, err := storeProductOptionFields(c.Query("fields"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_option": projectStoreProductOption(item, fields)})
}

func storeProductOptionFilter(c *gin.Context) (models.ProductOptionFilter, bool) {
	filter := models.ProductOptionFilter{IDs: queryValues(c, "id"), Titles: queryValues(c, "title"), Query: c.Query("q"), CreatedAt: collectionDateFilters(c, "created_at"), UpdatedAt: collectionDateFilters(c, "updated_at"), DeletedAt: collectionDateFilters(c, "deleted_at")}
	if value := c.Query("is_exclusive"); value != "" {
		parsed := strings.EqualFold(value, "true")
		if !parsed && !strings.EqualFold(value, "false") {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "is_exclusive must be true or false"})
			return filter, false
		}
		filter.IsExclusive = &parsed
	}
	for _, filters := range []map[string]string{filter.CreatedAt, filter.UpdatedAt, filter.DeletedAt} {
		for operator, value := range filters {
			if operator != "$eq" && operator != "$ne" && operator != "$gt" && operator != "$gte" && operator != "$lt" && operator != "$lte" {
				c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "unsupported timestamp filter operator: " + operator})
				return filter, false
			}
			if value == "null" && (operator == "$eq" || operator == "$ne") {
				continue
			}
			if _, err := time.Parse(time.RFC3339, value); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "timestamp filters must use RFC3339 values"})
				return filter, false
			}
		}
	}
	return filter, true
}

func storeProductOptionFields(requested string) ([]string, error) {
	if requested == "" {
		return append([]string(nil), storeProductOptionDefaults...), nil
	}
	fields := strings.Split(requested, ",")
	allowed := map[string]bool{"*": true, "id": true, "title": true, "is_exclusive": true, "values": true, "values.*": true, "values.id": true, "values.value": true, "created_at": true, "updated_at": true, "metadata": true}
	for _, field := range fields {
		field = strings.TrimSpace(strings.TrimLeft(field, "+-"))
		if !allowed[field] {
			return nil, errors.New("invalid product option field: " + field)
		}
	}
	return fields, nil
}

func projectStoreProductOption(item models.ProductOption, fields []string) map[string]any {
	encoded, _ := json.Marshal(item)
	var source map[string]any
	_ = json.Unmarshal(encoded, &source)
	out := map[string]any{}
	includeDefaults := false
	valueFields := []string{}
	allValues := false
	for _, field := range fields {
		if strings.HasPrefix(field, "+") || strings.HasPrefix(field, "-") {
			includeDefaults = true
		}
	}
	if includeDefaults {
		for _, field := range storeProductOptionDefaults {
			out[field] = source[field]
		}
	}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "*" {
			return source
		}
		if strings.HasPrefix(field, "-") {
			delete(out, strings.Split(strings.TrimPrefix(field, "-"), ".")[0])
			continue
		}
		field = strings.TrimPrefix(field, "+")
		if field == "values" || field == "values.*" {
			allValues = true
			continue
		}
		if strings.HasPrefix(field, "values.") {
			valueFields = append(valueFields, strings.TrimPrefix(field, "values."))
			continue
		}
		if value, ok := source[field]; ok {
			out[field] = value
		}
	}
	if allValues {
		out["values"] = source["values"]
	} else if len(valueFields) > 0 {
		values, _ := source["values"].([]any)
		projected := make([]any, 0, len(values))
		for _, value := range values {
			if row, ok := value.(map[string]any); ok {
				selected := map[string]any{}
				for _, field := range valueFields {
					if child, exists := row[field]; exists {
						selected[field] = child
					}
				}
				projected = append(projected, selected)
			}
		}
		out["values"] = projected
	}
	return out
}
