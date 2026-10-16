package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	flows "github.com/example/medusa-go-clone/internal/coreflows/fulfillment/workflows"
	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	"github.com/gin-gonic/gin"
)

func (h *FulfillmentHTTP) listShippingOptionTypes(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	split := func(k string) []string {
		v := c.QueryArray(k)
		if len(v) == 0 && c.Query(k) != "" {
			v = strings.Split(c.Query(k), ",")
		}
		return v
	}
	rows, count, err := h.Service.ListShippingOptionTypes(c, models.ShippingOptionTypeFilter{IDs: split("id"), Labels: split("label"), Codes: split("code"), Descriptions: split("description"), Query: c.Query("q")}, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"shipping_option_types": rows, "count": count, "offset": offset, "limit": limit})
}
func (h *FulfillmentHTTP) createShippingOptionType(c *gin.Context) {
	var in models.ShippingOptionTypeInput
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(&in); err != nil || in.Label == "" || in.Code == "" {
		c.JSON(400, gin.H{"type": "invalid_data", "message": "label and code are required"})
		return
	}
	rows, err := flows.CreateShippingOptionTypes(c, h.Service, []models.ShippingOptionTypeInput{in})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(rows) == 0 {
		c.JSON(500, gin.H{"type": "unexpected_state", "message": "create-shipping-option-types returned no result"})
		return
	}
	c.JSON(200, gin.H{"shipping_option_type": rows[0]})
}
func (h *FulfillmentHTTP) getShippingOptionType(c *gin.Context) {
	row, err := h.Service.GetShippingOptionType(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"shipping_option_type": row})
}
func (h *FulfillmentHTTP) updateShippingOptionType(c *gin.Context) {
	var in models.ShippingOptionTypeUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := flows.UpdateShippingOptionTypes(c, h.Service, c.Param("id"), in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"shipping_option_type": row})
}
func (h *FulfillmentHTTP) deleteShippingOptionType(c *gin.Context) {
	id := c.Param("id")
	if err := flows.DeleteShippingOptionTypes(c, h.Service, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "shipping_option_type", "deleted": true})
}
