package admin

import (
	"context"
	"github.com/example/medusa-go-clone/internal/coreflows/tax/workflows"
	"github.com/example/medusa-go-clone/internal/modules/tax/domain/models"
	"github.com/gin-gonic/gin"
	"strconv"
)

type TaxService interface {
	ListProviders(context.Context, []string, *bool, int, int) ([]models.TaxProvider, int, error)
	ListRegions(context.Context, []string, []string, []string, int, int) ([]models.TaxRegion, int, error)
	GetRegion(context.Context, string) (models.TaxRegion, error)
	CreateRegion(context.Context, models.RegionInput) (models.TaxRegion, error)
	UpdateRegion(context.Context, string, models.RegionUpdate) (models.TaxRegion, error)
	DeleteRegion(context.Context, string) error
	ListRates(context.Context, []string, []string, []string, int, int) ([]models.TaxRate, int, error)
	GetRate(context.Context, string) (models.TaxRate, error)
	CreateRate(context.Context, models.RateInput, string, *string) (models.TaxRate, error)
	UpdateRate(context.Context, string, models.RateUpdate, *string) (models.TaxRate, error)
	DeleteRate(context.Context, string) error
	CreateRateRules(context.Context, []models.RuleCreateInput) error
	DeleteRateRules(context.Context, []string) error
}
type TaxHTTP struct {
	Service     TaxService
	Permissions UserRolesWorkflow
	RBACEnabled bool
}

