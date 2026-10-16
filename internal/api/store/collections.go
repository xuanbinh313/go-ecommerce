package store

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
	"github.com/gin-gonic/gin"
)

type CollectionService interface {
	ListProductCollections(ctx context.Context, filter models.ProductCollectionFilter, limit, offset int) ([]models.ProductCollection, int, error)
	GetProductCollection(ctx context.Context, id string) (models.ProductCollection, error)
	ListProductIDsForCollection(ctx context.Context, id string) ([]string, error)
	GetProduct(ctx context.Context, id string) (models.Product, error)
}

type CollectionHTTP struct{ Service CollectionService }

var storeCollectionFields = []string{"id", "title", "handle", "external_id", "created_at", "updated_at"}

func (h *CollectionHTTP) Register(router *gin.RouterGroup) {
	router.GET("/store/collections", h.list)
	router.GET("/store/collections/:id", h.get)
}

func (h *CollectionHTTP) list(c *gin.Context) {
	limit, err := positiveQueryInt(c, "limit", 10)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "limit must be a positive integer"})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "offset must be a non-negative integer"})
		return
	}
	filter := models.ProductCollectionFilter{
		IDs: queryValues(c, "id"), Titles: queryValues(c, "title"), Handles: queryValues(c, "handle"),
		ExternalIDs: queryValues(c, "external_id"), Query: c.Query("q"), CreatedAt: collectionDateFilters(c, "created_at"),
		UpdatedAt: collectionDateFilters(c, "updated_at"), Order: c.DefaultQuery("order", "-created_at"),
	}
	orderField := strings.TrimPrefix(filter.Order, "-")
	switch orderField {
	case "created_at", "updated_at", "title", "handle", "external_id", "id":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "unsupported collection order field: " + orderField})
		return
	}
	for _, dateFilters := range []map[string]string{filter.CreatedAt, filter.UpdatedAt} {
		for operator, value := range dateFilters {
			if operator != "$eq" && operator != "$ne" && operator != "$gt" && operator != "$gte" && operator != "$lt" && operator != "$lte" {
				c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "unsupported timestamp filter operator: " + operator})
				return
			}
			if _, err := time.Parse(time.RFC3339, value); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "timestamp filters must use RFC3339 values"})
				return
			}
		}
	}
	rows, count, err := h.Service.ListProductCollections(c, filter, limit, offset)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	collections := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		projected, projectErr := h.project(c, row)
		if projectErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": projectErr.Error()})
			return
		}
		collections = append(collections, projected)
	}
	c.JSON(http.StatusOK, gin.H{"collections": collections, "count": count, "offset": offset, "limit": limit})
}

func (h *CollectionHTTP) get(c *gin.Context) {
	row, err := h.Service.GetProductCollection(c, c.Param("id"))
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	projected, err := h.project(c, row)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"collection": projected})
}

func (h *CollectionHTTP) project(c *gin.Context, row models.ProductCollection) (map[string]any, error) {
	fields, err := collectionFields(c.Query("fields"))
	if err != nil {
		return nil, err
	}
	encoded, _ := json.Marshal(row)
	var source map[string]any
	_ = json.Unmarshal(encoded, &source)
	if includesCollectionProducts(fields) {
		ids, err := h.Service.ListProductIDsForCollection(c, row.ID)
		if err != nil {
			return nil, err
		}
		products := make([]any, 0, len(ids))
		for _, id := range ids {
			product, err := h.Service.GetProduct(c, id)
			if err != nil {
				continue
			}
			if product.Status != "published" {
				continue
			}
			productJSON, _ := json.Marshal(product)
			var productMap map[string]any
			_ = json.Unmarshal(productJSON, &productMap)
			products = append(products, projectProductFields(productMap, fields))
		}
		source["products"] = products
	}
	return projectCollectionMap(source, fields), nil
}

func collectionFields(requested string) ([]string, error) {
	if requested == "" {
		return append([]string(nil), storeCollectionFields...), nil
	}
	fields := strings.Split(requested, ",")
	allowed := map[string]bool{"*": true, "id": true, "title": true, "handle": true, "external_id": true, "created_at": true, "updated_at": true, "metadata": true, "deleted_at": true, "products": true, "*products": true, "products.*": true}
	for _, name := range []string{"id", "title", "subtitle", "description", "handle", "is_giftcard", "discountable", "thumbnail", "collection_id", "type_id", "weight", "length", "height", "width", "hs_code", "origin_country", "mid_code", "material", "created_at", "updated_at", "metadata", "status", "external_id", "deleted_at", "options", "*options", "options.id", "options.title", "options.is_exclusive", "options.values", "options.values.id", "options.values.value", "*options.values", "variants", "*variants", "variants.id", "variants.title", "variants.sku", "variants.barcode", "variants.ean", "variants.upc", "variants.thumbnail", "variants.manage_inventory", "variants.allow_backorder", "variants.weight", "variants.length", "variants.height", "variants.width", "variants.hs_code", "variants.origin_country", "variants.mid_code", "variants.material", "variants.created_at", "variants.updated_at"} {
		allowed["products."+name] = true
	}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if strings.HasPrefix(field, "+") || strings.HasPrefix(field, "-") {
			field = field[1:]
		}
		if !allowed[field] {
			return nil, errors.New("invalid collection field: " + field)
		}
	}
	return fields, nil
}

