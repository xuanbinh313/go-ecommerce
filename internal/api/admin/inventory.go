package admin

import (
	"context"
	"strconv"

	flows "github.com/example/medusa-go-clone/internal/coreflows/inventory/workflows"
	linkflows "github.com/example/medusa-go-clone/internal/coreflows/links/workflows"
	reservationflows "github.com/example/medusa-go-clone/internal/coreflows/reservation/workflows"
	"github.com/example/medusa-go-clone/internal/modules/inventory/domain/models"
	linkdomain "github.com/example/medusa-go-clone/internal/modules/links/domain"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	productmodels "github.com/example/medusa-go-clone/internal/modules/product/domain/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type InventoryService interface {
	ListItems(context.Context, models.Filter) ([]models.InventoryItem, int, error)
	GetItem(context.Context, string) (models.InventoryItem, error)
	CreateItem(context.Context, models.ItemInput) (models.InventoryItem, error)
	UpdateItem(context.Context, string, models.UpdateItemInput) (models.InventoryItem, error)
	DeleteItem(context.Context, string) error
	ListLevels(context.Context, string, []string, int, int) ([]models.InventoryLevel, int, error)
	CreateLevels(context.Context, []models.LevelInput) ([]models.InventoryLevel, error)
	UpdateLevels(context.Context, []models.LevelUpdate) ([]models.InventoryLevel, error)
	DeleteLevels(context.Context, []string, bool) ([]string, error)
	UpdateLevelAtLocation(context.Context, string, string, models.LevelUpdate) (models.InventoryItem, error)
	DeleteLevelAtLocation(context.Context, string, string) (string, error)
	ListReservations(context.Context, models.ReservationFilter) ([]models.ReservationItem, int, error)
	GetReservation(context.Context, string) (models.ReservationItem, error)
	CreateReservations(context.Context, []models.ReservationInput, *string) ([]models.ReservationItem, error)
	UpdateReservations(context.Context, []models.ReservationUpdate) ([]models.ReservationItem, error)
	DeleteReservations(context.Context, []string) error
	RestoreReservations(context.Context, []string) error
}
type InventoryHTTP struct {
	Service     InventoryService
	Links       linkdomain.Repository
	Products    ProductReader
	Events      reservationflows.EventPublisher
	Locks       reservationflows.LockExecutor
	Permissions UserRolesWorkflow
	RBACEnabled bool
}

func (h *InventoryHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	a := router.Group("/admin", auth)
	a.GET("/inventory-items", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "read"), h.listItems)
	a.POST("/inventory-items", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "create"), h.createItem)
	a.GET("/inventory-items/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "read"), h.getItem)
	a.POST("/inventory-items/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "read", "inventory_item", "update"), h.updateItem)
	a.DELETE("/inventory-items/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "read"), h.deleteItem)
	a.GET("/inventory-items/:id/location-levels", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "read"), h.listLevels)
	a.POST("/inventory-items/:id/location-levels", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "read", "inventory_level", "create"), h.createLevel)
	a.POST("/inventory-items/:id/location-levels/batch", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "read", "inventory_level", "read", "inventory_level", "*"), h.batchItemLevels)
	a.POST("/inventory-items/location-levels/batch", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "read", "inventory_item", "*"), h.batchLevels)
	a.POST("/inventory-items/:id/location-levels/:location_id", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "read", "inventory_level", "read", "inventory_level", "update"), h.updateLevelAtLocation)
	a.DELETE("/inventory-items/:id/location-levels/:location_id", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "read", "inventory_level", "read", "inventory_level", "delete"), h.deleteLevelAtLocation)
	a.GET("/reservations", permissionMiddleware(h.Permissions, h.RBACEnabled, "reservation_item", "read"), h.listReservations)
	a.POST("/reservations", permissionMiddleware(h.Permissions, h.RBACEnabled, "reservation_item", "create"), h.createReservation)
	a.GET("/reservations/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "reservation_item", "read"), h.getReservation)
	a.POST("/reservations/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "reservation_item", "read", "reservation_item", "update"), h.updateReservation)
	a.DELETE("/reservations/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "reservation_item", "read", "reservation_item", "delete"), h.deleteReservation)
	a.POST("/products/:id/variants/inventory-items/batch", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "create", "inventory_item", "update", "inventory_item", "delete"), h.batchVariantInventoryItems)
	a.POST("/products/:id/variants/:variant_id/inventory-items", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "create"), h.createVariantInventoryItem)
	a.POST("/products/:id/variants/:variant_id/inventory-items/:inventory_item_id", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "read", "inventory_item", "update"), h.updateVariantInventoryItem)
	a.DELETE("/products/:id/variants/:variant_id/inventory-items/:inventory_item_id", permissionMiddleware(h.Permissions, h.RBACEnabled, "inventory_item", "read", "inventory_item", "delete"), h.deleteVariantInventoryItem)
}

