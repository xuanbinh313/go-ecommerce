package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *FulfillmentHTTP) listProviders(c *gin.Context) {
	providers, err := h.Service.ListFulfillmentProviders()
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if ids := queryValues(c, "id"); len(ids) > 0 {
		allowed := map[string]bool{}
		for _, id := range ids {
			allowed[id] = true
		}
		filtered := providers[:0]
		for _, p := range providers {
			if allowed[p.ID] {
				filtered = append(filtered, p)
			}
		}
		providers = filtered
	}
	if raw := c.Query("is_enabled"); raw != "" {
		enabled, e := strconv.ParseBool(raw)
		if e != nil {
			c.JSON(400, gin.H{"type": "invalid_data", "message": "is_enabled must be a boolean"})
			return
		}
		filtered := providers[:0]
		for _, p := range providers {
			if p.IsEnabled == enabled {
				filtered = append(filtered, p)
			}
		}
		providers = filtered
	}
	if q := strings.ToLower(c.Query("q")); q != "" {
		filtered := providers[:0]
		for _, p := range providers {
			if strings.Contains(strings.ToLower(p.ID), q) {
				filtered = append(filtered, p)
			}
		}
		providers = filtered
	}
	if locationID := c.Query("stock_location_id"); locationID != "" {
		rows, e := h.Links.List(c, "LocationFulfillmentProvider", map[string]any{"stock_location_id": locationID})
		if e != nil {
			writeModuleError(c, e)
			return
		}
		linked := map[string]bool{}
		for _, row := range rows {
			if id, ok := row.Fields["fulfillment_provider_id"].(string); ok {
				linked[id] = true
			}
		}
		filtered := providers[:0]
		for _, p := range providers {
			if linked[p.ID] {
				filtered = append(filtered, p)
			}
		}
		providers = filtered
	}
	total := len(providers)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	c.JSON(http.StatusOK, gin.H{"fulfillment_providers": providers[offset:end], "count": total, "offset": offset, "limit": limit})
}

func (h *FulfillmentHTTP) providerOptions(c *gin.Context) {
	options, err := h.Service.GetFulfillmentOptions(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	count := len(options)
	c.JSON(200, gin.H{"fulfillment_options": options, "count": count, "limit": count, "offset": 0})
}