func includesCollectionProducts(fields []string) bool {
	for _, field := range fields {
		field = strings.TrimLeft(strings.TrimSpace(field), "+-")
		if field == "*" || field == "products" || field == "*products" || strings.HasPrefix(field, "products.") {
			return true
		}
	}
	return false
}

func projectCollectionMap(source map[string]any, fields []string) map[string]any {
	all := false
	includeDefault := false
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "*" {
			all = true
		}
		if strings.HasPrefix(field, "+") || strings.HasPrefix(field, "-") {
			includeDefault = true
		}
	}
	if all {
		return source
	}
	out := map[string]any{}
	if includeDefault {
		for _, field := range storeCollectionFields {
			if value, ok := source[field]; ok {
				out[field] = value
			}
		}
	}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "*" {
			continue
		}
		if strings.HasPrefix(field, "-") {
			delete(out, strings.Split(strings.TrimPrefix(field, "-"), ".")[0])
			continue
		}
		field = strings.TrimPrefix(field, "+")
		if field == "*products" {
			field = "products"
		}
		if field == "products" {
			if products, ok := source["products"].([]any); ok {
				projected := make([]any, 0, len(products))
				for _, value := range products {
					if item, ok := value.(map[string]any); ok {
						projected = append(projected, projectProductFields(item, nil))
					}
				}
				out["products"] = projected
			}
			continue
		}
		if strings.HasPrefix(field, "products.") {
			if products, ok := source["products"].([]any); ok {
				projected := make([]any, 0, len(products))
				childField := strings.TrimPrefix(field, "products.")
				for _, value := range products {
					if item, ok := value.(map[string]any); ok {
						projected = append(projected, projectProductFields(item, []string{childField}))
					}
				}
				out["products"] = projected
			}
			continue
		}
		if value, ok := source[field]; ok {
			out[field] = value
		}
	}
	return out
}

func projectProductFields(source map[string]any, fields []string) map[string]any {
	all := false
	requested := []string{}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		field = strings.TrimPrefix(field, "products.")
		field = strings.TrimPrefix(field, "*")
		if field == "*" || field == "*products" {
			all = true
		} else if field != "" {
			requested = append(requested, field)
		}
	}
	if all {
		return source
	}
	if len(requested) == 0 {
		requested = []string{"id", "title", "subtitle", "description", "handle", "is_giftcard", "discountable", "thumbnail", "collection_id", "type_id", "weight", "length", "height", "width", "hs_code", "origin_country", "mid_code", "material", "created_at", "updated_at", "options.id", "options.title", "options.is_exclusive", "options.values.id", "options.values.value", "variants.id", "variants.title", "variants.sku", "variants.barcode", "variants.ean", "variants.upc", "variants.thumbnail", "variants.manage_inventory", "variants.allow_backorder", "variants.weight", "variants.length", "variants.height", "variants.width", "variants.hs_code", "variants.origin_country", "variants.mid_code", "variants.material", "variants.created_at", "variants.updated_at"}
	}
	out := map[string]any{}
	for _, path := range requested {
		copyCollectionField(out, source, path)
	}
	return out
}

func copyCollectionField(dst, src map[string]any, path string) {
	parts := strings.SplitN(path, ".", 2)
	value, ok := src[parts[0]]
	if !ok {
		return
	}
	if len(parts) == 1 {
		dst[parts[0]] = value
		return
	}
	childPath := parts[1]
	switch child := value.(type) {
	case map[string]any:
		target, _ := dst[parts[0]].(map[string]any)
		if target == nil {
			target = map[string]any{}
			dst[parts[0]] = target
		}
		copyCollectionField(target, child, childPath)
	case []any:
		target := make([]any, 0, len(child))
		for _, item := range child {
			if itemMap, ok := item.(map[string]any); ok {
				projected := map[string]any{}
				copyCollectionField(projected, itemMap, childPath)
				target = append(target, projected)
			}
		}
		dst[parts[0]] = target
	}
}

func writeCollectionError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	kind := "server_error"
	if strings.Contains(strings.ToLower(err.Error()), "no rows") || strings.Contains(strings.ToLower(err.Error()), "not found") {
		status = http.StatusNotFound
		kind = "not_found"
	}
	c.JSON(status, gin.H{"type": kind, "message": err.Error()})
}

func queryValues(c *gin.Context, key string) []string {
	values := c.QueryArray(key)
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if value := strings.TrimSpace(part); value != "" {
				out = append(out, value)
			}
		}
	}
	return out
}

func positiveQueryInt(c *gin.Context, key string, fallback int) (int, error) {
	if c.Query(key) == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(c.Query(key))
	if err != nil || value <= 0 {
		return 0, strconv.ErrSyntax
	}
	return value, nil
}

func collectionDateFilters(c *gin.Context, field string) map[string]string {
	filters := map[string]string{}
	for key, values := range c.Request.URL.Query() {
		prefix := field + "["
		if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, "]") || len(values) == 0 {
			continue
		}
		filters[strings.TrimSuffix(strings.TrimPrefix(key, prefix), "]")] = values[0]
	}
	return filters
}
