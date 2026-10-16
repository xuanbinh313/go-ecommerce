package admin

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	flows "github.com/example/medusa-go-clone/internal/coreflows/order/workflows"
	customerdomain "github.com/example/medusa-go-clone/internal/modules/customer/domain"
	eventmodels "github.com/example/medusa-go-clone/internal/modules/event-bus-local/domain/models"
	orderdomain "github.com/example/medusa-go-clone/internal/modules/order/domain"
	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
	"github.com/gin-gonic/gin"
)

type OrderReader interface {
	ListClaims(context.Context, models.ClaimFilter) ([]models.Claim, int, error)
	GetClaim(context.Context, string) (models.Claim, error)
	CreateClaim(context.Context, models.CreateClaimInput) (models.Claim, error)
	CancelBeginClaim(context.Context, string) error
	List(context.Context, models.Filter) (models.ListResult, error)
	Get(context.Context, string, *int) (models.Order, error)
	Archive(context.Context, []string) ([]models.Order, error)
	Complete(context.Context, []string) ([]models.Order, map[string]string, error)
	RestoreStatuses(context.Context, map[string]string) error
	Update(context.Context, models.UpdateInput) (models.Order, error)
	DeleteDraftOrders(context.Context, []string) error
	ListReturnReasons(context.Context, models.ReturnReasonFilter, int, int) ([]models.ReturnReason, int, error)
	GetReturnReason(context.Context, string) (models.ReturnReason, error)
	CreateReturnReasons(context.Context, []models.ReturnReasonInput) ([]models.ReturnReason, error)
	UpdateReturnReasons(context.Context, []string, map[string]any) ([]models.ReturnReason, error)
	SoftDeleteReturnReasons(context.Context, []string) error
	RestoreReturnReasons(context.Context, []string) error
	HardDeleteReturnReasons(context.Context, []string) error
}

type OrderEvents interface {
	Emit(context.Context, []eventmodels.Event, eventmodels.Options) error
	ReleaseGroupedEvents(context.Context, string) error
	ClearGroupedEvents(string, []string)
}

type OrderHTTP struct {
	Service            OrderReader
	Customers          customerdomain.OrderCustomerResolver
	Products           flows.ProductReader
	Translations       flows.TranslationReader
	TranslationEnabled bool
	Events             OrderEvents
	Permissions        UserRolesWorkflow
	RBACEnabled        bool
}

