package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	flows "github.com/example/medusa-go-clone/internal/coreflows/fulfillment/workflows"
	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	"github.com/gin-gonic/gin"
)

func (h *FulfillmentHTTP) listShippingOptions(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 {
		limit = 20
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}
	f := models.ShippingOptionFilter{IDs: queryValues(c, "id"), ServiceZoneIDs: queryValues(c, "service_zone_id"), ShippingProfileIDs: queryValues(c, "shipping_profile_id"), ProviderIDs: queryValues(c, "provider_id"), TypeIDs: queryValues(c, "shipping_option_type_id"), StockLocationIDs: queryValues(c, "stock_location_id"), Query: c.Query("q"), CreatedAt: paymentDateFilters(c, "created_at"), UpdatedAt: paymentDateFilters(c, "updated_at"), DeletedAt: paymentDateFilters(c, "deleted_at")}
	rows, count, err := h.Service.ListShippingOptions(c, f, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	projected := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		projected = append(projected, projectShippingOption(c, row))
	}
	c.JSON(http.StatusOK, gin.H{"shipping_options": projected, "count": count, "offset": offset, "limit": limit})
}
func (h *FulfillmentHTTP) getShippingOption(c *gin.Context) {
	row, err := h.Service.GetShippingOption(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"shipping_option": projectShippingOption(c, row)})
}

func (h *FulfillmentHTTP) deleteShippingOption(c *gin.Context) {
	id := c.Param("id")
	if err := flows.DeleteShippingOptions(c, h.Service, h.Events, []string{id}); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "shipping_option", "deleted": true})
}

func (h *FulfillmentHTTP) createShippingOption(c *gin.Context) {
	var input models.ShippingOptionInput
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := flows.CreateShippingOption(c, h.Service, h.Events, input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"shipping_option": projectShippingOption(c, row)})
}

func (h *FulfillmentHTTP) updateShippingOption(c *gin.Context) {
	var input models.ShippingOptionUpdateInput
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if input.Rules != nil {
		for _, rule := range *input.Rules {
			if rule.Attribute == nil || rule.Operator == nil || rule.Value == nil {
				c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "shipping option rules require attribute, operator, and value"})
				return
			}
		}
	}
	if input.Prices != nil {
		for _, price := range *input.Prices {
			if price.ID == nil && ((price.CurrencyCode == nil) == (price.RegionID == nil) || price.Amount == nil) || price.CurrencyCode != nil && price.RegionID != nil {
				c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "shipping option prices require an id or exactly one of currency_code or region_id and an amount"})
				return
			}
		}
	}
	row, err := flows.UpdateShippingOption(c, h.Service, h.Events, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"shipping_option": projectShippingOption(c, row)})
}

func (h *FulfillmentHTTP) batchShippingOptionRules(c *gin.Context) {
	var input models.ShippingOptionRuleBatchInput
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	result, err := flows.BatchShippingOptionRules(c, h.Service, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	created := make([]map[string]any, 0, len(result.Created))
	for _, rule := range result.Created {
		created = append(created, projectShippingOptionRule(c, rule))
	}
	updated := make([]map[string]any, 0, len(result.Updated))
	for _, rule := range result.Updated {
		updated = append(updated, projectShippingOptionRule(c, rule))
	}
	c.JSON(http.StatusOK, gin.H{"created": created, "updated": updated, "deleted": gin.H{"ids": result.Deleted, "object": "shipping_option_rule", "deleted": true}})
}
