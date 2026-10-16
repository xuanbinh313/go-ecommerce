package admin

import (
	"context"
	"github.com/example/medusa-go-clone/internal/coreflows/stock-location/workflows"
	"github.com/example/medusa-go-clone/internal/modules/stock-location/domain/models"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type StockLocationService interface {
	List(context.Context, models.Filter) ([]models.StockLocation, int, error)
	Get(context.Context, string) (models.StockLocation, error)
	Create(context.Context, models.Input) (models.StockLocation, error)
	Update(context.Context, string, models.UpdateInput) (models.StockLocation, error)
	Delete(context.Context, string) error
	LinkSalesChannels(context.Context, string, []string, []string) error
}
type StockLocationHTTP struct{ Service StockLocationService }

func (h *StockLocationHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	a := router.Group("/admin", auth)
	a.GET("/stock-locations", h.list)
	a.POST("/stock-locations", h.create)
	a.GET("/stock-locations/:id", h.get)
	a.POST("/stock-locations/:id", h.update)
	a.DELETE("/stock-locations/:id", h.delete)
	a.POST("/stock-locations/:id/sales-channels", h.linkSalesChannels)
}
func (h *StockLocationHTTP) list(c *gin.Context) {
	limit, offset := 20, 0
	if n, e := strconv.Atoi(c.DefaultQuery("limit", "20")); e == nil && n > 0 {
		limit = n
	}
	if n, e := strconv.Atoi(c.DefaultQuery("offset", "0")); e == nil && n >= 0 {
		offset = n
	}
	f := models.Filter{IDs: queryValues(c, "id"), Names: queryValues(c, "name"), AddressIDs: queryValues(c, "address_id"), SalesChannelIDs: queryValues(c, "sales_channel_id"), Query: c.Query("q"), Limit: limit, Offset: offset}
	rows, count, err := h.Service.List(c, f)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listResponse("stock_locations", rows, count, offset, limit))
}
func (h *StockLocationHTTP) create(c *gin.Context) {
	var in models.Input
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := workflows.CreateStockLocations(c, h.Service, in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"stock_location": row})
}
func (h *StockLocationHTTP) get(c *gin.Context) {
	row, err := h.Service.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"stock_location": row})
}
func (h *StockLocationHTTP) update(c *gin.Context) {
	var in models.UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := workflows.UpdateStockLocations(c, h.Service, c.Param("id"), in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"stock_location": row})
}
func (h *StockLocationHTTP) delete(c *gin.Context) {
	id := c.Param("id")
	if err := workflows.DeleteStockLocations(c, h.Service, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "stock_location", "deleted": true})
}
func (h *StockLocationHTTP) linkSalesChannels(c *gin.Context) {
	var in struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if err := workflows.LinkSalesChannelsToStockLocation(c, h.Service, c.Param("id"), in.Add, in.Remove); err != nil {
		writeModuleError(c, err)
		return
	}
	row, err := h.Service.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"stock_location": row})
}
