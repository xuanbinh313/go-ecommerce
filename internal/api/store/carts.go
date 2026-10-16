package store

import (
	"context"
	"net/http"

	flows "github.com/example/medusa-go-clone/internal/coreflows/cart/workflows"
	"github.com/example/medusa-go-clone/internal/modules/cart/domain/models"
	customermodels "github.com/example/medusa-go-clone/internal/modules/customer/domain/models"
	"github.com/gin-gonic/gin"
)

type CartService interface {
	Get(context.Context, string) (models.Cart, error)
	Create(context.Context, models.CreateInput) (models.Cart, error)
	Update(context.Context, string, models.UpdateInput) (models.Cart, error)
}
type CartCustomerService interface {
	Get(context.Context, string) (customermodels.Customer, error)
}
type CartEventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}
type CartHTTP struct {
	Service      CartService
	Customers    CartCustomerService
	Events       CartEventPublisher
	Authenticate gin.HandlerFunc
}

func (h *CartHTTP) Register(router *gin.RouterGroup) {
	post := router.POST
	if h.Authenticate != nil {
		post = func(path string, handlers ...gin.HandlerFunc) gin.IRoutes {
			return router.POST(path, append([]gin.HandlerFunc{h.Authenticate}, handlers...)...)
		}
	}
	post("/store/carts", h.create)
	if h.Authenticate != nil {
		router.POST("/store/carts/:id/customer", h.Authenticate, h.transferCustomer)
	} else {
		router.POST("/store/carts/:id/customer", h.transferCustomer)
	}
	router.GET("/store/carts/:id", h.get)
	router.POST("/store/carts/:id", h.update)
}
func (h *CartHTTP) transferCustomer(c *gin.Context) {
	actor, ok := c.Get("actor_id")
	customerID, valid := actor.(string)
	if !ok || !valid || customerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"type": "unauthorized", "message": "customer authentication is required"})
		return
	}
	var body struct {
		AdditionalData map[string]any `json:"additional_data"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if h.Customers == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "customer lookup is unavailable"})
		return
	}
	cart, err := flows.TransferCartCustomer(c, h.Service, h.Customers, h.Events, c.Param("id"), customerID)
	if err != nil {
		writeCartError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"cart": cart})
}
func (h *CartHTTP) create(c *gin.Context) {
	var in models.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if actor, ok := c.Get("actor_id"); ok {
		if id, ok := actor.(string); ok && id != "" {
			in.CustomerID = &id
		}
	}
	cart, err := flows.CreateCart(c, h.Service, in)
	if err != nil {
		writeCartError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"cart": cart})
}
func (h *CartHTTP) get(c *gin.Context) {
	cart, err := h.Service.Get(c, c.Param("id"))
	if err != nil {
		writeCartError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"cart": cart})
}
func (h *CartHTTP) update(c *gin.Context) {
	var in models.UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	cart, err := flows.UpdateCart(c, h.Service, c.Param("id"), in)
	if err != nil {
		writeCartError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"cart": cart})
}
func writeCartError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
}
