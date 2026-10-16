package models

import "time"

type Definition struct {
	Name        string
	TableName   string
	IDPrefix    string
	PrimaryKeys []string
	ForeignKeys []string
	ExtraFields []string
}

type Link struct {
	ID         string         `json:"id"`
	Definition string         `json:"definition"`
	Fields     map[string]any `json:"fields"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  *time.Time     `json:"deleted_at,omitempty"`
}

var Definitions = []Definition{
	{Name: "CartPaymentCollection", TableName: "cart_payment_collection", IDPrefix: "capaycol", PrimaryKeys: []string{"id", "cart_id", "payment_collection_id"}, ForeignKeys: []string{"cart_id", "payment_collection_id"}},
	{Name: "CartPromotion", TableName: "cart_promotion", IDPrefix: "cartpromo", PrimaryKeys: []string{"id", "cart_id", "promotion_id"}, ForeignKeys: []string{"cart_id", "promotion_id"}},
	{Name: "CustomerAccountHolder", TableName: "customer_account_holder", IDPrefix: "custacchldr", PrimaryKeys: []string{"id", "customer_id", "account_holder_id"}, ForeignKeys: []string{"customer_id", "account_holder_id"}},
	{Name: "LocationFulfillmentProvider", TableName: "location_fulfillment_provider", IDPrefix: "locfp", PrimaryKeys: []string{"id", "stock_location_id", "fulfillment_provider_id"}, ForeignKeys: []string{"stock_location_id", "fulfillment_provider_id"}},
	{Name: "LocationFulfillmentSet", TableName: "location_fulfillment_set", IDPrefix: "locfs", PrimaryKeys: []string{"id", "stock_location_id", "fulfillment_set_id"}, ForeignKeys: []string{"stock_location_id", "fulfillment_set_id"}},
	{Name: "InviteRBACRole", TableName: "invite_rbac_role", IDPrefix: "inviterole", PrimaryKeys: []string{"id", "invite_id", "rbac_role_id"}, ForeignKeys: []string{"invite_id", "rbac_role_id"}},
	{Name: "OrderCart", TableName: "order_cart", IDPrefix: "ordercart", PrimaryKeys: []string{"id", "order_id", "cart_id"}, ForeignKeys: []string{"order_id", "cart_id"}},
	{Name: "OrderClaimPaymentCollection", TableName: "order_claim_payment_collection", IDPrefix: "claimpay", PrimaryKeys: []string{"id", "claim_id", "payment_collection_id"}, ForeignKeys: []string{"claim_id", "payment_collection_id"}},
	{Name: "OrderExchangePaymentCollection", TableName: "order_exchange_payment_collection", IDPrefix: "excpay", PrimaryKeys: []string{"id", "exchange_id", "payment_collection_id"}, ForeignKeys: []string{"exchange_id", "payment_collection_id"}},
	{Name: "OrderFulfillment", TableName: "order_fulfillment", IDPrefix: "ordful", PrimaryKeys: []string{"id", "order_id", "fulfillment_id"}, ForeignKeys: []string{"order_id", "fulfillment_id"}},
	{Name: "OrderPaymentCollection", TableName: "order_payment_collection", IDPrefix: "ordpay", PrimaryKeys: []string{"id", "order_id", "payment_collection_id"}, ForeignKeys: []string{"order_id", "payment_collection_id"}},
	{Name: "OrderPromotion", TableName: "order_promotion", IDPrefix: "orderpromo", PrimaryKeys: []string{"id", "order_id", "promotion_id"}, ForeignKeys: []string{"order_id", "promotion_id"}},
	{Name: "ReturnFulfillment", TableName: "return_fulfillment", IDPrefix: "retful", PrimaryKeys: []string{"id", "return_id", "fulfillment_id"}, ForeignKeys: []string{"return_id", "fulfillment_id"}},
	{Name: "ProductSalesChannel", TableName: "product_sales_channel", IDPrefix: "prodsc", PrimaryKeys: []string{"id", "product_id", "sales_channel_id"}, ForeignKeys: []string{"product_id", "sales_channel_id"}},
	{Name: "ProductShippingProfile", TableName: "product_shipping_profile", IDPrefix: "prodsp", PrimaryKeys: []string{"id", "product_id", "shipping_profile_id"}, ForeignKeys: []string{"product_id", "shipping_profile_id"}},
	{Name: "ProductVariantInventoryItem", TableName: "product_variant_inventory_item", IDPrefix: "pvitem", PrimaryKeys: []string{"id", "variant_id", "inventory_item_id"}, ForeignKeys: []string{"variant_id", "inventory_item_id"}, ExtraFields: []string{"required_quantity"}},
	{Name: "ProductVariantPriceSet", TableName: "product_variant_price_set", IDPrefix: "pvps", PrimaryKeys: []string{"id", "variant_id", "price_set_id"}, ForeignKeys: []string{"variant_id", "price_set_id"}},
	{Name: "PublishableApiKeySalesChannel", TableName: "publishable_api_key_sales_channel", IDPrefix: "pksc", PrimaryKeys: []string{"id", "publishable_key_id", "sales_channel_id"}, ForeignKeys: []string{"publishable_key_id", "sales_channel_id"}},
	{Name: "RegionPaymentProvider", TableName: "region_payment_provider", IDPrefix: "regpp", PrimaryKeys: []string{"id", "region_id", "payment_provider_id"}, ForeignKeys: []string{"region_id", "payment_provider_id"}},
	{Name: "SalesChannelLocation", TableName: "sales_channel_stock_location", IDPrefix: "scloc", PrimaryKeys: []string{"id", "sales_channel_id", "stock_location_id"}, ForeignKeys: []string{"sales_channel_id", "stock_location_id"}},
	{Name: "ShippingOptionPriceSet", TableName: "shipping_option_price_set", IDPrefix: "sops", PrimaryKeys: []string{"id", "shipping_option_id", "price_set_id"}, ForeignKeys: []string{"shipping_option_id", "price_set_id"}},
	{Name: "UserRBACRole", TableName: "user_rbac_role", IDPrefix: "userrole", PrimaryKeys: []string{"id", "user_id", "rbac_role_id"}, ForeignKeys: []string{"user_id", "rbac_role_id"}},
}
