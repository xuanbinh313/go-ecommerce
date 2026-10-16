package store

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
	"github.com/gin-gonic/gin"
)

type StoreCategoryService interface {
	ListStoreProductCategories(context.Context, models.StoreProductCategoryFilter, int, int) ([]models.ProductCategory, int, error)
	ListProductCategories(context.Context, []string, int, int) ([]models.ProductCategory, int, error)
	GetProductCategory(context.Context, string) (models.ProductCategory, error)
	ListProductIDsForCategory(context.Context, string) ([]string, error)
	GetProduct(context.Context, string) (models.Product, error)
}

type StoreProductCategoryHTTP struct{ Service StoreCategoryService }

var storeProductCategoryFields = []string{"id", "name", "description", "handle", "rank", "external_id", "parent_category_id", "created_at", "updated_at", "metadata"}

func (h *StoreProductCategoryHTTP) Register(router *gin.RouterGroup) {
	router.GET("/store/product-categories", h.list)
	router.GET("/store/product-categories/:id", h.get)
}

func (h *StoreProductCategoryHTTP) list(c *gin.Context) {
	limit, err := positiveQueryInt(c, "limit", 50)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "limit must be a positive integer"})
		return
	}
	offset := 0
	if c.Query("offset") != "" {
		offset, err = strconv.Atoi(c.Query("offset"))
	}
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "offset must be a non-negative integer"})
		return
	}
	filter := storeCategoryFilter(c)
	if !validCategoryDates(c, filter) {
		return
	}
	if !validateStoreCategoryFields(c.Query("fields")) {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "fields contains unsupported product category fields"})
		return
	}
	rows, count, err := h.Service.ListStoreProductCategories(c, filter, limit, offset)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	all, _, err := h.Service.ListProductCategories(c, nil, 100000, 0)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		data, err := h.renderCategory(c, row, all, c.Query("include_descendants_tree") == "true", c.Query("include_ancestors_tree") == "true", c.Query("fields"), map[string]bool{})
		if err != nil {
			writeCollectionError(c, err)
			return
		}
		result = append(result, data)
	}
	c.JSON(http.StatusOK, gin.H{"product_categories": result, "count": count, "offset": offset, "limit": limit})
}

func (h *StoreProductCategoryHTTP) get(c *gin.Context) {
	filter := storeCategoryFilter(c)
	filter.IDs = []string{c.Param("id")}
	if !validateStoreCategoryFields(c.Query("fields")) {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "fields contains unsupported product category fields"})
		return
	}
	rows, _, err := h.Service.ListStoreProductCategories(c, filter, 1, 0)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	if len(rows) == 0 {
		writeCollectionError(c, errProductCategoryNotFound(c.Param("id")))
		return
	}
	all, _, err := h.Service.ListProductCategories(c, nil, 100000, 0)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	category, err := h.renderCategory(c, rows[0], all, c.Query("include_descendants_tree") == "true", c.Query("include_ancestors_tree") == "true", c.Query("fields"), map[string]bool{})
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_category": category})
}

func storeCategoryFilter(c *gin.Context) models.StoreProductCategoryFilter {
	return models.StoreProductCategoryFilter{IDs: queryValues(c, "id"), Names: queryValues(c, "name"), Descriptions: queryValues(c, "description"), Handles: queryValues(c, "handle"), ParentCategoryIDs: queryValues(c, "parent_category_id"), ExternalIDs: queryValues(c, "external_id"), Query: c.Query("q"), CreatedAt: collectionDateFilters(c, "created_at"), UpdatedAt: collectionDateFilters(c, "updated_at")}
}

func validCategoryDates(c *gin.Context, filter models.StoreProductCategoryFilter) bool {
	for _, values := range []map[string]string{filter.CreatedAt, filter.UpdatedAt} {
		for op, value := range values {
			if op != "$eq" && op != "$ne" && op != "$gt" && op != "$gte" && op != "$lt" && op != "$lte" {
				c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "unsupported timestamp filter operator: " + op})
				return false
			}
			if _, err := time.Parse(time.RFC3339, value); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "timestamp filters must use RFC3339 values"})
				return false
			}
		}
	}
	return true
}

