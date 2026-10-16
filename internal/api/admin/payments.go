package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	paymentModels "github.com/example/medusa-go-clone/internal/modules/payment/domain/models"
	"github.com/gin-gonic/gin"
)

type PaymentReader interface {
	ListRefundReasons(context.Context, paymentModels.RefundReasonFilter) ([]paymentModels.RefundReason, int64, error)
	GetRefundReason(context.Context, string) (*paymentModels.RefundReason, error)
	CreateRefundReasons(context.Context, []paymentModels.RefundReasonInput) ([]paymentModels.RefundReason, error)
	UpdateRefundReasons(context.Context, []paymentModels.RefundReason) ([]paymentModels.RefundReason, error)
	SoftDeleteRefundReasons(context.Context, []string) error
	RestoreRefundReasons(context.Context, []string) error
	DeleteRefundReasons(context.Context, []string) error
	ListPayments(context.Context, paymentModels.PaymentFilter) ([]paymentModels.Payment, int64, error)
	GetPayment(context.Context, string) (*paymentModels.Payment, error)
	ListPaymentProviders(context.Context, paymentModels.PaymentProviderFilter) ([]paymentModels.PaymentProvider, int64, error)
}

type PaymentHTTP struct{ Service PaymentReader }

func (h *PaymentHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	a := router.Group("/admin", auth)
	a.GET("/payments", h.list)
	a.GET("/payments/payment-providers", h.providers)
	a.GET("/refund-reasons", h.listRefundReasons)
	a.POST("/refund-reasons", h.createRefundReason)
	a.GET("/refund-reasons/:id", h.getRefundReason)
	a.POST("/refund-reasons/:id", h.updateRefundReason)
	a.DELETE("/refund-reasons/:id", h.deleteRefundReason)
	a.GET("/payments/:id", h.get)
}

func (h *PaymentHTTP) list(c *gin.Context) {
	limit, offset := paymentPagination(c)
	rows, count, err := h.Service.ListPayments(c, paymentModels.PaymentFilter{IDs: queryValues(c, "id"), PaymentSessionIDs: queryValues(c, "payment_session_id"), Query: c.Query("q"), CreatedAt: paymentDateFilters(c, "created_at"), UpdatedAt: paymentDateFilters(c, "updated_at"), DeletedAt: paymentDateFilters(c, "deleted_at"), Limit: limit, Offset: offset})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	selected := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		selected = append(selected, projectPayment(row, c.Query("fields")))
	}
	c.JSON(http.StatusOK, gin.H{"payments": selected, "count": count, "offset": offset, "limit": limit})
}