func (h *OrderHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	a := router.Group("/admin", auth)
	a.GET("/draft-orders", permissionMiddleware(h.Permissions, h.RBACEnabled, "order", "read"), h.listDraftOrders)
	a.GET("/draft-orders/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "order", "read"), h.getDraftOrder)
	a.DELETE("/draft-orders/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "order", "read"), h.deleteDraftOrder)
	a.GET("/orders", h.list)
	a.GET("/claims", permissionMiddleware(h.Permissions, h.RBACEnabled, "order_claim", "read"), h.listClaims)
	a.GET("/claims/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "order_claim", "read"), h.getClaim)
	a.POST("/claims", permissionMiddleware(h.Permissions, h.RBACEnabled, "order_claim", "create"), h.createClaim)
	a.DELETE("/claims/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "order_claim", "delete"), h.deleteClaim)
	a.GET("/orders/:id", h.get)
	a.POST("/orders/:id/archive", h.archive)
	a.POST("/orders/:id/complete", h.complete)
	a.POST("/orders/:id", h.update)
	a.GET("/return-reasons", h.listReturnReasons)
	a.POST("/return-reasons", h.createReturnReason)
	a.GET("/return-reasons/:id", h.getReturnReason)
	a.POST("/return-reasons/:id", h.updateReturnReason)
	a.DELETE("/return-reasons/:id", h.deleteReturnReason)
}
func (h *OrderHTTP) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	notDraft := false
	filter := models.Filter{IDs: queryValues(c, "id"), Status: queryValues(c, "status"), Query: c.Query("q"), CreatedAt: paymentDateFilters(c, "created_at"), UpdatedAt: paymentDateFilters(c, "updated_at"), Limit: limit, Offset: offset, IsDraft: &notDraft, Order: c.Query("order")}
	if id := c.Query("customer_id"); id != "" {
		filter.CustomerID = &id
	}
	if id := c.Query("region_id"); id != "" {
		filter.RegionID = &id
	}
	if id := c.Query("sales_channel_id"); id != "" {
		filter.SalesChannelID = &id
	}
	result, err := flows.GetOrdersList(c, h.Service, filter)
	if err != nil {
		writeOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": result.Orders, "count": result.Count, "offset": offset, "limit": limit})
}

func (h *OrderHTTP) listDraftOrders(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	isDraft := true
	f := models.Filter{IDs: queryValues(c, "id"), IsDraft: &isDraft, Query: c.Query("q"), CreatedAt: paymentDateFilters(c, "created_at"), UpdatedAt: paymentDateFilters(c, "updated_at"), Limit: limit, Offset: offset, Order: c.Query("order")}
	f.CustomerIDs = queryValues(c, "customer_id")
	f.RegionIDs = queryValues(c, "region_id")
	f.SalesChannelIDs = queryValues(c, "sales_channel_id")
	result, err := flows.GetOrdersList(c, h.Service, f)
	if err != nil {
		writeOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"draft_orders": result.Orders, "count": result.Count, "offset": offset, "limit": limit})
}

func (h *OrderHTTP) getDraftOrder(c *gin.Context) {
	var version *int
	if raw := c.Query("version"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(400, gin.H{"type": "invalid_data", "message": "version must be an integer"})
			return
		}
		version = &n
	}
	row, err := flows.GetOrderDetail(c, h.Service, c.Param("id"), version)
	if err != nil {
		writeOrderError(c, err)
		return
	}
	if !row.IsDraftOrder {
		c.JSON(http.StatusNotFound, gin.H{"type": "not_found", "message": "draft order not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"draft_order": row})
}

func (h *OrderHTTP) deleteDraftOrder(c *gin.Context) {
	id := c.Param("id")
	if err := flows.DeleteDraftOrders(c, h.Service, []string{id}); err != nil {
		writeOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "draft-order", "deleted": true})
}
func (h *OrderHTTP) get(c *gin.Context) {
	var version *int
	if raw := c.Query("version"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(400, gin.H{"type": "invalid_data", "message": "version must be an integer"})
			return
		}
		version = &n
	}
	row, err := flows.GetOrderDetail(c, h.Service, c.Param("id"), version)
	if err != nil {
		writeOrderError(c, err)
		return
	}
	c.JSON(200, gin.H{"order": row})
}

func (h *OrderHTTP) archive(c *gin.Context) {
	order, err := flows.ArchiveOrder(c, h.Service, h.Events, c.Param("id"))
	if err != nil {
		writeOrderError(c, err)
		return
	}
	c.JSON(200, gin.H{"order": order})
}

func (h *OrderHTTP) complete(c *gin.Context) {
	orders, err := flows.CompleteOrders(c, h.Service, h.Events, []string{c.Param("id")})
	if err != nil {
		writeOrderError(c, err)
		return
	}
	if len(orders) == 0 {
		c.JSON(404, gin.H{"type": "not_found", "message": "order not found"})
		return
	}
	c.JSON(200, gin.H{"order": orders[0]})
}

func (h *OrderHTTP) update(c *gin.Context) {
	var input models.UpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	input.ID = c.Param("id")
	if actor, ok := c.Get("actor_id"); ok {
		input.UserID, _ = actor.(string)
	}
	order, err := flows.UpdateOrder(c, h.Service, h.Customers, h.Products, h.Translations, h.TranslationEnabled, h.Events, input)
	if err != nil {
		writeOrderError(c, err)
		return
	}
	c.JSON(200, gin.H{"order": order})
}

func writeOrderError(c *gin.Context, err error) {
	var invalid *orderdomain.InvalidDataError
	if errors.As(err, &invalid) {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": invalid.Message})
		return
	}
	writeModuleError(c, err)
}
func boolQuery(c *gin.Context, key string) *bool {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return nil
	}
	return &v
}
