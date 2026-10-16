package admin

import (
	"net/http"
	"strconv"
	"strings"

	orderflows "github.com/example/medusa-go-clone/internal/coreflows/order/workflows"
	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
	"github.com/gin-gonic/gin"
)

var claimListFields = []string{"id", "type", "order_id", "return_id", "display_id", "order_version", "refund_amount", "created_by", "created_at", "updated_at", "canceled_at"}
var claimDetailFields = []string{"id", "type", "order_id", "return_id", "display_id", "order_version", "refund_amount", "created_by", "created_at", "updated_at", "canceled_at", "additional_items", "claim_items"}

func (h *OrderHTTP) listClaims(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "15"))
	if err != nil || limit < 1 {
		limit = 15
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}
	requestedFields := c.Query("fields")
	f := models.ClaimFilter{IDs: queryValues(c, "id"), OrderIDs: queryValues(c, "order_id"), CreatedAt: paymentDateFilters(c, "created_at"), UpdatedAt: paymentDateFilters(c, "updated_at"), DeletedAt: paymentDateFilters(c, "deleted_at"), Limit: limit, Offset: offset, IncludeItems: strings.Contains(requestedFields, "claim_items") || strings.Contains(requestedFields, "additional_items") || strings.Contains(requestedFields, "*")}
	rows, count, err := h.Service.ListClaims(c, f)
	if err != nil {
		writeOrderError(c, err)
		return
	}
	selected := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		selected = append(selected, projectPaymentValue(row, c.Query("fields"), claimListFields))
	}
	c.JSON(http.StatusOK, gin.H{"claims": selected, "count": count, "offset": offset, "limit": limit})
}
func (h *OrderHTTP) getClaim(c *gin.Context) {
	row, err := h.Service.GetClaim(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"claim": projectPaymentValue(row, c.Query("fields"), claimDetailFields)})
}

func (h *OrderHTTP) createClaim(c *gin.Context) {
	var input models.CreateClaimInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	input.CreatedBy = actorID(c)
	claim, err := orderflows.BeginClaimOrder(c, h.Service, input)
	if err != nil {
		writeOrderError(c, err)
		return
	}
	order, err := h.Service.Get(c, input.OrderID, nil)
	if err != nil {
		writeOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": order, "claim": projectPaymentValue(claim, c.Query("fields"), claimDetailFields)})
}

func (h *OrderHTTP) deleteClaim(c *gin.Context) {
	id := c.Param("id")
	if err := orderflows.CancelBeginOrderClaim(c, h.Service, id); err != nil {
		writeOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "claim", "deleted": true})
}
