package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"

	flows "github.com/example/medusa-go-clone/internal/coreflows/refund-reason/workflows"
	"github.com/example/medusa-go-clone/internal/modules/payment/domain/models"
	"github.com/gin-gonic/gin"
)

var refundReasonFields = []string{"id", "label", "code", "description", "created_at", "updated_at", "deleted_at"}

func (h *PaymentHTTP) listRefundReasons(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	f := models.RefundReasonFilter{IDs: queryValues(c, "id"), Query: c.Query("q"), CreatedAt: paymentDateFilters(c, "created_at"), UpdatedAt: paymentDateFilters(c, "updated_at"), DeletedAt: paymentDateFilters(c, "deleted_at"), Limit: limit, Offset: offset}
	rows, count, err := h.Service.ListRefundReasons(c, f)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectPaymentValue(row, c.Query("fields"), refundReasonFields))
	}
	c.JSON(http.StatusOK, gin.H{"refund_reasons": items, "count": count, "offset": offset, "limit": limit})
}
func (h *PaymentHTTP) getRefundReason(c *gin.Context) {
	row, err := h.Service.GetRefundReason(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if row == nil {
		c.JSON(404, gin.H{"type": "not_found", "message": "Refund reason not found"})
		return
	}
	c.JSON(200, gin.H{"refund_reason": projectPaymentValue(row, c.Query("fields"), refundReasonFields)})
}
func (h *PaymentHTTP) createRefundReason(c *gin.Context) {
	var in models.RefundReasonInput
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	rows, err := flows.CreateRefundReasons(c, h.Service, []models.RefundReasonInput{in})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(rows) == 0 {
		c.JSON(500, gin.H{"type": "unexpected_state", "message": "create-refund-reason returned no result"})
		return
	}
	c.JSON(200, gin.H{"refund_reason": projectPaymentValue(rows[0], c.Query("fields"), refundReasonFields)})
}
func (h *PaymentHTTP) updateRefundReason(c *gin.Context) {
	old, err := h.Service.GetRefundReason(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if old == nil {
		c.JSON(404, gin.H{"type": "not_found", "message": "Refund reason not found"})
		return
	}
	var in struct {
		Label       *string         `json:"label"`
		Code        *string         `json:"code"`
		Description json.RawMessage `json:"description"`
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if in.Label != nil {
		old.Label = *in.Label
	}
	if in.Code != nil {
		old.Code = *in.Code
	}
	if in.Description != nil {
		if bytes.Equal(in.Description, []byte("null")) {
			old.Description = nil
		} else {
			var desc string
			if err = json.Unmarshal(in.Description, &desc); err != nil {
				c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
				return
			}
			old.Description = &desc
		}
	}
	rows, err := flows.UpdateRefundReasons(c, h.Service, []models.RefundReason{*old})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(rows) == 0 {
		c.JSON(404, gin.H{"type": "not_found", "message": "Refund reason not found"})
		return
	}
	c.JSON(200, gin.H{"refund_reason": projectPaymentValue(rows[0], c.Query("fields"), refundReasonFields)})
}
func (h *PaymentHTTP) deleteRefundReason(c *gin.Context) {
	id := c.Param("id")
	if err := flows.DeleteRefundReasons(c, h.Service, []string{id}); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "refund_reason", "deleted": true})
}
