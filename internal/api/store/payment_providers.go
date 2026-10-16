package store

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	linkModels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	paymentModels "github.com/example/medusa-go-clone/internal/modules/payment/domain/models"
	"github.com/gin-gonic/gin"
)

type RegionPaymentLinks interface {
	List(context.Context, string, map[string]any) ([]linkModels.Link, error)
}
type PaymentProviderReader interface {
	ListPaymentProviders(context.Context, paymentModels.PaymentProviderFilter) ([]paymentModels.PaymentProvider, int64, error)
}

type PaymentProviderHTTP struct {
	Links    RegionPaymentLinks
	Payments PaymentProviderReader
}

func (h *PaymentProviderHTTP) Register(router *gin.RouterGroup) {
	router.GET("/store/payment-providers", h.list)
}

func (h *PaymentProviderHTTP) list(c *gin.Context) {
	regionID := c.Query("region_id")
	if regionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"type": "not_allowed", "message": "You must provide the region_id to list payment providers"})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 {
		limit = 20
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}
	relations, err := h.Links.List(c, "RegionPaymentProvider", map[string]any{"region_id": regionID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"type": "database_error", "message": err.Error()})
		return
	}
	count := len(relations)
	start := offset
	if start > count {
		start = count
	}
	end := start + limit
	if end > count {
		end = count
	}
	ids := make([]string, 0, end-start)
	for _, relation := range relations[start:end] {
		if id, ok := relation.Fields["payment_provider_id"].(string); ok {
			ids = append(ids, id)
		}
	}
	providers := []paymentModels.PaymentProvider{}
	if len(ids) > 0 {
		rows, _, queryErr := h.Payments.ListPaymentProviders(c, paymentModels.PaymentProviderFilter{IDs: ids, Limit: len(ids)})
		if queryErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"type": "database_error", "message": queryErr.Error()})
			return
		}
		byID := make(map[string]paymentModels.PaymentProvider, len(rows))
		for _, row := range rows {
			byID[row.ID] = row
		}
		for _, id := range ids {
			if provider, ok := byID[id]; ok {
				providers = append(providers, provider)
			}
		}
	}
	selected := make([]map[string]any, 0, len(providers))
	for _, provider := range providers {
		selected = append(selected, projectStorePaymentProvider(provider, c.Query("fields")))
	}
	c.JSON(http.StatusOK, gin.H{"payment_providers": selected, "count": count, "offset": offset, "limit": limit})
}

func projectStorePaymentProvider(row paymentModels.PaymentProvider, fields string) map[string]any {
	if fields == "*" {
		return map[string]any{"id": row.ID, "is_enabled": row.IsEnabled}
	}
	selected := []string{"id", "is_enabled"}
	if fields != "" {
		selected = strings.Split(fields, ",")
	}
	useDefaults := fields == ""
	for _, field := range selected {
		field = strings.TrimSpace(field)
		if strings.HasPrefix(field, "+") || strings.HasPrefix(field, "-") {
			useDefaults = true
		}
	}
	out := map[string]any{}
	if useDefaults {
		out["id"] = row.ID
		out["is_enabled"] = row.IsEnabled
	}
	for _, field := range selected {
		field = strings.TrimSpace(field)
		if strings.HasPrefix(field, "-") {
			delete(out, strings.TrimPrefix(field, "-"))
			continue
		}
		field = strings.TrimPrefix(field, "+")
		switch field {
		case "id":
			out["id"] = row.ID
		case "is_enabled":
			out["is_enabled"] = row.IsEnabled
		}
	}
	return out
}

func splitFields(value string) []string {
	parts := []string{}
	for _, part := range strings.Split(value, ",") {
		if field := strings.TrimSpace(part); field != "" {
			parts = append(parts, field)
		}
	}
	return parts
}
