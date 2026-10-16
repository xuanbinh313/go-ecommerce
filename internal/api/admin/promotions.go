package admin

import (
	"context"
	"fmt"
	flows "github.com/example/medusa-go-clone/internal/coreflows/promotion/workflows"
	"github.com/example/medusa-go-clone/internal/modules/promotion/domain/models"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

type PromotionService interface {
	ListCampaigns(context.Context, models.CampaignFilter) ([]models.Campaign, int, error)
	GetCampaign(context.Context, string) (models.Campaign, error)
	CreateCampaign(context.Context, models.CampaignInput) (models.Campaign, error)
	UpdateCampaign(context.Context, string, models.CampaignUpdate) (models.Campaign, error)
	DeleteCampaign(context.Context, string) error
	LinkCampaignPromotions(context.Context, string, []string, []string) error
	List(context.Context, models.Filter) ([]models.Promotion, int, error)
	Get(context.Context, string) (models.Promotion, error)
	Create(context.Context, models.CreateInput) (models.Promotion, error)
	Update(context.Context, string, models.UpdateInput) (models.Promotion, error)
	Delete(context.Context, string) error
}
type PromotionRuleOption struct {
	Label any `json:"label"`
	Value any `json:"value"`
}
type PromotionRuleOptionReader interface {
	ListPromotionRuleOptionValues(context.Context, string) ([]PromotionRuleOption, error)
}
type PromotionHTTP struct {
	Service     PromotionService
	RuleOptions PromotionRuleOptionReader
	Permissions UserRolesWorkflow
	RBACEnabled bool
}

func (h *PromotionHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	campaigns := router.Group("/admin/campaigns", auth)
	campaigns.GET("", permissionMiddleware(h.Permissions, h.RBACEnabled, "campaign", "read"), h.listCampaigns)
	campaigns.POST("", permissionMiddleware(h.Permissions, h.RBACEnabled, "campaign", "create"), h.createCampaign)
	campaigns.GET("/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "campaign", "read"), h.getCampaign)
	campaigns.POST("/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "campaign", "update"), h.updateCampaign)
	campaigns.DELETE("/:id", permissionMiddleware(h.Permissions, h.RBACEnabled, "campaign", "delete"), h.deleteCampaign)
	campaigns.POST("/:id/promotions", permissionMiddleware(h.Permissions, h.RBACEnabled, "campaign", "update"), h.manageCampaignPromotions)
	a := router.Group("/admin/promotions", auth)
	a.GET("", h.list)
	a.POST("", h.create)
	a.GET("/:id", h.get)
	a.POST("/:id", h.update)
	a.DELETE("/:id", h.delete)
	a.GET("/rule-attribute-options/:rule_type", h.listRuleAttributeOptions)
	a.GET("/rule-value-options/:rule_type/:rule_attribute_id", h.listRuleValueOptions)
}

func (h *PromotionHTTP) listRuleValueOptions(c *gin.Context) {
	ruleType := strings.ReplaceAll(c.Param("rule_type"), "-", "_")
	if ruleType != "rules" && ruleType != "target_rules" && ruleType != "buy_rules" {
		c.JSON(400, gin.H{"type": "invalid_data", "message": fmt.Sprintf("Invalid param rule_type (%s)", c.Param("rule_type"))})
		return
	}
	attribute := c.Param("rule_attribute_id")
	valid := false
	for _, item := range promotionRuleAttributes(ruleType, "", "", c.Query("application_method_target_type")) {
		if item.ID == attribute {
			valid = true
			break
		}
	}
	if _, ok := map[string]bool{"region": true, "currency_code": true, "customer_group": true, "sales_channel": true, "country": true, "product": true, "product_category": true, "product_collection": true, "product_type": true, "product_tag": true, "shipping_option_type": true}[attribute]; !ok || !valid {
		c.JSON(400, gin.H{"type": "invalid_data", "message": fmt.Sprintf("Invalid rule attribute - %s", attribute)})
		return
	}
	if h.RuleOptions == nil {
		c.JSON(501, gin.H{"type": "not_implemented", "message": "Promotion rule value options are unavailable for this module"})
		return
	}
	rows, err := h.RuleOptions.ListPromotionRuleOptionValues(c, attribute)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	value := c.Query("value")
	filtered := make([]PromotionRuleOption, 0, len(rows))
	for _, row := range rows {
		if value != "" && fmt.Sprint(row.Value) != value {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(fmt.Sprint(row.Label)), q) {
			continue
		}
		filtered = append(filtered, row)
	}
	limit, offset := 50, 0
	if n, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && n > 0 {
		limit = n
	}
	if n, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && n >= 0 {
		offset = n
	}
	count := len(filtered)
	if offset > count {
		offset = count
	}
	end := offset + limit
	if end > count {
		end = count
	}
	c.JSON(200, gin.H{"values": filtered[offset:end], "count": count, "offset": offset, "limit": limit})
}

type promotionRuleOperator struct {
	ID    string `json:"id"`
	Value string `json:"value"`
	Label string `json:"label"`
}

type promotionRuleAttribute struct {
	ID        string                  `json:"id"`
	Value     string                  `json:"value"`
	Label     string                  `json:"label"`
	Required  bool                    `json:"required"`
	FieldType string                  `json:"field_type"`
	Operators []promotionRuleOperator `json:"operators"`
	Disguised bool                    `json:"disguised,omitempty"`
	Hydrate   bool                    `json:"hydrate,omitempty"`
}

var promotionRuleOperators = []promotionRuleOperator{
	{ID: "in", Value: "in", Label: "In"},
	{ID: "eq", Value: "eq", Label: "Equals"},
	{ID: "ne", Value: "ne", Label: "Not In"},
}

func (h *PromotionHTTP) listRuleAttributeOptions(c *gin.Context) {
	ruleType := strings.ReplaceAll(c.Param("rule_type"), "-", "_")
	if ruleType != "rules" && ruleType != "target_rules" && ruleType != "buy_rules" {
		c.JSON(400, gin.H{"type": "invalid_data", "message": fmt.Sprintf("Invalid param rule_type (%s)", c.Param("rule_type"))})
		return
	}
	attributes := promotionRuleAttributes(ruleType, c.Query("promotion_type"), c.Query("application_method_type"), c.Query("application_method_target_type"))
	c.JSON(200, gin.H{"attributes": attributes})
}

func promotionRuleAttributes(ruleType, promotionType, methodType, targetType string) []promotionRuleAttribute {
	items := []promotionRuleAttribute{
		{ID: "customer_group", Value: "customer.groups.id", Label: "Customer Group", FieldType: "multiselect", Operators: promotionRuleOperators},
		{ID: "region", Value: "region.id", Label: "Region", FieldType: "multiselect", Operators: promotionRuleOperators},
		{ID: "country", Value: "shipping_address.country_code", Label: "Country", FieldType: "multiselect", Operators: promotionRuleOperators},
		{ID: "sales_channel", Value: "sales_channel_id", Label: "Sales Channel", FieldType: "multiselect", Operators: promotionRuleOperators},
	}
	productItems := []promotionRuleAttribute{
		{ID: "product", Value: "items.product.id", Label: "Product", FieldType: "multiselect", Operators: promotionRuleOperators},
		{ID: "product_category", Value: "items.product.categories.id", Label: "Product Category", FieldType: "multiselect", Operators: promotionRuleOperators},
		{ID: "product_collection", Value: "items.product.collection_id", Label: "Product Collection", FieldType: "multiselect", Operators: promotionRuleOperators},
		{ID: "product_type", Value: "items.product.type_id", Label: "Product Type", FieldType: "multiselect", Operators: promotionRuleOperators},
		{ID: "product_tag", Value: "items.product.tags.id", Label: "Product Tag", FieldType: "multiselect", Operators: promotionRuleOperators},
	}
	shippingItems := []promotionRuleAttribute{{ID: "shipping_option_type", Value: "shipping_methods.shipping_option.shipping_option_type_id", Label: "Shipping Option Type", FieldType: "multiselect", Operators: promotionRuleOperators}}
	eq := []promotionRuleOperator{{ID: "eq", Value: "eq", Label: "Equals"}}
	currencyRequired := methodType == "fixed"
	currency := promotionRuleAttribute{ID: "currency_code", Value: "currency_code", Label: "Currency Code", Required: currencyRequired, FieldType: "select", Operators: eq, Disguised: true, Hydrate: true}
	if ruleType == "rules" {
		items = append(items, currency)
		return items
	}
	if targetType == "shipping_methods" {
		items = shippingItems
	} else {
		items = productItems
	}
	if promotionType == "buyget" {
		if ruleType == "buy_rules" {
			items = append(items, promotionRuleAttribute{ID: "buy_rules_min_quantity", Value: "buy_rules_min_quantity", Label: "Minimum quantity of items", Required: true, FieldType: "number", Operators: eq, Disguised: true})
		} else {
			items = append(items, promotionRuleAttribute{ID: "apply_to_quantity", Value: "apply_to_quantity", Label: "Quantity of items promotion will apply to", Required: true, FieldType: "number", Operators: eq, Disguised: true})
		}
	}
	return items
}
func (h *PromotionHTTP) list(c *gin.Context) {
	limit, offset := 50, 0
	if n, e := strconv.Atoi(c.DefaultQuery("limit", "50")); e == nil && n > 0 {
		limit = n
	}
	if n, e := strconv.Atoi(c.DefaultQuery("offset", "0")); e == nil && n >= 0 {
		offset = n
	}
	f := models.Filter{IDs: queryValues(c, "id"), Codes: queryValues(c, "code"), CampaignIDs: queryValues(c, "campaign_id"), Statuses: queryValues(c, "status"), Limit: limit, Offset: offset}
	rows, count, err := h.Service.List(c, f)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, listResponse("promotions", rows, count, offset, limit))
}
func (h *PromotionHTTP) get(c *gin.Context) {
	v, err := h.Service.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"promotion": v})
}
func (h *PromotionHTTP) create(c *gin.Context) {
	var in models.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	v, err := flows.CreatePromotion(c, h.Service, in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"promotion": v})
}
func (h *PromotionHTTP) update(c *gin.Context) {
	var in models.UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	v, err := flows.UpdatePromotion(c, h.Service, c.Param("id"), in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"promotion": v})
}
func (h *PromotionHTTP) delete(c *gin.Context) {
	id := c.Param("id")
	if err := flows.DeletePromotion(c, h.Service, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "promotion", "deleted": true})
}
