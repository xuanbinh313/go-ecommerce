package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	flows "github.com/example/medusa-go-clone/internal/coreflows/fulfillment/workflows"
	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	"github.com/gin-gonic/gin"
)

func (h *FulfillmentHTTP) listShippingProfiles(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	f := models.ShippingProfileFilter{IDs: queryValues(c, "id"), Names: queryValues(c, "name"), Type: c.Query("type"), Query: c.Query("q")}
	rows, count, err := h.Service.ListShippingProfiles(c, f, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"shipping_profiles": rows, "count": count, "offset": offset, "limit": limit})
}
func (h *FulfillmentHTTP) getShippingProfile(c *gin.Context) {
	row, err := h.Service.GetShippingProfile(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"shipping_profile": row})
}
func (h *FulfillmentHTTP) createShippingProfile(c *gin.Context) {
	var in models.ShippingProfileInput
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	rows, err := flows.CreateShippingProfiles(c, h.Service, []models.ShippingProfileInput{in})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(rows) == 0 {
		c.JSON(500, gin.H{"type": "unexpected_state", "message": "create-shipping-profiles returned no result"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"shipping_profile": rows[0]})
}
func (h *FulfillmentHTTP) updateShippingProfile(c *gin.Context) {
	var in map[string]any
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := flows.UpdateShippingProfile(c, h.Service, c.Param("id"), in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"shipping_profile": row})
}
func (h *FulfillmentHTTP) deleteShippingProfile(c *gin.Context) {
	id := c.Param("id")
	if _, err := h.Service.GetShippingProfile(c, id); err != nil {
		writeModuleError(c, err)
		return
	}
	if err := flows.DeleteShippingProfile(c, h.Service, h.Links, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "shipping_profile", "deleted": true})
}
