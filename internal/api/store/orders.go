package store

import (
	"context"
	"net/http"
	"strconv"

	flows "github.com/example/medusa-go-clone/internal/coreflows/order/workflows"
	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
	"github.com/gin-gonic/gin"
)

type OrderReader interface {
	List(context.Context, models.Filter) (models.ListResult, error)
	Get(context.Context, string, *int) (models.Order, error)
}
type OrderHTTP struct {
	Service      OrderReader
	RequireActor gin.HandlerFunc
}

func (h *OrderHTTP) Register(router *gin.RouterGroup) {
	router.GET("/store/orders", h.RequireActor, h.list)
	router.GET("/store/orders/:id", h.get)
}
func (h *OrderHTTP) list(c *gin.Context) {
	actor, _ := c.Get("actor_id")
	customerID, ok := actor.(string)
	if !ok || customerID == "" {
		c.JSON(401, gin.H{"type": "unauthorized", "message": "customer authentication required"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	result, err := flows.GetOrdersList(c, h.Service, models.Filter{CustomerID: &customerID, IsDraft: boolPtr(false), IDs: c.QueryArray("id"), Limit: limit, Offset: offset, Order: c.Query("order")})
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": result.Orders, "count": result.Count, "offset": offset, "limit": limit})
}
func (h *OrderHTTP) get(c *gin.Context) {
	row, err := flows.GetOrderDetail(c, h.Service, c.Param("id"), nil)
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	if row.IsDraftOrder {
		c.JSON(404, gin.H{"type": "not_found", "message": "order not found"})
		return
	}
	c.JSON(200, gin.H{"order": row})
}
func boolPtr(v bool) *bool { return &v }