func (h *InventoryHTTP) listReservations(c *gin.Context) {
	limit, offset := 20, 0
	if v, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && v > 0 {
		limit = v
	}
	if v, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && v >= 0 {
		offset = v
	}
	filter := models.ReservationFilter{IDs: queryValues(c, "id"), InventoryItemIDs: queryValues(c, "inventory_item_id"), LocationIDs: queryValues(c, "location_id"), LineItemIDs: queryValues(c, "line_item_id"), CreatedBy: queryValues(c, "created_by"), Query: c.Query("q"), Description: c.Query("description"), Limit: limit, Offset: offset}
	rows, count, err := h.Service.ListReservations(c, filter)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, listResponse("reservations", rows, count, offset, limit))
}

func (h *InventoryHTTP) getReservation(c *gin.Context) {
	row, err := h.Service.GetReservation(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"reservation": row})
}

func (h *InventoryHTTP) createReservation(c *gin.Context) {
	var input models.ReservationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	created, err := reservationflows.CreateReservations(c, h.Service, h.Locks, h.Events, []models.ReservationInput{input}, nil)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(created) == 0 {
		writeModuleError(c, pgx.ErrNoRows)
		return
	}
	c.JSON(200, gin.H{"reservation": created[0]})
}

func (h *InventoryHTTP) updateReservation(c *gin.Context) {
	var input models.ReservationUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	input.ID = c.Param("id")
	updated, err := reservationflows.UpdateReservations(c, h.Service, h.Locks, h.Events, []models.ReservationUpdate{input})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(updated) == 0 {
		writeModuleError(c, pgx.ErrNoRows)
		return
	}
	c.JSON(200, gin.H{"reservation": updated[0]})
}

