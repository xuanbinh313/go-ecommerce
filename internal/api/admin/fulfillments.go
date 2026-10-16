package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"

	flows "github.com/example/medusa-go-clone/internal/coreflows/fulfillment/workflows"
	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	"github.com/gin-gonic/gin"
)

var defaultFulfillmentFields = []string{
	"id", "location_id", "packed_at", "shipped_at", "marked_shipped_by", "created_by", "delivered_at", "canceled_at", "data", "provider_id", "shipping_option_id", "metadata", "order", "created_at", "updated_at", "deleted_at", "*delivery_address", "*items", "*labels",
}

func (h *FulfillmentHTTP) createFulfillment(c *gin.Context) {
	var input models.FulfillmentCreateInput
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if input.LocationID == "" || input.ProviderID == "" || input.OrderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "location_id, provider_id, and order_id are required"})
		return
	}
	if err := validateFulfillmentPayload(input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if err := validateFulfillmentLabels(input.Labels); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if actor, ok := c.Get("actor_id"); ok {
		input.CreatedBy = contextValue(actor)
	}
	location, err := h.Locations.Get(c, input.LocationID)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	locationJSON, _ := json.Marshal(location)
	_ = json.Unmarshal(locationJSON, &input.Location)
	row, err := flows.CreateFulfillment(c, h.Service, h.Events, input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"fulfillment": projectFulfillment(c, row)})
}

func (h *FulfillmentHTTP) cancelFulfillment(c *gin.Context) {
	row, err := flows.CancelFulfillment(c, h.Service, h.Events, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"fulfillment": projectFulfillment(c, row)})
}

func (h *FulfillmentHTTP) createShipment(c *gin.Context) {
	var input models.CreateShipmentInput
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if err := validateFulfillmentLabels(input.Labels); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if actor, ok := c.Get("actor_id"); ok {
		input.MarkedShippedBy = contextValue(actor)
	}
	row, err := flows.CreateFulfillmentShipment(c, h.Service, h.Events, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"fulfillment": projectFulfillment(c, row)})
}

func validateFulfillmentLabels(labels []models.FulfillmentLabelInput) error {
	for _, label := range labels {
		for _, value := range []string{label.TrackingURL, label.LabelURL} {
			if value == "" || value == "#" {
				continue
			}
			parsed, err := url.ParseRequestURI(value)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return fmt.Errorf("label URLs must use http or https")
			}
		}
	}
	return nil
}

func validateFulfillmentPayload(input models.FulfillmentCreateInput) error {
	for _, field := range []struct {
		name string
		raw  json.RawMessage
	}{{"data", input.Data}, {"metadata", input.Metadata}} {
		if len(field.raw) == 0 || string(field.raw) == "null" {
			continue
		}
		var object map[string]any
		if err := json.Unmarshal(field.raw, &object); err != nil || object == nil {
			return fmt.Errorf("%s must be an object or null", field.name)
		}
	}
	for i, item := range input.Items {
		if item.Title == "" || item.SKU == "" || item.Barcode == "" || len(item.Quantity) == 0 {
			return fmt.Errorf("items[%d] requires title, sku, quantity, and barcode", i)
		}
		var number json.Number
		decoder := json.NewDecoder(bytes.NewReader(item.Quantity))
		decoder.UseNumber()
		if decoder.Decode(&number) != nil {
			return fmt.Errorf("items[%d].quantity must be a number", i)
		}
		if _, ok := new(big.Rat).SetString(number.String()); !ok {
			return fmt.Errorf("items[%d].quantity must be a number", i)
		}
	}
	return nil
}

func projectFulfillment(c *gin.Context, row models.FulfillmentRecord) map[string]any {
	encoded, _ := json.Marshal(row)
	var source map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	_ = decoder.Decode(&source)
	delete(source, "order_id")
	if row.OrderID != nil {
		source["order"] = map[string]any{"id": *row.OrderID}
	}
	return projectShippingOptionFields(c, source, defaultFulfillmentFields)
}
