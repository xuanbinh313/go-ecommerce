package admin

import (
	"context"
	"strconv"

	priceListFlows "github.com/example/medusa-go-clone/internal/coreflows/price-list/workflows"
	flows "github.com/example/medusa-go-clone/internal/coreflows/pricing/workflows"
	"github.com/example/medusa-go-clone/internal/modules/pricing/domain/models"
	"github.com/gin-gonic/gin"
)

type PricingService interface {
	ListPreferences(context.Context, []string, []string, int, int) ([]models.PricePreference, int, error)
	GetPreference(context.Context, string) (models.PricePreference, error)
	CreatePreference(context.Context, models.PreferenceInput) (models.PricePreference, error)
	UpdatePreference(context.Context, string, models.PreferenceUpdate) (models.PricePreference, error)
	DeletePreference(context.Context, string) error
	ListPriceLists(context.Context, []string, []string, []string, int, int) ([]models.PriceList, int, error)
	GetPriceList(context.Context, string) (models.PriceList, error)
	CreatePriceList(context.Context, models.PriceList) (models.PriceList, error)
	UpdatePriceList(context.Context, string, models.PriceList) (models.PriceList, error)
	DeletePriceList(context.Context, string) error
}
type PricingHTTP struct{ Service PricingService }

func (h *PricingHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	a := router.Group("/admin", auth)
	a.GET("/price-preferences", h.listPreferences)
	a.POST("/price-preferences", h.createPreference)
	a.GET("/price-preferences/:id", h.getPreference)
	a.POST("/price-preferences/:id", h.updatePreference)
	a.DELETE("/price-preferences/:id", h.deletePreference)
	a.GET("/price-lists", h.listPriceLists)
	a.POST("/price-lists", h.createPriceList)
	a.GET("/price-lists/:id", h.getPriceList)
	a.POST("/price-lists/:id", h.updatePriceList)
	a.DELETE("/price-lists/:id", h.deletePriceList)
}

func (h *PricingHTTP) listPriceLists(c *gin.Context) {
	limit, offset := 20, 0
	if n, e := strconv.Atoi(c.DefaultQuery("limit", "20")); e == nil && n > 0 {
		limit = n
	}
	if n, e := strconv.Atoi(c.DefaultQuery("offset", "0")); e == nil && n >= 0 {
		offset = n
	}
	rows, count, err := h.Service.ListPriceLists(c, queryValues(c, "id"), queryValues(c, "status"), queryValues(c, "type"), limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, listResponse("price_lists", rows, count, offset, limit))
}
func (h *PricingHTTP) getPriceList(c *gin.Context) {
	v, e := h.Service.GetPriceList(c, c.Param("id"))
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"price_list": v})
}
func (h *PricingHTTP) createPriceList(c *gin.Context) {
	var in models.PriceList
	if e := c.ShouldBindJSON(&in); e != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": e.Error()})
		return
	}
	v, e := priceListFlows.CreatePriceList(c, h.Service, in)
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"price_list": v})
}
func (h *PricingHTTP) updatePriceList(c *gin.Context) {
	var in models.PriceList
	if e := c.ShouldBindJSON(&in); e != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": e.Error()})
		return
	}
	v, e := priceListFlows.UpdatePriceList(c, h.Service, c.Param("id"), in)
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"price_list": v})
}
func (h *PricingHTTP) deletePriceList(c *gin.Context) {
	id := c.Param("id")
	if e := priceListFlows.DeletePriceList(c, h.Service, id); e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "price_list", "deleted": true})
}
func (h *PricingHTTP) listPreferences(c *gin.Context) {
	limit, offset := 300, 0
	if n, e := strconv.Atoi(c.DefaultQuery("limit", "300")); e == nil && n > 0 {
		limit = n
	}
	if n, e := strconv.Atoi(c.DefaultQuery("offset", "0")); e == nil && n >= 0 {
		offset = n
	}
	rows, count, err := h.Service.ListPreferences(c, queryValues(c, "attribute"), queryValues(c, "value"), limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, listResponse("price_preferences", rows, count, offset, limit))
}
func (h *PricingHTTP) getPreference(c *gin.Context) {
	v, e := h.Service.GetPreference(c, c.Param("id"))
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"price_preference": v})
}
func (h *PricingHTTP) createPreference(c *gin.Context) {
	var in models.PreferenceInput
	if e := c.ShouldBindJSON(&in); e != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": e.Error()})
		return
	}
	v, e := flows.CreatePricePreference(c, h.Service, in)
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"price_preference": v})
}
func (h *PricingHTTP) updatePreference(c *gin.Context) {
	var in models.PreferenceUpdate
	if e := c.ShouldBindJSON(&in); e != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": e.Error()})
		return
	}
	v, e := flows.UpdatePricePreference(c, h.Service, c.Param("id"), in)
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"price_preference": v})
}
func (h *PricingHTTP) deletePreference(c *gin.Context) {
	id := c.Param("id")
	if e := flows.DeletePricePreference(c, h.Service, id); e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "price_preference", "deleted": true})
}
