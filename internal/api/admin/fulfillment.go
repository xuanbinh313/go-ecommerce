package admin

import (
	"context"
	"net/http"

	fulfillmentflows "github.com/example/medusa-go-clone/internal/coreflows/fulfillment/workflows"
	"github.com/example/medusa-go-clone/internal/coreflows/stock-location/workflows"
	fulfillmentmodels "github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	stockmodels "github.com/example/medusa-go-clone/internal/modules/stock-location/domain/models"
	"github.com/gin-gonic/gin"
)

type FulfillmentService interface {
	ListShippingOptions(context.Context, fulfillmentmodels.ShippingOptionFilter, int, int) ([]fulfillmentmodels.ShippingOption, int, error)
	GetShippingOption(context.Context, string) (fulfillmentmodels.ShippingOption, error)
	CreateShippingOption(context.Context, fulfillmentmodels.ShippingOptionInput) (fulfillmentmodels.ShippingOption, error)
	UpdateShippingOption(context.Context, string, fulfillmentmodels.ShippingOptionUpdateInput) (fulfillmentmodels.ShippingOption, error)
	HardDeleteShippingOption(context.Context, string) error
	SoftDeleteShippingOptions(context.Context, []string) error
	RestoreShippingOptions(context.Context, []string) error
	BatchShippingOptionRules(context.Context, string, fulfillmentmodels.ShippingOptionRuleBatchInput) (fulfillmentmodels.ShippingOptionRuleBatchResult, error)
	CreateFulfillmentRecord(context.Context, fulfillmentmodels.FulfillmentCreateInput) (fulfillmentmodels.FulfillmentRecord, error)
	GetFulfillment(context.Context, string) (fulfillmentmodels.FulfillmentRecord, error)
	CancelFulfillmentRecord(context.Context, string) (fulfillmentmodels.FulfillmentRecord, error)
	CreateFulfillmentShipment(context.Context, string, fulfillmentmodels.CreateShipmentInput) (fulfillmentmodels.FulfillmentRecord, fulfillmentmodels.FulfillmentRecord, error)
	RestoreFulfillment(context.Context, fulfillmentmodels.FulfillmentRecord) error
	HardDeleteFulfillment(context.Context, string) error
	ListFulfillmentProviders() ([]fulfillmentmodels.FulfillmentProvider, error)
	GetFulfillmentOptions(context.Context, string) ([]fulfillmentmodels.FulfillmentOption, error)
	CreateSet(context.Context, fulfillmentmodels.CreateSetInput) (fulfillmentmodels.FulfillmentSet, error)
	ListShippingProfiles(context.Context, fulfillmentmodels.ShippingProfileFilter, int, int) ([]fulfillmentmodels.ShippingProfile, int, error)
	GetShippingProfile(context.Context, string) (fulfillmentmodels.ShippingProfile, error)
	CreateShippingProfiles(context.Context, []fulfillmentmodels.ShippingProfileInput) ([]fulfillmentmodels.ShippingProfile, error)
	UpdateShippingProfile(context.Context, string, map[string]any) (fulfillmentmodels.ShippingProfile, error)
	SoftDeleteShippingProfiles(context.Context, []string) error
	RestoreShippingProfiles(context.Context, []string) error
	HardDeleteShippingProfiles(context.Context, []string) error
	ListShippingOptionTypes(context.Context, fulfillmentmodels.ShippingOptionTypeFilter, int, int) ([]fulfillmentmodels.ShippingOptionType, int, error)
	GetShippingOptionType(context.Context, string) (fulfillmentmodels.ShippingOptionType, error)
	CreateShippingOptionTypes(context.Context, []fulfillmentmodels.ShippingOptionTypeInput) ([]fulfillmentmodels.ShippingOptionType, error)
	UpdateShippingOptionType(context.Context, string, fulfillmentmodels.ShippingOptionTypeUpdate) (fulfillmentmodels.ShippingOptionType, error)
	SoftDeleteShippingOptionTypes(context.Context, []string) error
	RestoreShippingOptionTypes(context.Context, []string) error
	HardDeleteShippingOptionTypes(context.Context, []string) error
	ShippingOptionTypeInUse(context.Context, string) (bool, error)
	GetSet(context.Context, string) (fulfillmentmodels.FulfillmentSet, error)
	DeleteSet(context.Context, string) error
	RestoreSet(context.Context, string) error
	CreateServiceZone(context.Context, string, fulfillmentmodels.ServiceZoneInput) (fulfillmentmodels.FulfillmentSet, error)
	GetServiceZone(context.Context, string) (fulfillmentmodels.ServiceZone, error)
	UpdateServiceZone(context.Context, string, fulfillmentmodels.ServiceZoneUpdate) (fulfillmentmodels.FulfillmentSet, error)
	DeleteServiceZone(context.Context, string) (fulfillmentmodels.FulfillmentSet, error)
}
type FulfillmentStockLocations interface {
	Get(context.Context, string) (stockmodels.StockLocation, error)
}
type FulfillmentLinks interface {
	Create(context.Context, string, []map[string]any) ([]linkmodels.Link, error)
	Dismiss(context.Context, string, []map[string]any) ([]linkmodels.Link, error)
	Restore(context.Context, string, []map[string]any) ([]linkmodels.Link, error)
	List(context.Context, string, map[string]any) ([]linkmodels.Link, error)
}
type FulfillmentEvents interface {
	Publish(context.Context, string, []map[string]any) error
}