func (h *StoreProductCategoryHTTP) renderCategory(c *gin.Context, row models.ProductCategory, all []models.ProductCategory, includeDescendants, includeAncestors bool, requested string, path map[string]bool) (map[string]any, error) {
	if path[row.ID] {
		return nil, nil
	}
	path[row.ID] = true
	defer delete(path, row.ID)
	encoded, _ := json.Marshal(row)
	var source map[string]any
	_ = json.Unmarshal(encoded, &source)
	fields := strings.Split(requested, ",")
	if requested == "" {
		fields = append(append([]string(nil), storeProductCategoryFields...), "*parent_category", "*category_children")
	}
	out := map[string]any{}
	includeAll := false
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "*" {
			includeAll = true
		}
		if strings.HasPrefix(field, "+") || strings.HasPrefix(field, "-") {
			for _, def := range storeProductCategoryFields {
				out[def] = source[def]
			}
		}
	}
	if includeAll {
		for key, value := range source {
			out[key] = value
		}
	} else {
		for _, field := range fields {
			field = strings.TrimSpace(field)
			if field == "*parent_category" || field == "*category_children" || field == "*" {
				continue
			}
			if strings.HasPrefix(field, "-") {
				delete(out, strings.Split(strings.TrimPrefix(field, "-"), ".")[0])
				continue
			}
			field = strings.TrimPrefix(field, "+")
			if isCategoryScalarField(field) {
				if value, ok := source[field]; ok {
					out[field] = value
				}
			}
		}
	}
	if includesCategoryProducts(fields) {
		ids, err := h.Service.ListProductIDsForCategory(c, row.ID)
		if err != nil {
			return nil, err
		}
		products := make([]any, 0, len(ids))
		for _, id := range ids {
			product, err := h.Service.GetProduct(c, id)
			if err != nil || product.Status != "published" {
				continue
			}
			raw, _ := json.Marshal(product)
			var item map[string]any
			_ = json.Unmarshal(raw, &item)
			products = append(products, projectProductFields(item, categoryProductProjection(fields)))
		}
		out["products"] = products
	}
	for _, related := range []struct {
		name          string
		enabled       bool
		relationDepth int
	}{{"parent_category", includeAncestors || categoryRelationRequested(fields, "parent_category"), 0}, {"category_children", includeDescendants || categoryRelationRequested(fields, "category_children"), 0}} {
		if !related.enabled {
			continue
		}
		if related.name == "parent_category" {
			if row.ParentCategoryID == nil {
				out[related.name] = nil
				continue
			}
			parent := categoryByID(all, *row.ParentCategoryID)
			if parent != nil {
				nestedFields := nestedCategoryFields(fields, "parent_category", includeAncestors)
				nested, err := h.renderCategory(c, *parent, all, false, includeAncestors, strings.Join(nestedFields, ","), path)
				if err != nil {
					return nil, err
				}
				out[related.name] = nested
			}
			continue
		}
		children := make([]any, 0)
		for _, child := range all {
			if child.ParentCategoryID != nil && *child.ParentCategoryID == row.ID {
				nestedFields := nestedCategoryFields(fields, "category_children", includeDescendants)
				nested, err := h.renderCategory(c, child, all, includeDescendants, false, strings.Join(nestedFields, ","), path)
				if err != nil {
					return nil, err
				}
				children = append(children, nested)
			}
		}
		out[related.name] = children
	}
	return out, nil
}

func categoryByID(categories []models.ProductCategory, id string) *models.ProductCategory {
	for i := range categories {
		if categories[i].ID == id {
			return &categories[i]
		}
	}
	return nil
}
func isCategoryScalarField(field string) bool {
	for _, name := range storeProductCategoryFields {
		if name == field {
			return true
		}
	}
	return field == "deleted_at"
}
func categoryRelationRequested(fields []string, relation string) bool {
	for _, field := range fields {
		field = strings.TrimSpace(strings.TrimLeft(field, "+-"))
		if field == "*" || field == "*"+relation || field == relation || strings.HasPrefix(field, relation+".") {
			return true
		}
	}
	return false
}
func includesCategoryProducts(fields []string) bool {
	for _, field := range fields {
		field = strings.TrimSpace(strings.TrimLeft(field, "+-"))
		if field == "products" || strings.HasPrefix(field, "products.") || field == "*" {
			return true
		}
	}
	return false
}

func categoryProductProjection(fields []string) []string {
	var productFields []string
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "products" {
			return nil
		}
		if strings.HasPrefix(field, "products.") {
			productFields = append(productFields, strings.TrimPrefix(field, "products."))
		}
	}
	return productFields
}

func nestedCategoryFields(fields []string, relation string, recursive bool) []string {
	if recursive {
		return append(append([]string(nil), storeProductCategoryFields...), "mpath", "*"+relation)
	}
	var nested []string
	for _, field := range fields {
		field = strings.TrimSpace(strings.TrimLeft(field, "+-"))
		if field == "*"+relation {
			return append([]string(nil), storeProductCategoryFields...)
		}
		if strings.HasPrefix(field, relation+".") {
			nested = append(nested, strings.TrimPrefix(field, relation+"."))
		}
	}
	if len(nested) == 0 {
		return append([]string(nil), storeProductCategoryFields...)
	}
	return nested
}

func validateStoreCategoryFields(requested string) bool {
	if requested == "" {
		return true
	}
	fields := strings.Split(requested, ",")
	for _, field := range fields {
		field = strings.TrimSpace(strings.TrimLeft(field, "+-"))
		if field == "*" || isCategoryScalarField(field) || field == "*parent_category" || field == "*category_children" || field == "parent_category" || field == "category_children" || field == "products" || field == "products.title" || field == "products.variants" || field == "products.options" {
			continue
		}
		if strings.HasPrefix(field, "parent_category.") || strings.HasPrefix(field, "category_children.") {
			parts := strings.Split(field, ".")
			if len(parts) <= 4 {
				continue
			}
		}
		return false
	}
	return true
}
func errProductCategoryNotFound(id string) error {
	return fmt.Errorf("product category with id: %s was not found", id)
}
