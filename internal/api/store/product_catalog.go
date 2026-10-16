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

type ProductCatalogService interface {
	ListStoreProductTags(context.Context, models.ProductCatalogFilter, int, int) ([]models.ProductTag, int, error)
	GetProductTag(context.Context, string) (models.ProductTag, error)
	ListProductIDsForTag(context.Context, string) ([]string, error)
	ListStoreProductTypes(context.Context, models.ProductCatalogFilter, int, int) ([]models.ProductType, int, error)
	GetProductType(context.Context, string) (models.ProductType, error)
	ListProductIDsForType(context.Context, string) ([]string, error)
	GetProduct(context.Context, string) (models.Product, error)
}

type ProductCatalogHTTP struct{ Service ProductCatalogService }

func (h *ProductCatalogHTTP) Register(router *gin.RouterGroup) {
	router.GET("/store/product-tags", h.listTags)
	router.GET("/store/product-tags/:id", h.getTag)
	router.GET("/store/product-types", h.listTypes)
	router.GET("/store/product-types/:id", h.getType)
}

func (h *ProductCatalogHTTP) listTags(c *gin.Context) {
	limit, offset, ok := productCatalogPage(c)
	if !ok {
		return
	}
	filter := productCatalogFilter(c)
	if !validateProductCatalogFilters(c, filter) {
		return
	}
	items, count, err := h.Service.ListStoreProductTags(c, filter, limit, offset)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	output := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row, projectErr := h.projectCatalog(c, item, h.Service.ListProductIDsForTag, []string{"id", "value", "external_id", "created_at", "updated_at", "metadata"})
		if projectErr != nil {
			writeCollectionError(c, projectErr)
			return
		}
		output = append(output, row)
	}
	c.JSON(http.StatusOK, gin.H{"product_tags": output, "count": count, "offset": offset, "limit": limit})
}

func (h *ProductCatalogHTTP) getTag(c *gin.Context) {
	item, err := h.Service.GetProductTag(c, c.Param("id"))
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	row, err := h.projectCatalog(c, item, h.Service.ListProductIDsForTag, []string{"id", "value", "external_id", "created_at", "updated_at", "metadata"})
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_tag": row})
}

func (h *ProductCatalogHTTP) listTypes(c *gin.Context) {
	limit, offset, ok := productCatalogPage(c)
	if !ok {
		return
	}
	filter := productCatalogFilter(c)
	if !validateProductCatalogFilters(c, filter) {
		return
	}
	items, count, err := h.Service.ListStoreProductTypes(c, filter, limit, offset)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	output := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row, projectErr := h.projectCatalog(c, item, h.Service.ListProductIDsForType, []string{"id", "value", "external_id", "created_at", "updated_at", "metadata"})
		if projectErr != nil {
			writeCollectionError(c, projectErr)
			return
		}
		output = append(output, row)
	}
	c.JSON(http.StatusOK, gin.H{"product_types": output, "count": count, "offset": offset, "limit": limit})
}

func (h *ProductCatalogHTTP) getType(c *gin.Context) {
	item, err := h.Service.GetProductType(c, c.Param("id"))
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	row, err := h.projectCatalog(c, item, h.Service.ListProductIDsForType, []string{"id", "value", "external_id", "created_at", "updated_at", "metadata"})
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_type": row})
}

type ProductIDsLookup func(context.Context, string) ([]string, error)

