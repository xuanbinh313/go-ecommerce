package models

type FulfillmentProvider struct {
	ID        string `json:"id"`
	IsEnabled bool   `json:"is_enabled"`
}

type FulfillmentOption struct {
	ID       string `json:"id"`
	IsReturn bool   `json:"is_return,omitempty"`
}

type FulfillmentProviderResult struct {
	Data   map[string]any `json:"data"`
	Labels []any          `json:"labels"`
}

type CreateFulfillmentInput struct {
	Data           map[string]any   `json:"data"`
	Items          []map[string]any `json:"items"`
	Order          map[string]any   `json:"order"`
	Fulfillment    map[string]any   `json:"fulfillment"`
	AdditionalData map[string]any   `json:"additional_data,omitempty"`
}

type ShippingPriceResult struct {
	Amount                        any  `json:"amount,omitempty"`
	IsCalculatedPriceTaxInclusive bool `json:"is_calculated_price_tax_inclusive,omitempty"`
}