func (h *PaymentHTTP) get(c *gin.Context) {
	row, err := h.Service.GetPayment(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if row == nil {
		c.JSON(http.StatusNotFound, gin.H{"type": "not_found", "message": "Payment not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"payment": projectPayment(*row, c.Query("fields"))})
}

func (h *PaymentHTTP) providers(c *gin.Context) {
	limit, offset := paymentPagination(c)
	var enabled *bool
	if raw := c.Query("is_enabled"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(400, gin.H{"type": "invalid_data", "message": "is_enabled must be a boolean"})
			return
		}
		enabled = &v
	}
	rows, count, err := h.Service.ListPaymentProviders(c, paymentModels.PaymentProviderFilter{IDs: queryValues(c, "id"), IsEnabled: enabled, Limit: limit, Offset: offset})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	selected := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		selected = append(selected, projectPaymentProvider(row, c.Query("fields")))
	}
	c.JSON(http.StatusOK, gin.H{"payment_providers": selected, "count": count, "offset": offset, "limit": limit})
}

var defaultPaymentFields = []string{"id", "currency_code", "amount", "captured_at", "payment_collection_id", "payment_session_id", "captures.id", "captures.amount", "refunds.id", "refunds.amount", "refunds.note", "refunds.payment_id", "refunds.refund_reason.label", "refunds.refund_reason.code"}

func projectPayment(row paymentModels.Payment, requested string) map[string]any {
	return projectPaymentValue(row, requested, defaultPaymentFields)
}
func projectPaymentProvider(row paymentModels.PaymentProvider, requested string) map[string]any {
	return projectPaymentValue(row, requested, []string{"id", "is_enabled"})
}
func projectPaymentValue(row any, requested string, defaults []string) map[string]any {
	encoded, _ := json.Marshal(row)
	var source map[string]any
	_ = json.Unmarshal(encoded, &source)
	fields := defaults
	if requested != "" {
		fields = strings.Split(requested, ",")
	}
	all := false
	includeDefaults := requested == ""
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "*" {
			all = true
		}
		if strings.HasPrefix(field, "+") || strings.HasPrefix(field, "-") {
			includeDefaults = true
		}
	}
	out := map[string]any{}
	if all {
		return source
	}
	if includeDefaults {
		for _, field := range defaults {
			copyPaymentField(out, source, field)
		}
	}
	if requested != "" {
		for _, field := range fields {
			field = strings.TrimSpace(field)
			if field == "*" {
				continue
			}
			if strings.HasPrefix(field, "-") {
				delete(out, strings.Split(strings.TrimPrefix(field, "-"), ".")[0])
				continue
			}
			field = strings.TrimPrefix(field, "+")
			copyPaymentField(out, source, field)
		}
	}
	return out
}

func copyPaymentField(dst, src map[string]any, path string) {
	parts := strings.SplitN(path, ".", 2)
	value, ok := src[parts[0]]
	if !ok {
		return
	}
	if len(parts) == 1 {
		dst[parts[0]] = value
		return
	}
	if value == nil {
		dst[parts[0]] = nil
		return
	}
	projectChild := func(source any) any {
		switch item := source.(type) {
		case map[string]any:
			target := map[string]any{}
			copyPaymentField(target, item, parts[1])
			return target
		case []any:
			target := make([]any, 0, len(item))
			for _, entry := range item {
				target = append(target, projectChildValue(entry, parts[1]))
			}
			return target
		default:
			return nil
		}
	}
	projected := projectChild(value)
	if projected == nil {
		return
	}
	if existing, ok := dst[parts[0]]; ok {
		if target, ok := existing.(map[string]any); ok {
			if next, ok := projected.(map[string]any); ok {
				mergePaymentFields(target, next)
				return
			}
		}
	}
	dst[parts[0]] = projected
}
func projectChildValue(source any, path string) any {
	item, ok := source.(map[string]any)
	if !ok {
		return nil
	}
	target := map[string]any{}
	copyPaymentField(target, item, path)
	return target
}
func mergePaymentFields(dst, src map[string]any) {
	for key, value := range src {
		if old, ok := dst[key]; ok {
			oldMap, oldOK := old.(map[string]any)
			newMap, newOK := value.(map[string]any)
			if oldOK && newOK {
				mergePaymentFields(oldMap, newMap)
				continue
			}
			oldList, oldOK := old.([]any)
			newList, newOK := value.([]any)
			if oldOK && newOK {
				for i := 0; i < len(oldList) && i < len(newList); i++ {
					oldItem, oldOK := oldList[i].(map[string]any)
					newItem, newOK := newList[i].(map[string]any)
					if oldOK && newOK {
						mergePaymentFields(oldItem, newItem)
					}
				}
				continue
			}
		}
		dst[key] = value
	}
}

func paymentPagination(c *gin.Context) (int, int) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 {
		limit = 20
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}
	return limit, offset
}

func paymentDateFilters(c *gin.Context, key string) map[string]string {
	ops := map[string]string{}
	allowed := map[string]string{"gt": "$gt", "gte": "$gte", "lt": "$lt", "lte": "$lte", "eq": "$eq", "ne": "$ne"}
	for raw, op := range allowed {
		for _, queryKey := range []string{key + "[" + raw + "]", key + "[$" + raw + "]"} {
			if value, exists := c.GetQuery(queryKey); exists {
				ops[op] = value
			}
		}
	}
	if value, exists := c.GetQuery(key); exists {
		ops["$eq"] = value
	}
	if len(ops) == 0 {
		return nil
	}
	return ops
}
