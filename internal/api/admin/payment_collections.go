package admin

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"

	paymentflows "github.com/example/medusa-go-clone/internal/coreflows/payment/workflows"
	linkModels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	orderModels "github.com/example/medusa-go-clone/internal/modules/order/domain/models"
	paymentModels "github.com/example/medusa-go-clone/internal/modules/payment/domain/models"
	"github.com/gin-gonic/gin"
)

type PaymentOrderReader interface {
	Get(context.Context, string, *int) (orderModels.Order, error)
}
type PaymentCollectionService interface {
	CreatePaymentCollections(context.Context, []paymentModels.CreatePaymentCollectionInput) ([]paymentModels.PaymentCollection, error)
	DeletePaymentCollections(context.Context, []string) error
	GetPaymentCollection(context.Context, string) (*paymentModels.PaymentCollection, error)
}
type PaymentLinkService interface {
	Create(context.Context, string, []map[string]any) ([]linkModels.Link, error)
	Dismiss(context.Context, string, []map[string]any) ([]linkModels.Link, error)
	Restore(context.Context, string, []map[string]any) ([]linkModels.Link, error)
}

type PaymentCollectionHTTP struct {
	Orders   PaymentOrderReader
	Payments PaymentCollectionService
	Links    PaymentLinkService
}

func (h *PaymentCollectionHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	a := router.Group("/admin", auth)
	a.POST("/payment-collections", h.create)
	a.DELETE("/payment-collections/:id", h.delete)
}

func (h *PaymentCollectionHTTP) delete(c *gin.Context) {
	if err := paymentflows.DeleteOrderPaymentCollection(c, h.Payments, h.Links, c.Param("id")); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": c.Param("id"), "object": "payment-collection", "deleted": true})
}

func (h *PaymentCollectionHTTP) create(c *gin.Context) {
	var request struct {
		OrderID string          `json:"order_id"`
		Amount  json.RawMessage `json:"amount"`
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		paymentCollectionError(c, http.StatusBadRequest, err)
		return
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		paymentCollectionError(c, http.StatusBadRequest, io.ErrUnexpectedEOF)
		return
	}
	if request.OrderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "order_id is required"})
		return
	}
	var numeric json.Number
	if err := json.Unmarshal(request.Amount, &numeric); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "amount must be a number"})
		return
	}
	value, parseErr := strconv.ParseFloat(numeric.String(), 64)
	if parseErr != nil || math.IsInf(value, 0) || math.IsNaN(value) || value < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "amount must be a non-negative number"})
		return
	}
	collections, err := paymentflows.CreateOrderPaymentCollection(c, h.Orders, h.Payments, h.Links, paymentflows.CreateOrderPaymentCollectionInput{OrderID: request.OrderID, Amount: paymentModels.Amount(numeric.String())})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(collections) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"type": "internal_error", "message": "payment module created no payment collection"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"payment_collection": collections[0]})
}

func paymentCollectionError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{"type": "invalid_data", "message": err.Error()})
}
