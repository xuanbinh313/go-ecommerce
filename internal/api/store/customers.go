package store

import (
	"errors"
	"net/http"
	"strconv"

	customerflows "github.com/example/medusa-go-clone/internal/coreflows/customer/workflows"
	authdomain "github.com/example/medusa-go-clone/internal/modules/auth/domain"
	customerapp "github.com/example/medusa-go-clone/internal/modules/customer/application"
	customerdomain "github.com/example/medusa-go-clone/internal/modules/customer/domain"
	customermodels "github.com/example/medusa-go-clone/internal/modules/customer/domain/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type CustomerHTTP struct {
	Customers    customerapp.CustomerService
	Accounts     customerdomain.AccountService
	Auth         authdomain.IdentityMetadata
	Authenticate gin.HandlerFunc
	RequireActor gin.HandlerFunc
}

func (h *CustomerHTTP) Register(router *gin.RouterGroup) {
	router.POST("/store/customers", h.Authenticate, h.createAccount)
	router.GET("/store/customers/me", h.RequireActor, h.me)
	router.POST("/store/customers/me", h.RequireActor, h.updateMe)
	router.GET("/store/customers/me/addresses", h.RequireActor, h.listAddresses)
	router.POST("/store/customers/me/addresses", h.RequireActor, h.addAddress)
	router.GET("/store/customers/me/addresses/:address_id", h.RequireActor, h.getAddress)
	router.POST("/store/customers/me/addresses/:address_id", h.RequireActor, h.updateAddress)
	router.DELETE("/store/customers/me/addresses/:address_id", h.RequireActor, h.deleteAddress)
}
func (h *CustomerHTTP) createAccount(c *gin.Context) {
	if actor, _ := c.Get("actor_id"); actor != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request already authenticated as a customer."})
		return
	}
	identityID, _ := c.Get("auth_identity_id")
	actorType, _ := c.Get("actor_type")
	if actorType != "customer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "customer auth token required"})
		return
	}
	var customerData customermodels.Input
	if c.ShouldBindJSON(&customerData) != nil {
		c.JSON(400, gin.H{"error": "invalid customer payload"})
		return
	}
	created, err := customerflows.CreateCustomerAccount(c, h.Accounts, h.Auth, customerflows.CreateCustomerAccountInput{AuthIdentityID: identityID.(string), CustomerData: customerData})
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"customer": created})
}
func (h *CustomerHTTP) me(c *gin.Context) {
	id, _ := c.Get("actor_id")
	out, err := h.Customers.Get(c, id.(string))
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	c.JSON(200, gin.H{"customer": out})
}
func (h *CustomerHTTP) updateMe(c *gin.Context) {
	var in customermodels.Input
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid customer payload"})
		return
	}
	id, _ := c.Get("actor_id")
	out, err := h.Customers.Update(c, id.(string), in)
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	c.JSON(200, gin.H{"customer": out})
}
func (h *CustomerHTTP) listAddresses(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	id, _ := c.Get("actor_id")
	addresses, count, err := h.Customers.ListAddresses(c, id.(string), limit, offset)
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	c.JSON(200, gin.H{"addresses": addresses, "count": count, "offset": offset, "limit": limit})
}
func (h *CustomerHTTP) addAddress(c *gin.Context) {
	var in customermodels.AddressInput
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid address payload"})
		return
	}
	id, _ := c.Get("actor_id")
	out, err := h.Customers.CreateAddress(c, id.(string), in)
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": out})
}
func (h *CustomerHTTP) getAddress(c *gin.Context) {
	id, _ := c.Get("actor_id")
	address, err := h.Customers.GetAddress(c, id.(string), c.Param("address_id"))
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	c.JSON(200, gin.H{"address": address})
}
func (h *CustomerHTTP) updateAddress(c *gin.Context) {
	var in customermodels.AddressUpdate
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid address payload"})
		return
	}
	id, _ := c.Get("actor_id")
	record, err := h.Customers.UpdateAddress(c, id.(string), c.Param("address_id"), in)
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	c.JSON(200, gin.H{"customer": record})
}
func (h *CustomerHTTP) deleteAddress(c *gin.Context) {
	id, _ := c.Get("actor_id")
	customerID := id.(string)
	if err := h.Customers.DeleteAddress(c, customerID, c.Param("address_id")); err != nil {
		writeCustomerError(c, err)
		return
	}
	parent, err := h.Customers.Get(c, customerID)
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": customerID, "object": "address", "deleted": true, "parent": parent})
}
func writeCustomerError(c *gin.Context, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
		c.JSON(409, gin.H{"error": "duplicate customer data"})
		return
	}
	c.JSON(500, gin.H{"error": err.Error()})
}