type FulfillmentHTTP struct {
	Service     FulfillmentService
	Locations   FulfillmentStockLocations
	Links       FulfillmentLinks
	Permissions UserRolesWorkflow
	RBACEnabled bool
	Events      FulfillmentEvents
}

func (h *FulfillmentHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	a := router.Group("/admin", auth)
	a.POST("/stock-locations/:id/fulfillment-sets", permissionMiddleware(h.Permissions, h.RBACEnabled, "stock_location", "update"), h.createLocationSet)
	a.POST("/stock-locations/:id/fulfillment-providers", permissionMiddleware(h.Permissions, h.RBACEnabled, "stock_location", "update"), h.linkFulfillmentProviders)
	a.DELETE("/fulfillment-sets/:id", h.deleteSet)
	a.POST("/fulfillment-sets/:id/service-zones", h.createZone)
	a.GET("/fulfillment-sets/:id/service-zones/:zone_id", h.getZone)
	a.POST("/fulfillment-sets/:id/service-zones/:zone_id", h.updateZone)
	a.DELETE("/fulfillment-sets/:id/service-zones/:zone_id", h.deleteZone)
	a.GET("/shipping-profiles", h.listShippingProfiles)
	a.POST("/shipping-profiles", h.createShippingProfile)
	a.GET("/shipping-profiles/:id", h.getShippingProfile)
	a.POST("/shipping-profiles/:id", h.updateShippingProfile)
	a.DELETE("/shipping-profiles/:id", h.deleteShippingProfile)
	a.GET("/shipping-options", permissionMiddleware(h.Permissions, h.RBACEnabled, "shipping_option", "read"), h.listShippingOptions)
	a.POST("/shipping-options", permissionMiddleware(h.Permissions, h.RBACEnabled, "shipping_option", "create"), h.createShippingOption)
	a.POST("/shipping-options/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "shipping_option", "update"), h.updateShippingOption)
	a.GET("/shipping-options/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "shipping_option", "read"), h.getShippingOption)
	a.DELETE("/shipping-options/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "shipping_option", "delete"), h.deleteShippingOption)
	a.POST("/shipping-options/:id/rules/batch", permissionMiddleware(h.Permissions, h.RBACEnabled, "shipping_option", "update"), h.batchShippingOptionRules)
	a.GET("/shipping-option-types", h.listShippingOptionTypes)
	a.POST("/shipping-option-types", h.createShippingOptionType)
	a.GET("/shipping-option-types/:id", h.getShippingOptionType)
	a.POST("/shipping-option-types/:id", h.updateShippingOptionType)
	a.DELETE("/shipping-option-types/:id", h.deleteShippingOptionType)
	a.GET("/fulfillment-providers", h.listProviders)
	a.GET("/fulfillment-providers/:id/options", h.providerOptions)
	a.POST("/fulfillments", permissionMiddleware(h.Permissions, h.RBACEnabled, "fulfillment", "create"), h.createFulfillment)
	a.POST("/fulfillments/:id/cancel", permissionMiddleware(h.Permissions, h.RBACEnabled, "fulfillment", "update"), h.cancelFulfillment)
	a.POST("/fulfillments/:id/shipment", permissionMiddleware(h.Permissions, h.RBACEnabled, "fulfillment", "update"), h.createShipment)
}

func (h *FulfillmentHTTP) createLocationSet(c *gin.Context) {
	locationID := c.Param("id")
	if _, err := h.Locations.Get(c, locationID); err != nil {
		writeModuleError(c, err)
		return
	}
	var in fulfillmentmodels.CreateSetInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	_, err := workflows.CreateLocationFulfillmentSet(c, h.Service, h.Links, locationID, in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	location, err := h.Locations.Get(c, locationID)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"stock_location": location})
}

func (h *FulfillmentHTTP) linkFulfillmentProviders(c *gin.Context) {
	locationID := c.Param("id")
	if _, err := h.Locations.Get(c, locationID); err != nil {
		writeModuleError(c, err)
		return
	}
	var in struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	create := make([]map[string]any, 0, len(in.Add))
	remove := make([]map[string]any, 0, len(in.Remove))
	for _, id := range in.Add {
		create = append(create, map[string]any{"stock_location_id": locationID, "fulfillment_provider_id": id})
	}
	for _, id := range in.Remove {
		remove = append(remove, map[string]any{"stock_location_id": locationID, "fulfillment_provider_id": id})
	}
	// Match batchLinksWorkflow's add/remove behavior and restore additions if
	// dismissing existing links fails.
	created, err := h.Links.Create(c, "LocationFulfillmentProvider", create)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(remove) > 0 {
		if _, err = h.Links.Dismiss(c, "LocationFulfillmentProvider", remove); err != nil {
			rollback := make([]map[string]any, 0, len(created))
			for _, link := range created {
				rollback = append(rollback, link.Fields)
			}
			if len(rollback) > 0 {
				_, _ = h.Links.Dismiss(c, "LocationFulfillmentProvider", rollback)
			}
			writeModuleError(c, err)
			return
		}
	}
	location, err := h.Locations.Get(c, locationID)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"stock_location": location})
}
func (h *FulfillmentHTTP) deleteSet(c *gin.Context) {
	id := c.Param("id")
	if err := fulfillmentflows.DeleteFulfillmentSets(c, h.Service, h.Links, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "fulfillment_set", "deleted": true})
}
func (h *FulfillmentHTTP) createZone(c *gin.Context) {
	var in fulfillmentmodels.ServiceZoneInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := fulfillmentflows.CreateServiceZones(c, h.Service, c.Param("id"), in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"fulfillment_set": row})
}
func (h *FulfillmentHTTP) getZone(c *gin.Context) {
	row, err := h.Service.GetServiceZone(c, c.Param("zone_id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"service_zone": row})
}
func (h *FulfillmentHTTP) updateZone(c *gin.Context) {
	var in fulfillmentmodels.ServiceZoneUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	zone, err := h.Service.GetServiceZone(c, c.Param("zone_id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if zone.FulfillmentSetID != c.Param("id") {
		c.JSON(404, gin.H{"type": "not_found", "message": "Service zone not found on fulfillment set"})
		return
	}
	row, err := fulfillmentflows.UpdateServiceZones(c, h.Service, c.Param("zone_id"), in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"fulfillment_set": row})
}
func (h *FulfillmentHTTP) deleteZone(c *gin.Context) {
	zone, err := h.Service.GetServiceZone(c, c.Param("zone_id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if zone.FulfillmentSetID != c.Param("id") {
		c.JSON(404, gin.H{"type": "not_found", "message": "Service zone not found on fulfillment set"})
		return
	}
	parent, err := h.Service.GetSet(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	_, err = fulfillmentflows.DeleteServiceZones(c, h.Service, c.Param("zone_id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": c.Param("zone_id"), "object": "service_zone", "deleted": true, "parent": parent})
}
