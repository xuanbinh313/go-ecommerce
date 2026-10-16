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

type StoreProductVariantService interface {
	ListProductVariants(context.Context, string, models.ProductVariantFilter, int, int) ([]models.ProductVariant, int, error)
}

type StoreProductVariantHTTP struct{ Service StoreProductVariantService }

var storeProductVariantDefaults = []string{"id", "title", "sku", "barcode", "ean", "upc", "allow_backorder", "manage_inventory", "variant_rank", "product_id", "thumbnail", "hs_code", "origin_country", "mid_code", "material", "weight", "length", "height", "width", "created_at", "updated_at", "metadata", "options", "images", "product"}

func (h *StoreProductVariantHTTP) Register(router *gin.RouterGroup) {
	router.GET("/store/product-variants", h.list)
	router.GET("/store/product-variants/:id", h.get)
}

func (h *StoreProductVariantHTTP) list(c *gin.Context) {
	limit, offset, ok := productVariantPage(c)
	if !ok {
		return
	}
	filter, valid := storeProductVariantFilter(c)
	if !valid {
		return
	}
	rows, count, err := h.Service.ListProductVariants(c, "", filter, limit, offset)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	fields, err := storeProductVariantFields(c.Query("fields"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if unsupported := unsupportedVariantFields(fields, c.Query("fields") != ""); len(unsupported) > 0 {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "store variant fields are unavailable: " + strings.Join(unsupported, ",")})
		return
	}
	variants := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		variants = append(variants, projectStoreVariant(row, fields))
	}
	c.JSON(http.StatusOK, gin.H{"variants": variants, "count": count, "offset": offset, "limit": limit})
}

func (h *StoreProductVariantHTTP) get(c *gin.Context) {
	filter := models.ProductVariantFilter{IDs: []string{c.Param("id")}, PublishedOnly: true}
	if channels := queryValues(c, "sales_channel_id"); len(channels) > 0 {
		filter.SalesChannelIDs = channels
	}
	rows, _, err := h.Service.ListProductVariants(c, "", filter, 1, 0)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	if len(rows) == 0 {
		writeCollectionError(c, fmt.Errorf("product variant with id: %s was not found", c.Param("id")))
		return
	}
	fields, err := storeProductVariantFields(c.Query("fields"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if unsupported := unsupportedVariantFields(fields, c.Query("fields") != ""); len(unsupported) > 0 {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "store variant fields are unavailable: " + strings.Join(unsupported, ",")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"variant": projectStoreVariant(rows[0], fields)})
}

func productVariantPage(c *gin.Context) (int, int, bool) {
	limit, err := positiveQueryInt(c, "limit", 20)
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

func storeProductVariantFilter(c *gin.Context) (models.ProductVariantFilter, bool) {
	filter := models.ProductVariantFilter{IDs: queryValues(c, "id"), ProductIDs: queryValues(c, "product_id"), SKUs: queryValues(c, "sku"), Query: c.Query("q"), PublishedOnly: true, SalesChannelIDs: queryValues(c, "sales_channel_id"), CreatedAt: collectionDateFilters(c, "created_at"), UpdatedAt: collectionDateFilters(c, "updated_at"), DeletedAt: collectionDateFilters(c, "deleted_at")}
	parseBool := func(key string) (*bool, bool) {
		raw := c.Query(key)
		if raw == "" {
			return nil, true
		}
		value, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": key + " must be true or false"})
			return nil, false
		}
		return &value, true
	}
	var valid bool
	if filter.AllowBackorder, valid = parseBool("allow_backorder"); !valid {
		return filter, false
	}
	if filter.ManageInventory, valid = parseBool("manage_inventory"); !valid {
		return filter, false
	}
	options := c.QueryMap("options")
	if value := options["value"]; value != "" {
		filter.OptionValue = &value
	}
	if id := options["option_id"]; id != "" {
		filter.OptionID = &id
	}
	for _, dates := range []map[string]string{filter.CreatedAt, filter.UpdatedAt, filter.DeletedAt} {
		for operator, value := range dates {
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

func storeProductVariantFields(requested string) ([]string, error) {
	if requested == "" {
		return append([]string(nil), storeProductVariantDefaults...), nil
	}
	fields := strings.Split(requested, ",")
	allowed := map[string]bool{"*": true, "id": true, "title": true, "sku": true, "barcode": true, "ean": true, "upc": true, "allow_backorder": true, "manage_inventory": true, "variant_rank": true, "product_id": true, "thumbnail": true, "hs_code": true, "origin_country": true, "mid_code": true, "material": true, "weight": true, "length": true, "height": true, "width": true, "created_at": true, "updated_at": true, "metadata": true, "deleted_at": true, "options": true, "*options": true, "options.id": true, "options.value": true, "images": true, "*images": true, "images.id": true, "images.url": true, "images.rank": true, "product.id": true, "product.type_id": true, "calculated_price": true, "inventory_quantity": true}
	for _, field := range fields {
		field = strings.TrimSpace(strings.TrimLeft(field, "+-"))
		if !allowed[field] {
			return nil, fmt.Errorf("invalid product variant field: %s", field)
		}
	}
	return fields, nil
}

func unsupportedVariantFields(fields []string, explicit bool) []string {
	unsupported := []string{}
	for _, field := range fields {
		field = strings.TrimSpace(strings.TrimLeft(field, "+-"))
		if field == "calculated_price" || field == "inventory_quantity" || (explicit && strings.HasPrefix(field, "images")) {
			unsupported = append(unsupported, field)
		}
	}
	return unsupported
}

func projectStoreVariant(row models.ProductVariant, fields []string) map[string]any {
	encoded, _ := json.Marshal(row)
	var source map[string]any
	_ = json.Unmarshal(encoded, &source)
	source["images"] = []any{}
	out := map[string]any{}
	includeDefaults := false
	optionFields := []string{}
	allOptions := false
	for _, field := range fields {
		if strings.HasPrefix(field, "+") || strings.HasPrefix(field, "-") {
			includeDefaults = true
		}
	}
	if includeDefaults {
		for _, field := range storeProductVariantDefaults {
			if field != "options" && field != "images" && field != "product" {
				out[field] = source[field]
			}
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
		if field == "options" || field == "*options" {
			allOptions = true
			continue
		}
		if strings.HasPrefix(field, "options.") {
			optionFields = append(optionFields, strings.TrimPrefix(field, "options."))
			continue
		}
		if field == "product" || field == "product.id" || field == "product.type_id" {
			copyCollectionField(out, source, field)
			continue
		}
		if value, ok := source[field]; ok {
			out[field] = value
		}
	}
	if allOptions {
		out["options"] = source["options"]
	} else if len(optionFields) > 0 {
		options, _ := source["options"].([]any)
		projected := make([]any, 0, len(options))
		for _, value := range options {
			if option, ok := value.(map[string]any); ok {
				selected := map[string]any{}
				for _, field := range optionFields {
					if v, exists := option[field]; exists {
						selected[field] = v
					}
				}
				projected = append(projected, selected)
			}
		}
		out["options"] = projected
	}
	if strings.Contains(strings.Join(fields, ","), "images") {
		out["images"] = source["images"]
	}
	return out
}
