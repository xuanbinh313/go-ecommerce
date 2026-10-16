package admin

import (
	"net/http"
	"strconv"

	flows "github.com/example/medusa-go-clone/internal/coreflows/product/workflows"
	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
	"github.com/gin-gonic/gin"
)

type CollectionHTTP struct {
	Service flows.ProductCollectionService
	Events  flows.EventPublisher
}

func (h *CollectionHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	a := router.Group("/admin", auth)
	a.GET("/collections", h.list)
	a.POST("/collections", h.create)
	a.GET("/collections/:id", h.get)
	a.POST("/collections/:id", h.update)
	a.DELETE("/collections/:id", h.delete)
	a.POST("/collections/:id/products", h.linkProducts)
}
func (h *CollectionHTTP) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	f := models.ProductCollectionFilter{IDs: queryValues(c, "id"), Titles: queryValues(c, "title"), Handles: queryValues(c, "handle"), ExternalIDs: queryValues(c, "external_id"), Query: c.Query("q")}
	rows, count, err := h.Service.ListProductCollections(c, f, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"collections": rows, "count": count, "offset": offset, "limit": limit})
}
func (h *CollectionHTTP) get(c *gin.Context) {
	row, err := h.Service.GetProductCollection(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"collection": row})
}
func (h *CollectionHTTP) create(c *gin.Context) {
	var in models.ProductCollectionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	rows, err := flows.CreateCollections(c, h.Service, h.Events, []models.ProductCollectionInput{in})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(rows) == 0 {
		c.JSON(500, gin.H{"type": "unexpected_state", "message": "create-collections returned no result"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"collection": rows[0]})
}
func (h *CollectionHTTP) update(c *gin.Context) {
	var in map[string]any
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if _, err := h.Service.GetProductCollection(c, c.Param("id")); err != nil {
		writeModuleError(c, err)
		return
	}
	row, err := flows.UpdateCollection(c, h.Service, h.Events, c.Param("id"), in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"collection": row})
}
func (h *CollectionHTTP) delete(c *gin.Context) {
	id := c.Param("id")
	if err := flows.DeleteCollections(c, h.Service, h.Events, []string{id}); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "collection", "deleted": true})
}
func (h *CollectionHTTP) linkProducts(c *gin.Context) {
	var in struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if err := flows.BatchLinkProductsToCollection(c, h.Service, h.Events, c.Param("id"), in.Add, in.Remove); err != nil {
		writeModuleError(c, err)
		return
	}
	row, err := h.Service.GetProductCollection(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"collection": row})
}
