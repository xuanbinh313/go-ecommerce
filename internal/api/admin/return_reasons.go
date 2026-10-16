package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	flows "github.com/example/medusa-go-clone/internal/coreflows/return-reason/workflows"
	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
	"github.com/gin-gonic/gin"
)

func (h *OrderHTTP) listReturnReasons(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	f := models.ReturnReasonFilter{IDs: queryValues(c, "id"), Values: queryValues(c, "value"), Labels: queryValues(c, "label"), Descriptions: queryValues(c, "description"), ParentIDs: queryValues(c, "parent_return_reason_id"), Query: c.Query("q")}
	rows, count, err := h.Service.ListReturnReasons(c, f, limit, offset)
	if err != nil {
		writeOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"return_reasons": rows, "count": count, "offset": offset, "limit": limit})
}
func (h *OrderHTTP) getReturnReason(c *gin.Context) {
	v, err := h.Service.GetReturnReason(c, c.Param("id"))
	if err != nil {
		writeOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"return_reason": v})
}
func (h *OrderHTTP) createReturnReason(c *gin.Context) {
	var in models.ReturnReasonInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if in.Metadata != nil && !json.Valid(in.Metadata) {
		c.JSON(400, gin.H{"type": "invalid_data", "message": "metadata must be a JSON object"})
		return
	}
	rows, err := flows.CreateReturnReasons(c, h.Service, []models.ReturnReasonInput{in})
	if err != nil {
		writeOrderError(c, err)
		return
	}
	if len(rows) == 0 {
		c.JSON(500, gin.H{"type": "unexpected_state", "message": "create-return-reasons returned no result"})
		return
	}
	row, err := h.Service.GetReturnReason(c, rows[0].ID)
	if err != nil {
		writeOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"return_reason": row})
}
func (h *OrderHTTP) updateReturnReason(c *gin.Context) {
	var in struct {
		Value       *string         `json:"value"`
		Label       *string         `json:"label"`
		Description json.RawMessage `json:"description"`
		Metadata    json.RawMessage `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if in.Value == nil || in.Label == nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": "value and label are required"})
		return
	}
	fields := map[string]any{"value": *in.Value, "label": *in.Label}
	if in.Description != nil {
		if string(in.Description) == "null" {
			fields["description"] = nil
		} else {
			var description string
			if err := json.Unmarshal(in.Description, &description); err != nil {
				c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
				return
			}
			fields["description"] = description
		}
	}
	if in.Metadata != nil {
		if !json.Valid(in.Metadata) {
			c.JSON(400, gin.H{"type": "invalid_data", "message": "metadata must be valid JSON"})
			return
		}
		if string(in.Metadata) == "null" {
			fields["metadata"] = nil
		} else {
			fields["metadata"] = in.Metadata
		}
	}
	rows, err := flows.UpdateReturnReasons(c, h.Service, c.Param("id"), fields)
	if err != nil {
		writeOrderError(c, err)
		return
	}
	if len(rows) == 0 {
		c.JSON(404, gin.H{"type": "not_found", "message": "Return reason not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"return_reason": rows[0]})
}
func (h *OrderHTTP) deleteReturnReason(c *gin.Context) {
	id := c.Param("id")
	if err := flows.DeleteReturnReasons(c, h.Service, []string{id}); err != nil {
		writeOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "return_reason", "deleted": true})
}
