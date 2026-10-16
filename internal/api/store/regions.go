package store

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	paymentModels "github.com/example/medusa-go-clone/internal/modules/payment/domain/models"
	"github.com/example/medusa-go-clone/internal/modules/region/domain/models"
	"github.com/gin-gonic/gin"
)

type StoreRegionService interface {
	List(context.Context) ([]models.Region, error)
	Get(context.Context, string) (models.Region, error)
}

type RegionHTTP struct {
	Service  StoreRegionService
	Links    RegionPaymentLinks
	Payments PaymentProviderReader
}

func (h *RegionHTTP) Register(router *gin.RouterGroup) {
	router.GET("/store/regions", h.list)
	router.GET("/store/regions/:id", h.get)
}

var regionFields = []string{"id", "name", "currency_code", "created_at", "updated_at", "deleted_at", "metadata", "countries"}
var regionExtraFields = []string{"payment_providers", "payment_providers.id", "payment_providers.is_enabled"}

func queryOffset(c *gin.Context) (int, error) {
	if c.Query("offset") == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(c.Query("offset"))
	if err != nil || n < 0 {
		return 0, errors.New("invalid offset")
	}
	return n, nil
}

func (h *RegionHTTP) list(c *gin.Context) {
	limit, err := positiveQueryInt(c, "limit", 20)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "limit must be a positive integer"})
		return
	}
	offset, err := queryOffset(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "offset must be a non-negative integer"})
		return
	}
	rows, err := h.Service.List(c)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	ids, currencies, names := queryValues(c, "id"), queryValues(c, "currency_code"), queryValues(c, "name")
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	createdAt, updatedAt := collectionDateFilters(c, "created_at"), collectionDateFilters(c, "updated_at")
	if !validateRegionDateFilters(c, createdAt) || !validateRegionDateFilters(c, updatedAt) {
		return
	}
	filtered := make([]models.Region, 0, len(rows))
	for _, r := range rows {
		if len(ids) > 0 && !containsFold(ids, r.ID) || len(currencies) > 0 && !containsFold(currencies, r.CurrencyCode) || len(names) > 0 && !containsFold(names, r.Name) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(r.Name+" "+r.CurrencyCode), q) {
			continue
		}
		if !regionDateMatches(r.CreatedAt, createdAt) || !regionDateMatches(r.UpdatedAt, updatedAt) {
			continue
		}
		filtered = append(filtered, r)
	}
	count := len(filtered)
	if offset >= count {
		filtered = []models.Region{}
	} else {
		end := offset + limit
		if end > count {
			end = count
		}
		filtered = filtered[offset:end]
	}
	fields, err := regionProjection(c.Query("fields"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	out := make([]map[string]any, 0, len(filtered))
	for _, r := range filtered {
		providers, providerErr := h.regionPaymentProviders(c, r.ID, fields)
		if providerErr != nil {
			writeCollectionError(c, providerErr)
			return
		}
		out = append(out, projectRegion(r, fields, providers))
	}
	c.JSON(http.StatusOK, gin.H{"regions": out, "count": count, "offset": offset, "limit": limit})
}

func (h *RegionHTTP) get(c *gin.Context) {
	r, err := h.Service.Get(c, c.Param("id"))
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	fields, err := regionProjection(c.Query("fields"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	providers, err := h.regionPaymentProviders(c, r.ID, fields)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"region": projectRegion(r, fields, providers)})
}

func (h *RegionHTTP) regionPaymentProviders(ctx context.Context, regionID string, fields []string) ([]paymentModels.PaymentProvider, error) {
	wanted := false
	for _, field := range fields {
		if field == "*" || field == "payment_providers" || strings.HasPrefix(field, "payment_providers.") {
			wanted = true
			break
		}
	}
	if !wanted {
		return nil, nil
	}
	if h.Links == nil || h.Payments == nil {
		return nil, errors.New("region payment-provider query is not configured")
	}
	links, err := h.Links.List(ctx, "RegionPaymentProvider", map[string]any{"region_id": regionID})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(links))
	for _, link := range links {
		if id, ok := link.Fields["payment_provider_id"].(string); ok {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return []paymentModels.PaymentProvider{}, nil
	}
	rows, _, err := h.Payments.ListPaymentProviders(ctx, paymentModels.PaymentProviderFilter{IDs: ids, Limit: len(ids)})
	if err != nil {
		return nil, err
	}
	byID := make(map[string]paymentModels.PaymentProvider, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	out := make([]paymentModels.PaymentProvider, 0, len(ids))
	for _, id := range ids {
		if row, ok := byID[id]; ok {
			out = append(out, row)
		}
	}
	return out, nil
}

func regionProjection(requested string) ([]string, error) {
	if requested == "" {
		return append([]string(nil), regionFields...), nil
	}
	out := []string{}
	valid := map[string]bool{"*": true}
	for _, f := range regionFields {
		valid[f] = true
	}
	for _, f := range regionExtraFields {
		valid[f] = true
	}
	parts := strings.Split(requested, ",")
	useDefaults := false
	for _, raw := range parts {
		if strings.HasPrefix(strings.TrimSpace(raw), "+") || strings.HasPrefix(strings.TrimSpace(raw), "-") {
			useDefaults = true
		}
	}
	if useDefaults {
		out = append(out, regionFields...)
	}
	for _, raw := range parts {
		f := strings.TrimSpace(raw)
		prefix := ""
		if strings.HasPrefix(f, "+") || strings.HasPrefix(f, "-") {
			prefix, f = f[:1], f[1:]
		}
		f = strings.TrimSpace(strings.TrimPrefix(f, "*"))
		if f == "" {
			f = "*"
		}
		if !valid[f] {
			return nil, errors.New("invalid region field: " + f)
		}
		if prefix == "-" {
			filtered := out[:0]
			for _, current := range out {
				if current != f {
					filtered = append(filtered, current)
				}
			}
			out = filtered
			continue
		}
		if prefix == "+" && !containsFold(out, f) {
			out = append(out, f)
			continue
		}
		if useDefaults && prefix == "" {
			continue
		}
		out = append(out, f)
	}
	return out, nil
}

func validateRegionDateFilters(c *gin.Context, filters map[string]string) bool {
	for op, value := range filters {
		if op != "$eq" && op != "$ne" && op != "$gt" && op != "$gte" && op != "$lt" && op != "$lte" {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "unsupported timestamp filter operator: " + op})
			return false
		}
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "timestamp filters must use RFC3339 values"})
			return false
		}
	}
	return true
}