func (h *ProductCatalogHTTP) projectCatalog(c *gin.Context, item any, productIDs ProductIDsLookup, defaults []string) (map[string]any, error) {
	requested := strings.Split(c.Query("fields"), ",")
	if c.Query("fields") == "" {
		requested = append([]string(nil), defaults...)
	}
	encoded, _ := json.Marshal(item)
	var source map[string]any
	_ = json.Unmarshal(encoded, &source)
	includeProducts := false
	for _, field := range requested {
		field = strings.TrimSpace(field)
		if field == "*" || field == "products" || field == "*products" || strings.HasPrefix(field, "products.") {
			includeProducts = true
		}
		if !validCatalogField(field) {
			return nil, errors.New("invalid product catalog field: " + field)
		}
	}
	if includeProducts {
		id, _ := source["id"].(string)
		ids, err := productIDs(c, id)
		if err != nil {
			return nil, err
		}
		products := make([]any, 0, len(ids))
		for _, productID := range ids {
			product, err := h.Service.GetProduct(c, productID)
			if err != nil || product.Status != "published" {
				continue
			}
			encodedProduct, _ := json.Marshal(product)
			var productMap map[string]any
			_ = json.Unmarshal(encodedProduct, &productMap)
			products = append(products, projectProductFields(productMap, catalogProductProjection(requested)))
		}
		source["products"] = products
	}
	out := map[string]any{}
	includeDefaults := c.Query("fields") == ""
	for _, field := range requested {
		if strings.HasPrefix(field, "+") || strings.HasPrefix(field, "-") {
			includeDefaults = true
		}
	}
	if includeDefaults {
		for _, field := range defaults {
			if field != "metadata" {
				out[field] = source[field]
			} else {
				out[field] = source[field]
			}
		}
	}
	for _, field := range requested {
		field = strings.TrimSpace(field)
		if field == "*" {
			return source, nil
		}
		if strings.HasPrefix(field, "-") {
			delete(out, strings.TrimPrefix(field, "-"))
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
					if productMap, ok := value.(map[string]any); ok {
						projected = append(projected, projectProductFields(productMap, nil))
					}
				}
				out["products"] = projected
			}
		} else if strings.HasPrefix(field, "products.") {
			if products, ok := source["products"].([]any); ok {
				projected := make([]any, 0, len(products))
				for _, value := range products {
					if productMap, ok := value.(map[string]any); ok {
						projected = append(projected, projectProductFields(productMap, []string{strings.TrimPrefix(field, "products.")}))
					}
				}
				out["products"] = projected
			}
		} else if value, ok := source[field]; ok {
			out[field] = value
		}
	}
	return out, nil
}

func validCatalogField(field string) bool {
	field = strings.TrimPrefix(strings.TrimPrefix(field, "+"), "-")
	if field == "" || field == "*" || field == "id" || field == "value" || field == "external_id" || field == "created_at" || field == "updated_at" || field == "metadata" || field == "deleted_at" || field == "products" || field == "*products" {
		return true
	}
	if !strings.HasPrefix(field, "products.") {
		return false
	}
	return validCollectionProductField(strings.TrimPrefix(field, "products."))
}

func validCollectionProductField(field string) bool {
	for _, candidate := range []string{"*", "id", "title", "subtitle", "description", "handle", "is_giftcard", "discountable", "thumbnail", "collection_id", "type_id", "weight", "length", "height", "width", "hs_code", "origin_country", "mid_code", "material", "created_at", "updated_at", "metadata", "status", "external_id", "deleted_at", "options", "*options", "options.id", "options.title", "options.is_exclusive", "options.values", "options.values.id", "options.values.value", "*options.values", "variants", "*variants", "variants.id", "variants.title", "variants.sku", "variants.barcode", "variants.ean", "variants.upc", "variants.thumbnail", "variants.manage_inventory", "variants.allow_backorder", "variants.weight", "variants.length", "variants.height", "variants.width", "variants.hs_code", "variants.origin_country", "variants.mid_code", "variants.material", "variants.created_at", "variants.updated_at"} {
		if field == candidate {
			return true
		}
	}
	return false
}

func catalogProductProjection(fields []string) []string {
	var productFields []string
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "products" || field == "*products" {
			return nil
		}
		if strings.HasPrefix(field, "products.") {
			productFields = append(productFields, strings.TrimPrefix(field, "products."))
		}
	}
	return productFields
}

func productCatalogPage(c *gin.Context) (int, int, bool) {
	limit, err := positiveQueryInt(c, "limit", 50)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "limit must be a positive integer"})
		return 0, 0, false
	}
	offset := 0
	if c.Query("offset") != "" {
		offset, err = strconv.Atoi(c.Query("offset"))
	}
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "offset must be a non-negative integer"})
		return 0, 0, false
	}
	return limit, offset, true
}

func productCatalogFilter(c *gin.Context) models.ProductCatalogFilter {
	return models.ProductCatalogFilter{IDs: queryValues(c, "id"), Values: queryValues(c, "value"), ExternalIDs: queryValues(c, "external_id"), Query: c.Query("q"), CreatedAt: collectionDateFilters(c, "created_at"), UpdatedAt: collectionDateFilters(c, "updated_at"), DeletedAt: collectionDateFilters(c, "deleted_at")}
}

func validateProductCatalogFilters(c *gin.Context, filter models.ProductCatalogFilter) bool {
	for _, dateFilters := range []map[string]string{filter.CreatedAt, filter.UpdatedAt, filter.DeletedAt} {
		for operator, value := range dateFilters {
			if operator != "$eq" && operator != "$ne" && operator != "$gt" && operator != "$gte" && operator != "$lt" && operator != "$lte" {
				c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "unsupported timestamp filter operator: " + operator})
				return false
			}
			if value == "null" && (operator == "$eq" || operator == "$ne") {
				continue
			}
			if _, err := time.Parse(time.RFC3339, value); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "timestamp filters must use RFC3339 values"})
				return false
			}
		}
	}
	return true
}
