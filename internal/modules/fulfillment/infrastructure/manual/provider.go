package manual

import (
	"context"
	"errors"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
)

const Identifier = "manual"

type Provider struct{}

func New() *Provider                 { return &Provider{} }
func (*Provider) Identifier() string { return Identifier }
func (*Provider) GetFulfillmentOptions(context.Context) ([]models.FulfillmentOption, error) {
	return []models.FulfillmentOption{{ID: "manual-fulfillment"}, {ID: "manual-fulfillment-return", IsReturn: true}}, nil
}
func (*Provider) ValidateFulfillmentData(_ context.Context, _ map[string]any, data map[string]any, _ map[string]any) (map[string]any, error) {
	return data, nil
}
func (*Provider) CalculatePrice(context.Context, map[string]any, map[string]any, map[string]any) (models.ShippingPriceResult, error) {
	return models.ShippingPriceResult{}, errors.New("Manual fulfillment does not support price calculation")
}
func (*Provider) CanCalculate(context.Context, map[string]any) (bool, error)   { return false, nil }
func (*Provider) ValidateOption(context.Context, map[string]any) (bool, error) { return true, nil }
func (*Provider) CreateFulfillment(context.Context, models.CreateFulfillmentInput) (models.FulfillmentProviderResult, error) {
	return models.FulfillmentProviderResult{Data: map[string]any{}, Labels: []any{}}, nil
}
func (*Provider) CancelFulfillment(context.Context, map[string]any) (map[string]any, error) {
	return map[string]any{}, nil
}
func (*Provider) CreateReturnFulfillment(context.Context, map[string]any) (models.FulfillmentProviderResult, error) {
	return models.FulfillmentProviderResult{Data: map[string]any{}, Labels: []any{}}, nil
}