func regionDateMatches(actual time.Time, filters map[string]string) bool {
	for op, raw := range filters {
		expected, _ := time.Parse(time.RFC3339, raw)
		cmp := actual.Compare(expected)
		switch op {
		case "$eq":
			if cmp != 0 {
				return false
			}
		case "$ne":
			if cmp == 0 {
				return false
			}
		case "$gt":
			if cmp <= 0 {
				return false
			}
		case "$gte":
			if cmp < 0 {
				return false
			}
		case "$lt":
			if cmp >= 0 {
				return false
			}
		case "$lte":
			if cmp > 0 {
				return false
			}
		}
	}
	return true
}

func projectRegion(r models.Region, fields []string, providers []paymentModels.PaymentProvider) map[string]any {
	b, _ := json.Marshal(r)
	src := map[string]any{}
	_ = json.Unmarshal(b, &src)
	src["deleted_at"] = nil
	if providers != nil {
		src["payment_providers"] = projectRegionPaymentProviders(providers, fields)
	}
	out := map[string]any{}
	all := false
	for _, f := range fields {
		if f == "*" {
			all = true
		}
	}
	if all {
		return src
	}
	for _, f := range fields {
		if v, ok := src[f]; ok {
			out[f] = v
		} else if f == "payment_providers.id" || f == "payment_providers.is_enabled" {
			out["payment_providers"] = projectRegionPaymentProviders(providers, []string{f})
		} else if f == "payment_providers" {
			out[f] = projectRegionPaymentProviders(providers, fields)
		}
	}
	return out
}

func projectRegionPaymentProviders(providers []paymentModels.PaymentProvider, fields []string) []map[string]any {
	selectID, selectEnabled, all := false, false, false
	for _, field := range fields {
		if field == "*" || field == "payment_providers" {
			all = true
		}
		if field == "payment_providers.id" {
			selectID = true
		}
		if field == "payment_providers.is_enabled" {
			selectEnabled = true
		}
	}
	if all {
		selectID, selectEnabled = true, true
	}
	out := make([]map[string]any, 0, len(providers))
	for _, provider := range providers {
		item := map[string]any{}
		if selectID {
			item["id"] = provider.ID
		}
		if selectEnabled {
			item["is_enabled"] = provider.IsEnabled
		}
		out = append(out, item)
	}
	return out
}

func containsFold(values []string, value string) bool {
	for _, v := range values {
		if strings.EqualFold(v, value) {
			return true
		}
	}
	return false
}
