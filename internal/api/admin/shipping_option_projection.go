package admin

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	"github.com/gin-gonic/gin"
)

var defaultShippingOptionFields = []string{
	"id", "name", "price_type", "data", "provider_id", "metadata", "created_at", "updated_at",
	"*rules", "*type", "*prices", "*prices.price_rules", "*service_zone", "*shipping_profile", "*provider",
}

func projectShippingOption(c *gin.Context, option models.ShippingOption) map[string]any {
	raw, _ := json.Marshal(option)
	var source map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	_ = decoder.Decode(&source)

	return projectShippingOptionFields(c, source, defaultShippingOptionFields)
}

func projectShippingOptionRule(c *gin.Context, rule models.ShippingOptionRule) map[string]any {
	raw, _ := json.Marshal(rule)
	var source map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	_ = decoder.Decode(&source)
	if value, ok := source["value"]; ok {
		source["values"] = map[string]any{"value": value}
	}
	return projectShippingOptionFields(c, source, []string{"id", "description", "attribute", "operator", "values.value"})
}

func projectShippingOptionFields(c *gin.Context, source map[string]any, defaults []string) map[string]any {
	requested := strings.Split(c.Query("fields"), ",")
	if len(requested) == 1 && strings.TrimSpace(requested[0]) == "" {
		requested = defaults
	}
	positive := make([]string, 0, len(requested))
	additive := []string{}
	negative := []string{}
	for _, field := range requested {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if strings.HasPrefix(field, "-") {
			negative = append(negative, strings.TrimPrefix(field, "-"))
			continue
		}
		if strings.HasPrefix(field, "+") {
			additive = append(additive, strings.TrimPrefix(field, "+"))
		} else {
			positive = append(positive, field)
		}
	}
	if len(positive) == 0 {
		positive = append(positive, defaults...)
	}
	positive = append(positive, additive...)
	projected := map[string]any{}
	for _, field := range positive {
		path := strings.Split(strings.TrimPrefix(field, "*"), ".")
		if field == "*" {
			for key, value := range source {
				projected[key] = value
			}
			break
		}
		if strings.HasPrefix(field, "*") {
			path[0] = "*" + path[0]
		}
		addShippingOptionField(projected, source, path)
	}
	for _, field := range negative {
		removeShippingOptionField(projected, strings.Split(field, "."))
	}
	return projected
}

func addShippingOptionField(dst, src map[string]any, path []string) {
	if len(path) == 0 {
		return
	}
	key := strings.TrimPrefix(path[0], "*")
	value, ok := src[key]
	if !ok {
		return
	}
	if len(path) == 1 || strings.HasPrefix(path[0], "*") {
		dst[key] = value
		return
	}
	var child map[string]any
	if existing, ok := dst[key].(map[string]any); ok {
		child = existing
	} else {
		child = map[string]any{}
	}
	if srcMap, ok := value.(map[string]any); ok {
		addShippingOptionField(child, srcMap, path[1:])
		dst[key] = child
		return
	}
	if srcRows, ok := value.([]any); ok {
		outRows := make([]any, len(srcRows))
		var existingRows []any
		if rows, ok := dst[key].([]any); ok {
			existingRows = rows
		}
		for i, row := range srcRows {
			rowMap, ok := row.(map[string]any)
			if !ok {
				outRows[i] = row
				continue
			}
			projectedRow := map[string]any{}
			if i < len(existingRows) {
				if current, ok := existingRows[i].(map[string]any); ok {
					projectedRow = current
				}
			}
			addShippingOptionField(projectedRow, rowMap, path[1:])
			outRows[i] = projectedRow
		}
		dst[key] = outRows
	}
}

func removeShippingOptionField(dst map[string]any, path []string) {
	if len(path) == 0 {
		return
	}
	if len(path) == 1 {
		delete(dst, path[0])
		return
	}
	value, ok := dst[path[0]]
	if !ok {
		return
	}
	if child, ok := value.(map[string]any); ok {
		removeShippingOptionField(child, path[1:])
	} else if rows, ok := value.([]any); ok {
		for _, row := range rows {
			if child, ok := row.(map[string]any); ok {
				removeShippingOptionField(child, path[1:])
			}
		}
	}
}