func (h *InventoryHTTP) deleteReservation(c *gin.Context) {
	id := c.Param("id")
	if err := reservationflows.DeleteReservations(c, h.Service, h.Locks, h.Events, []string{id}); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "reservation", "deleted": true})
}
func (h *InventoryHTTP) listItems(c *gin.Context) {
	limit, offset := 20, 0
	if v, e := strconv.Atoi(c.DefaultQuery("limit", "20")); e == nil && v > 0 {
		limit = v
	}
	if v, e := strconv.Atoi(c.DefaultQuery("offset", "0")); e == nil && v >= 0 {
		offset = v
	}
	locationIDs := queryValues(c, "location_levels.location_id")
	if len(locationIDs) == 0 {
		locationIDs = queryValues(c, "location_levels[location_id]")
	}
	f := models.Filter{IDs: queryValues(c, "id"), SKUs: queryValues(c, "sku"), OriginCountries: queryValues(c, "origin_country"), MidCodes: queryValues(c, "mid_code"), HSCodes: queryValues(c, "hs_code"), Materials: queryValues(c, "material"), LocationIDs: locationIDs, Query: c.Query("q"), Limit: limit, Offset: offset}
	if raw := c.Query("requires_shipping"); raw != "" {
		v, e := strconv.ParseBool(raw)
		if e != nil {
			c.JSON(400, gin.H{"type": "invalid_data", "message": "requires_shipping must be a boolean"})
			return
		}
		f.RequiresShipping = &v
	}
	rows, count, err := h.Service.ListItems(c, f)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, listResponse("inventory_items", rows, count, offset, limit))
}
func (h *InventoryHTTP) createItem(c *gin.Context) {
	var in models.ItemInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := flows.CreateInventoryItems(c, h.Service, in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"inventory_item": row})
}
func (h *InventoryHTTP) getItem(c *gin.Context) {
	row, err := h.Service.GetItem(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"inventory_item": row})
}
func (h *InventoryHTTP) updateItem(c *gin.Context) {
	var in models.UpdateItemInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := flows.UpdateInventoryItems(c, h.Service, c.Param("id"), in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"inventory_item": row})
}
func (h *InventoryHTTP) deleteItem(c *gin.Context) {
	id := c.Param("id")
	if err := flows.DeleteInventoryItem(c, h.Service, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "inventory_item", "deleted": true})
}
func (h *InventoryHTTP) listLevels(c *gin.Context) {
	limit, offset := 50, 0
	if v, e := strconv.Atoi(c.DefaultQuery("limit", "50")); e == nil && v > 0 {
		limit = v
	}
	if v, e := strconv.Atoi(c.DefaultQuery("offset", "0")); e == nil && v >= 0 {
		offset = v
	}
	rows, count, err := h.Service.ListLevels(c, c.Param("id"), queryValues(c, "location_id"), limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, listResponse("inventory_levels", rows, count, offset, limit))
}
func (h *InventoryHTTP) createLevel(c *gin.Context) {
	var in models.LevelInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	in.InventoryItemID = c.Param("id")
	if _, err := h.Service.CreateLevels(c, []models.LevelInput{in}); err != nil {
		writeModuleError(c, err)
		return
	}
	row, err := h.Service.GetItem(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"inventory_item": row})
}
func (h *InventoryHTTP) batchItemLevels(c *gin.Context) {
	var in models.BatchLevels
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	id := c.Param("id")
	for i := range in.Create {
		in.Create[i].InventoryItemID = id
	}
	for i := range in.Update {
		in.Update[i].InventoryItemID = id
	}
	out, err := h.runBatch(c, in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, out)
}
func (h *InventoryHTTP) batchLevels(c *gin.Context) {
	var in models.BatchLevels
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	out, err := h.runBatch(c, in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, out)
}
func (h *InventoryHTTP) runBatch(c *gin.Context, in models.BatchLevels) (gin.H, error) {
	created := []models.InventoryLevel{}
	updated := []models.InventoryLevel{}
	deleted := []string{}
	var err error
	if len(in.Create) > 0 {
		created, err = h.Service.CreateLevels(c, in.Create)
		if err != nil {
			return nil, err
		}
	}
	if len(in.Update) > 0 {
		updated, err = h.Service.UpdateLevels(c, in.Update)
		if err != nil {
			return nil, err
		}
	}
	if len(in.Delete) > 0 {
		deleted, err = h.Service.DeleteLevels(c, in.Delete, in.Force)
		if err != nil {
			return nil, err
		}
	}
	return gin.H{"created": created, "updated": updated, "deleted": deleted}, nil
}
func (h *InventoryHTTP) updateLevelAtLocation(c *gin.Context) {
	var in models.LevelUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := h.Service.UpdateLevelAtLocation(c, c.Param("id"), c.Param("location_id"), in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"inventory_item": row})
}
func (h *InventoryHTTP) deleteLevelAtLocation(c *gin.Context) {
	id, err := h.Service.DeleteLevelAtLocation(c, c.Param("id"), c.Param("location_id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	parent, err := h.Service.GetItem(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "inventory-level", "deleted": true, "parent": parent})
}

type variantInventoryLinkResponse struct {
	ID               string                `json:"id"`
	VariantID        string                `json:"variant_id"`
	InventoryItemID  string                `json:"inventory_item_id"`
	Inventory        *models.InventoryItem `json:"inventory,omitempty"`
	RequiredQuantity float64               `json:"required_quantity"`
}

type variantWithInventoryLinks struct {
	productmodels.ProductVariant
	InventoryItems []variantInventoryLinkResponse `json:"inventory_items"`
}

func (h *InventoryHTTP) variantWithLinks(c *gin.Context) (variantWithInventoryLinks, error) {
	variantID := c.Param("variant_id")
	variants, _, err := h.Products.ListProductVariants(c, c.Param("id"), productmodels.ProductVariantFilter{IDs: []string{variantID}}, 1, 0)
	if err != nil {
		return variantWithInventoryLinks{}, err
	}
	if len(variants) == 0 {
		return variantWithInventoryLinks{}, pgx.ErrNoRows
	}
	links, err := h.Links.List(c, inventoryLinkDefinition(), map[string]any{"variant_id": variantID})
	if err != nil {
		return variantWithInventoryLinks{}, err
	}
	items := make([]variantInventoryLinkResponse, 0, len(links))
	for _, link := range links {
		itemID, _ := link.Fields["inventory_item_id"].(string)
		item, err := h.Service.GetItem(c, itemID)
		if err != nil {
			return variantWithInventoryLinks{}, err
		}
		quantity, _ := link.Fields["required_quantity"].(float64)
		items = append(items, variantInventoryLinkResponse{ID: link.ID, VariantID: variantID, InventoryItemID: itemID, Inventory: &item, RequiredQuantity: quantity})
	}
	return variantWithInventoryLinks{ProductVariant: variants[0], InventoryItems: items}, nil
}

func inventoryLinkDefinition() linkmodels.Definition {
	for _, definition := range linkmodels.Definitions {
		if definition.Name == "ProductVariantInventoryItem" {
			return definition
		}
	}
	return linkmodels.Definition{}
}

func (h *InventoryHTTP) createVariantInventoryItem(c *gin.Context) {
	var input struct {
		InventoryItemID  string  `json:"inventory_item_id"`
		RequiredQuantity float64 `json:"required_quantity"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	_, err := linkflows.CreateLinks(c, h.Links, []linkflows.Definition{{Name: "ProductVariantInventoryItem", Fields: map[string]any{"variant_id": c.Param("variant_id"), "inventory_item_id": input.InventoryItemID, "required_quantity": input.RequiredQuantity}}})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	variant, err := h.variantWithLinks(c)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"variant": variant})
}

func (h *InventoryHTTP) updateVariantInventoryItem(c *gin.Context) {
	var input struct {
		RequiredQuantity float64 `json:"required_quantity"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	filters := map[string]any{"variant_id": c.Param("variant_id"), "inventory_item_id": c.Param("inventory_item_id")}
	_, err := linkflows.UpdateLinks(c, h.Links, []linkflows.Definition{{Name: "ProductVariantInventoryItem", Filters: filters, Data: map[string]any{"required_quantity": input.RequiredQuantity}}})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	variant, err := h.variantWithLinks(c)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"variant": variant})
}

func (h *InventoryHTTP) deleteVariantInventoryItem(c *gin.Context) {
	filters := map[string]any{"variant_id": c.Param("variant_id"), "inventory_item_id": c.Param("inventory_item_id")}
	deleted, err := linkflows.DismissLinks(c, h.Links, []linkflows.Definition{{Name: "ProductVariantInventoryItem", Filters: filters}})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	variant, err := h.variantWithLinks(c)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	var result any
	if len(deleted) > 0 {
		result = variantInventoryLinkResponse{ID: deleted[0].ID, VariantID: c.Param("variant_id"), InventoryItemID: c.Param("inventory_item_id"), RequiredQuantity: requiredQuantity(deleted[0])}
	}
	c.JSON(200, gin.H{"id": result, "object": "variant-inventory-item-link", "deleted": true, "parent": variant})
}

func requiredQuantity(link linkmodels.Link) float64 {
	value, _ := link.Fields["required_quantity"].(float64)
	return value
}

func (h *InventoryHTTP) batchVariantInventoryItems(c *gin.Context) {
	var input struct {
		Create []struct {
			VariantID        string  `json:"variant_id"`
			InventoryItemID  string  `json:"inventory_item_id"`
			RequiredQuantity float64 `json:"required_quantity"`
		} `json:"create"`
		Update []struct {
			VariantID        string  `json:"variant_id"`
			InventoryItemID  string  `json:"inventory_item_id"`
			RequiredQuantity float64 `json:"required_quantity"`
		} `json:"update"`
		Delete []struct {
			VariantID       string `json:"variant_id"`
			InventoryItemID string `json:"inventory_item_id"`
		} `json:"delete"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	const name = "ProductVariantInventoryItem"
	batch := linkflows.BatchInput{}
	for _, row := range input.Create {
		batch.Create = append(batch.Create, linkflows.Definition{Name: name, Fields: map[string]any{"variant_id": row.VariantID, "inventory_item_id": row.InventoryItemID, "required_quantity": row.RequiredQuantity}})
	}
	for _, row := range input.Update {
		batch.Update = append(batch.Update, linkflows.Definition{Name: name, Filters: map[string]any{"variant_id": row.VariantID, "inventory_item_id": row.InventoryItemID}, Data: map[string]any{"required_quantity": row.RequiredQuantity}})
	}
	for _, row := range input.Delete {
		batch.Delete = append(batch.Delete, linkflows.Definition{Name: name, Filters: map[string]any{"variant_id": row.VariantID, "inventory_item_id": row.InventoryItemID}})
	}
	out, err := linkflows.BatchLinks(c, h.Links, batch)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, out)
}
