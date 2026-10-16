package store

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	currencymodels "github.com/example/medusa-go-clone/internal/modules/currency/domain/models"
	storemodels "github.com/example/medusa-go-clone/internal/modules/store/domain/models"
	translationmodels "github.com/example/medusa-go-clone/internal/modules/translation/domain/models"
	"github.com/gin-gonic/gin"
)

type StoreCurrencyReader interface {
	ListCurrencies(context.Context, []string) ([]currencymodels.Currency, error)
	RetrieveCurrency(context.Context, string) (currencymodels.Currency, error)
}
type StoreSettingsReader interface {
	List(context.Context) ([]storemodels.Store, error)
}
type StoreLocaleReader interface {
	ListLocales(context.Context, translationmodels.LocaleFilter, int, int) ([]translationmodels.Locale, int, error)
}

type StoreDiscoveryHTTP struct {
	Currencies StoreCurrencyReader
	Stores     StoreSettingsReader
	Locales    StoreLocaleReader
}

var storeCurrencyDefaultFields = []string{"code", "name", "symbol", "symbol_native", "decimal_digits", "rounding"}
var storeCurrencyAllowedFields = map[string]bool{"code": true, "name": true, "symbol": true, "symbol_native": true, "decimal_digits": true, "rounding": true, "created_at": true, "updated_at": true, "deleted_at": true}

func (h *StoreDiscoveryHTTP) Register(router *gin.RouterGroup, translationEnabled bool) {
	router.GET("/store/currencies", h.listCurrencies)
	router.GET("/store/currencies/:code", h.getCurrency)
	if translationEnabled {
		router.GET("/store/locales", h.listLocales)
	}
}

func (h *StoreDiscoveryHTTP) listCurrencies(c *gin.Context) {
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
	rows, err := h.Currencies.ListCurrencies(c, queryValues(c, "code"))
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	query := strings.ToLower(strings.TrimSpace(c.Query("q")))
	filtered := make([]currencymodels.Currency, 0, len(rows))
	for _, row := range rows {
		if query == "" || strings.Contains(strings.ToLower(row.Code+" "+row.Name), query) {
			filtered = append(filtered, row)
		}
	}
	count := len(filtered)
	if offset >= count {
		filtered = []currencymodels.Currency{}
	} else {
		end := offset + limit
		if end > count {
			end = count
		}
		filtered = filtered[offset:end]
	}
	fields, err := currencyFields(c.Query("fields"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	result := make([]map[string]any, 0, len(filtered))
	for _, row := range filtered {
		result = append(result, projectCurrency(row, fields))
	}
	c.JSON(http.StatusOK, gin.H{"currencies": result, "count": count, "offset": offset, "limit": limit})
}

func (h *StoreDiscoveryHTTP) getCurrency(c *gin.Context) {
	row, err := h.Currencies.RetrieveCurrency(c, c.Param("code"))
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	fields, err := currencyFields(c.Query("fields"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"currency": projectCurrency(row, fields)})
}

func currencyFields(requested string) ([]string, error) {
	if requested == "" {
		return append([]string(nil), storeCurrencyDefaultFields...), nil
	}
	fields := strings.Split(requested, ",")
	for _, field := range fields {
		field = strings.TrimSpace(strings.TrimLeft(field, "+-"))
		if !storeCurrencyAllowedFields[field] && field != "*" {
			return nil, errors.New("invalid currency field: " + field)
		}
	}
	return fields, nil
}

func projectCurrency(row currencymodels.Currency, fields []string) map[string]any {
	encoded, _ := json.Marshal(row)
	var source map[string]any
	_ = json.Unmarshal(encoded, &source)
	source["deleted_at"] = nil
	all := false
	includeDefaults := false
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "*" {
			all = true
		}
		if strings.HasPrefix(field, "+") || strings.HasPrefix(field, "-") {
			includeDefaults = true
		}
	}
	if all {
		return source
	}
	out := map[string]any{}
	if includeDefaults {
		for _, field := range storeCurrencyDefaultFields {
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
			delete(out, strings.TrimPrefix(field, "-"))
			continue
		}
		field = strings.TrimPrefix(field, "+")
		if value, ok := source[field]; ok {
			out[field] = value
		}
	}
	return out
}

func (h *StoreDiscoveryHTTP) listLocales(c *gin.Context) {
	stores, err := h.Stores.List(c)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	if len(stores) == 0 {
		c.JSON(http.StatusOK, gin.H{"locales": []map[string]string{}})
		return
	}
	codes := make([]string, 0, len(stores[0].SupportedLocales))
	for _, locale := range stores[0].SupportedLocales {
		codes = append(codes, locale.LocaleCode)
	}
	if len(codes) == 0 {
		c.JSON(http.StatusOK, gin.H{"locales": []map[string]string{}})
		return
	}
	locales, _, err := h.Locales.ListLocales(c, translationmodels.LocaleFilter{Codes: codes}, len(codes), 0)
	if err != nil {
		writeCollectionError(c, err)
		return
	}
	byCode := make(map[string]string, len(locales))
	for _, locale := range locales {
		byCode[locale.Code] = locale.Name
	}
	result := make([]map[string]string, 0, len(codes))
	for _, code := range codes {
		if name, ok := byCode[code]; ok {
			result = append(result, map[string]string{"code": code, "name": name})
		}
	}
	c.JSON(http.StatusOK, gin.H{"locales": result})
}
