package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
)

const (
	UpdateShippingOptionsWorkflowID = "update-shipping-options-workflow"
	SetShippingOptionsPricesStepID  = "set-shipping-options-prices-step"
	ShippingOptionUpdatedEvent      = "shipping-option.updated"
)

type ShippingOptionUpdater interface {
	GetShippingOption(context.Context, string) (models.ShippingOption, error)
	UpdateShippingOption(context.Context, string, models.ShippingOptionUpdateInput) (models.ShippingOption, error)
}

func UpdateShippingOption(ctx context.Context, service ShippingOptionUpdater, events ShippingOptionEventPublisher, id string, input models.ShippingOptionUpdateInput) (models.ShippingOption, error) {
	previous, err := service.GetShippingOption(ctx, id)
	if err != nil {
		return models.ShippingOption{}, err
	}
	updated, err := service.UpdateShippingOption(ctx, id, input)
	if err != nil {
		return models.ShippingOption{}, err
	}
	if events == nil {
		err = errors.New("shipping option event publisher is required")
	} else {
		err = events.Publish(ctx, ShippingOptionUpdatedEvent, []map[string]any{{"id": updated.ID}})
	}
	if err != nil {
		rollback := shippingOptionSnapshot(previous)
		if _, compensationErr := service.UpdateShippingOption(ctx, id, rollback); compensationErr != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", UpsertShippingOptionsStepID, compensationErr))
		}
		return models.ShippingOption{}, err
	}
	return updated, nil
}

func shippingOptionSnapshot(option models.ShippingOption) models.ShippingOptionUpdateInput {
	name, priceType := option.Name, option.PriceType
	providerID, profileID, typeID := option.ProviderID, option.ShippingProfileID, option.ShippingOptionTypeID
	var data map[string]any
	var metadata map[string]any
	_ = json.Unmarshal(option.Data, &data)
	_ = json.Unmarshal(option.Metadata, &metadata)
	input := models.ShippingOptionUpdateInput{Name: &name, PriceType: &priceType, ProviderID: providerID, ShippingProfileID: profileID, TypeID: &typeID, Data: &data, Metadata: &metadata}
	if option.Type != nil {
		input.Type = &models.ShippingOptionTypeInput{Label: option.Type.Label, Description: option.Type.Description, Code: option.Type.Code}
		input.TypeID = nil
	}
	rules := make([]models.ShippingOptionRuleUpdate, 0, len(option.Rules))
	for _, rule := range option.Rules {
		attribute, operator, value := rule.Attribute, rule.Operator, json.RawMessage(rule.Value)
		rules = append(rules, models.ShippingOptionRuleUpdate{ID: rule.ID, Attribute: &attribute, Operator: &operator, Value: &value})
	}
	input.Rules = &rules
	prices := make([]models.ShippingOptionUpdatePriceInput, 0, len(option.Prices))
	for _, row := range option.Prices {
		price := models.ShippingOptionUpdatePriceInput{}
		if id, ok := row["id"].(string); ok {
			price.ID = &id
		}
		if code, ok := row["currency_code"].(string); ok {
			price.CurrencyCode = &code
		}
		if amount, ok := row["amount"]; ok {
			raw, _ := json.Marshal(amount)
			message := json.RawMessage(raw)
			price.Amount = &message
		}
		if ruleRows, ok := row["price_rules"].([]map[string]any); ok {
			convertedRules := make([]models.ShippingOptionPriceRuleInput, 0, len(ruleRows))
			for _, rule := range ruleRows {
				attribute, _ := rule["attribute"].(string)
				operator, _ := rule["operator"].(string)
				value, _ := rule["value"]
				if attribute == "region_id" && operator == "eq" {
					region, _ := value.(string)
					price.RegionID = &region
					continue
				}
				raw, _ := json.Marshal(value)
				convertedRules = append(convertedRules, models.ShippingOptionPriceRuleInput{Attribute: attribute, Operator: operator, Value: raw})
			}
			price.Rules = &convertedRules
		}
		prices = append(prices, price)
	}
	input.Prices = &prices
	return input
}