func (h *TaxHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	a := router.Group("/admin", auth)
	a.GET("/tax-providers", permissionMiddleware(h.Permissions, h.RBACEnabled, "tax_provider", "read"), h.listProviders)
	a.GET("/tax-regions", permissionMiddleware(h.Permissions, h.RBACEnabled, "tax_region", "read"), h.listRegions)
	a.POST("/tax-regions", permissionMiddleware(h.Permissions, h.RBACEnabled, "tax_region", "create"), h.createRegion)
	a.GET("/tax-regions/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "tax_region", "read"), h.getRegion)
	a.POST("/tax-regions/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "tax_region", "read", "tax_region", "update"), h.updateRegion)
	a.DELETE("/tax-regions/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "tax_region", "read", "tax_region", "delete"), h.deleteRegion)
	a.GET("/tax-rates", permissionMiddleware(h.Permissions, h.RBACEnabled, "tax_rate", "read"), h.listRates)
	a.POST("/tax-rates", permissionMiddleware(h.Permissions, h.RBACEnabled, "tax_rate", "create"), h.createRate)
	a.GET("/tax-rates/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "tax_rate", "read"), h.getRate)
	a.POST("/tax-rates/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "tax_rate", "read", "tax_rate", "update"), h.updateRate)
	a.DELETE("/tax-rates/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "tax_rate", "read", "tax_rate", "delete"), h.deleteRate)
	a.POST("/tax-rates/:id/rules", permissionMiddleware(h.Permissions, h.RBACEnabled, "tax_rate", "read", "tax_rate", "update"), h.createRateRule)
	a.DELETE("/tax-rates/:id/rules/:rule_id", permissionMiddleware(h.Permissions, h.RBACEnabled, "tax_rate", "read", "tax_rate", "update"), h.deleteRateRule)
}

func (h *TaxHTTP) listProviders(c *gin.Context) {
	l, o := taxPagination(c)
	var enabled *bool
	if raw := c.Query("is_enabled"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(400, gin.H{"type": "invalid_data", "message": "is_enabled must be a boolean"})
			return
		}
		enabled = &v
	}
	rows, count, err := h.Service.ListProviders(c, queryValues(c, "id"), enabled, l, o)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"tax_providers": rows, "count": count, "offset": o, "limit": l})
}

func (h *TaxHTTP) createRateRule(c *gin.Context) {
	var input models.RuleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if err := workflows.CreateTaxRateRules(c, h.Service, []models.RuleCreateInput{{TaxRateID: c.Param("id"), Reference: input.Reference, ReferenceID: input.ReferenceID, Metadata: input.Metadata, CreatedBy: actorID(c)}}); err != nil {
		writeModuleError(c, err)
		return
	}
	rate, err := h.Service.GetRate(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"tax_rate": rate})
}

func (h *TaxHTTP) deleteRateRule(c *gin.Context) {
	ruleID := c.Param("rule_id")
	if err := workflows.DeleteTaxRateRules(c, h.Service, []string{ruleID}); err != nil {
		writeModuleError(c, err)
		return
	}
	rate, err := h.Service.GetRate(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": ruleID, "object": "tax_rate_rule", "deleted": true, "parent": rate})
}
func taxPagination(c *gin.Context) (int, int) {
	limit, offset := 20, 0
	if n, e := strconv.Atoi(c.DefaultQuery("limit", "20")); e == nil && n > 0 {
		limit = n
	}
	if n, e := strconv.Atoi(c.DefaultQuery("offset", "0")); e == nil && n >= 0 {
		offset = n
	}
	return limit, offset
}
func (h *TaxHTTP) listRegions(c *gin.Context) {
	l, o := taxPagination(c)
	v, n, e := h.Service.ListRegions(c, queryValues(c, "id"), queryValues(c, "country_code"), queryValues(c, "parent_id"), l, o)
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, listResponse("tax_regions", v, n, o, l))
}
func (h *TaxHTTP) getRegion(c *gin.Context) {
	v, e := h.Service.GetRegion(c, c.Param("id"))
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"tax_region": v})
}
func actorID(c *gin.Context) *string {
	if v, ok := c.Get("actor_id"); ok {
		if s, ok := v.(string); ok && s != "" {
			return &s
		}
	}
	return nil
}
func (h *TaxHTTP) createRegion(c *gin.Context) {
	var in models.RegionInput
	if e := c.ShouldBindJSON(&in); e != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": e.Error()})
		return
	}
	in.CreatedBy = actorID(c)
	v, e := workflows.CreateTaxRegion(c, h.Service, in)
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"tax_region": v})
}
func (h *TaxHTTP) updateRegion(c *gin.Context) {
	var in models.RegionUpdate
	if e := c.ShouldBindJSON(&in); e != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": e.Error()})
		return
	}
	v, e := workflows.UpdateTaxRegion(c, h.Service, c.Param("id"), in)
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"tax_region": v})
}
func (h *TaxHTTP) deleteRegion(c *gin.Context) {
	id := c.Param("id")
	if e := workflows.DeleteTaxRegion(c, h.Service, id); e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "tax_region", "deleted": true})
}
func (h *TaxHTTP) listRates(c *gin.Context) {
	l, o := taxPagination(c)
	v, n, e := h.Service.ListRates(c, queryValues(c, "id"), queryValues(c, "tax_region_id"), queryValues(c, "code"), l, o)
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, listResponse("tax_rates", v, n, o, l))
}
func (h *TaxHTTP) getRate(c *gin.Context) {
	v, e := h.Service.GetRate(c, c.Param("id"))
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"tax_rate": v})
}
func (h *TaxHTTP) createRate(c *gin.Context) {
	var in struct {
		models.RateInput
		TaxRegionID string `json:"tax_region_id"`
	}
	if e := c.ShouldBindJSON(&in); e != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": e.Error()})
		return
	}
	v, e := workflows.CreateTaxRate(c, h.Service, in.RateInput, in.TaxRegionID, actorID(c))
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"tax_rate": v})
}
func (h *TaxHTTP) updateRate(c *gin.Context) {
	var in models.RateUpdate
	if e := c.ShouldBindJSON(&in); e != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": e.Error()})
		return
	}
	v, e := workflows.UpdateTaxRate(c, h.Service, c.Param("id"), in, actorID(c))
	if e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"tax_rate": v})
}
func (h *TaxHTTP) deleteRate(c *gin.Context) {
	id := c.Param("id")
	if e := workflows.DeleteTaxRate(c, h.Service, id); e != nil {
		writeModuleError(c, e)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "tax_rate", "deleted": true})
}
