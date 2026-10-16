package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain"
	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	"time"
)

type Service struct {
	repo      domain.Repository
	providers *ProviderService
}

func (s *Service) ListShippingOptions(ctx context.Context, f models.ShippingOptionFilter, limit, offset int) ([]models.ShippingOption, int, error) {
	return s.repo.ListShippingOptions(ctx, f, limit, offset)
}
func (s *Service) GetShippingOption(ctx context.Context, id string) (models.ShippingOption, error) {
	return s.repo.GetShippingOption(ctx, id)
}
func (s *Service) CreateShippingOption(ctx context.Context, in models.ShippingOptionInput) (models.ShippingOption, error) {
	if in.Name == "" || in.ServiceZoneID == "" || in.ShippingProfileID == "" || in.ProviderID == "" || in.PriceType == "" || in.Prices == nil {
		return models.ShippingOption{}, fmt.Errorf("%w: shipping option name, service_zone_id, shipping_profile_id, provider_id, price_type, and prices are required", domain.ErrInvalidData)
	}
	if in.PriceType != "flat" && in.PriceType != "calculated" {
		return models.ShippingOption{}, fmt.Errorf("%w: invalid shipping option price_type %q", domain.ErrInvalidData, in.PriceType)
	}
	if (in.Type == nil) == (in.TypeID == "") {
		return models.ShippingOption{}, fmt.Errorf("%w: exactly one of type or type_id must be provided", domain.ErrInvalidData)
	}
	for _, rule := range in.Rules {
		if !validRuleOperator(rule.Operator) || rule.Attribute == "" {
			return models.ShippingOption{}, fmt.Errorf("%w: invalid shipping option rule", domain.ErrInvalidData)
		}
	}
	if in.Type != nil && (in.Type.Label == "" || in.Type.Code == "") {
		return models.ShippingOption{}, fmt.Errorf("%w: shipping option type label and code are required", domain.ErrInvalidData)
	}
	for _, price := range in.Prices {
		if (price.CurrencyCode == nil) == (price.RegionID == nil) || len(price.Amount) == 0 {
			return models.ShippingOption{}, fmt.Errorf("%w: each price requires exactly one of currency_code or region_id and an amount", domain.ErrInvalidData)
		}
		for _, rule := range price.Rules {
			if rule.Attribute != "item_total" || !validPriceRuleOperator(rule.Operator) {
				return models.ShippingOption{}, fmt.Errorf("%w: invalid shipping option price rule", domain.ErrInvalidData)
			}
		}
	}
	p, err := s.providerService()
	if err != nil {
		return models.ShippingOption{}, err
	}
	provider, err := p.Retrieve(in.ProviderID)
	if err != nil {
		return models.ShippingOption{}, err
	}
	if in.PriceType == "calculated" {
		canCalculate, err := provider.CanCalculate(ctx, in.Data)
		if err != nil {
			return models.ShippingOption{}, err
		}
		if !canCalculate {
			return models.ShippingOption{}, fmt.Errorf("%w: provider %q cannot calculate shipping option prices", domain.ErrInvalidData, in.ProviderID)
		}
	}
	allowed, err := s.repo.CanProviderServeZone(ctx, in.ServiceZoneID, in.ProviderID)
	if err != nil {
		return models.ShippingOption{}, err
	}
	if !allowed {
		return models.ShippingOption{}, fmt.Errorf("%w: provider %q is not enabled for the service zone", domain.ErrNotAllowed, in.ProviderID)
	}
	if in.ShippingProfileID != "" {
		if _, err = s.repo.GetShippingProfile(ctx, in.ShippingProfileID); err != nil {
			return models.ShippingOption{}, err
		}
	}
	return s.repo.CreateShippingOption(ctx, in)
}

func (s *Service) CreateFulfillmentRecord(ctx context.Context, input models.FulfillmentCreateInput) (models.FulfillmentRecord, error) {
	if input.LocationID == "" || input.ProviderID == "" || input.OrderID == "" {
		return models.FulfillmentRecord{}, fmt.Errorf("%w: location_id, provider_id, and order_id are required", domain.ErrInvalidData)
	}
	created, err := s.repo.PersistFulfillment(ctx, input)
	if err != nil {
		return models.FulfillmentRecord{}, err
	}
	data := map[string]any{}
	if len(input.Data) > 0 {
		_ = json.Unmarshal(input.Data, &data)
	}
	items := make([]map[string]any, 0, len(input.Items))
	for _, item := range input.Items {
		var itemData map[string]any
		encoded, _ := json.Marshal(item)
		_ = json.Unmarshal(encoded, &itemData)
		items = append(items, itemData)
	}
	address, _ := json.Marshal(input.DeliveryAddress)
	var addressData map[string]any
	_ = json.Unmarshal(address, &addressData)
	fulfillmentData := map[string]any{"id": created.ID, "location_id": input.LocationID, "location": input.Location, "provider_id": input.ProviderID, "order_id": input.OrderID, "shipping_option_id": input.ShippingOptionID, "delivery_address": addressData, "items": items, "created_by": input.CreatedBy}
	provider, err := s.CreateFulfillment(ctx, input.ProviderID, models.CreateFulfillmentInput{Data: data, Items: items, Order: map[string]any{"id": input.OrderID}, Fulfillment: fulfillmentData})
	if err != nil {
		if compensateErr := s.repo.HardDeleteFulfillment(ctx, created.ID); compensateErr != nil {
			return models.FulfillmentRecord{}, errors.Join(err, fmt.Errorf("compensate create-fulfillment: %w", compensateErr))
		}
		return models.FulfillmentRecord{}, err
	}
	updated, err := s.repo.UpdateFulfillmentProviderResult(ctx, created.ID, provider)
	if err != nil {
		if compensateErr := s.repo.HardDeleteFulfillment(ctx, created.ID); compensateErr != nil {
			err = errors.Join(err, fmt.Errorf("compensate create-fulfillment: %w", compensateErr))
		}
		return models.FulfillmentRecord{}, err
	}
	return updated, nil
}

func (s *Service) GetFulfillment(ctx context.Context, id string) (models.FulfillmentRecord, error) {
	return s.repo.GetFulfillment(ctx, id)
}

func (s *Service) CancelFulfillmentRecord(ctx context.Context, id string) (models.FulfillmentRecord, error) {
	current, err := s.repo.GetFulfillment(ctx, id)
	if err != nil {
		return models.FulfillmentRecord{}, err
	}
	if current.CanceledAt != nil {
		return current, nil
	}
	if current.ShippedAt != nil || current.DeliveredAt != nil {
		return models.FulfillmentRecord{}, fmt.Errorf("%w: fulfillment %s can no longer be canceled", domain.ErrInvalidData, id)
	}
	if current.ProviderID == nil {
		return models.FulfillmentRecord{}, fmt.Errorf("fulfillment provider is missing")
	}
	providerData := map[string]any{}
	_ = json.Unmarshal(current.Data, &providerData)
	if _, err = s.CancelFulfillment(ctx, *current.ProviderID, providerData); err != nil {
		return models.FulfillmentRecord{}, err
	}
	return s.repo.MarkFulfillmentCanceled(ctx, id)
}

func (s *Service) CreateFulfillmentShipment(ctx context.Context, id string, input models.CreateShipmentInput) (models.FulfillmentRecord, models.FulfillmentRecord, error) {
	return s.repo.MarkFulfillmentShipped(ctx, id, input, time.Now().UTC())
}

func (s *Service) RestoreFulfillment(ctx context.Context, previous models.FulfillmentRecord) error {
	return s.repo.RestoreFulfillment(ctx, previous)
}

func (s *Service) HardDeleteFulfillment(ctx context.Context, id string) error {
	return s.repo.HardDeleteFulfillment(ctx, id)
}
func (s *Service) UpdateShippingOption(ctx context.Context, id string, in models.ShippingOptionUpdateInput) (models.ShippingOption, error) {
	current, err := s.repo.GetShippingOption(ctx, id)
	if err != nil {
		return models.ShippingOption{}, err
	}
	if in.PriceType != nil && *in.PriceType != "flat" && *in.PriceType != "calculated" {
		return models.ShippingOption{}, fmt.Errorf("%w: invalid shipping option price_type %q", domain.ErrInvalidData, *in.PriceType)
	}
	effectivePriceType := current.PriceType
	if in.PriceType != nil {
		effectivePriceType = *in.PriceType
	}
	providerID := current.ProviderID
	if in.ProviderID != nil {
		providerID = in.ProviderID
	}
	if effectivePriceType == "calculated" {
		if providerID == nil || *providerID == "" {
			return models.ShippingOption{}, fmt.Errorf("%w: a provider is required for calculated shipping options", domain.ErrInvalidData)
		}
		p, providerErr := s.providerService()
		if providerErr != nil {
			return models.ShippingOption{}, providerErr
		}
		provider, providerErr := p.Retrieve(*providerID)
		if providerErr != nil {
			return models.ShippingOption{}, providerErr
		}
		optionData := map[string]any{}
		if in.Data != nil {
			optionData = *in.Data
		} else if len(current.Data) > 0 {
			_ = json.Unmarshal(current.Data, &optionData)
		}
		canCalculate, providerErr := provider.CanCalculate(ctx, optionData)
		if providerErr != nil {
			return models.ShippingOption{}, providerErr
		}
		if !canCalculate {
			return models.ShippingOption{}, fmt.Errorf("%w: provider %q cannot calculate shipping option prices", domain.ErrInvalidData, *providerID)
		}
	}
	if providerID == nil || *providerID == "" {
		return models.ShippingOption{}, fmt.Errorf("%w: provider is required", domain.ErrInvalidData)
	}
	allowed, providerErr := s.repo.CanProviderServeZone(ctx, current.ServiceZoneID, *providerID)
	if providerErr != nil {
		return models.ShippingOption{}, providerErr
	}
	if !allowed {
		return models.ShippingOption{}, fmt.Errorf("%w: provider %q is not enabled for the service zone", domain.ErrNotAllowed, *providerID)
	}
	if in.ShippingProfileID != nil && *in.ShippingProfileID != "" {
		if _, err = s.repo.GetShippingProfile(ctx, *in.ShippingProfileID); err != nil {
			return models.ShippingOption{}, err
		}
	}
	if in.Type != nil && in.TypeID != nil {
		return models.ShippingOption{}, fmt.Errorf("%w: only one of type or type_id can be provided", domain.ErrInvalidData)
	}
	if in.Type != nil && (in.Type.Label == "" || in.Type.Code == "") {
		return models.ShippingOption{}, fmt.Errorf("%w: shipping option type label and code are required", domain.ErrInvalidData)
	}
	if in.Rules != nil {
		for _, rule := range *in.Rules {
			if rule.ID == "" && (rule.Attribute == nil || rule.Operator == nil || rule.Value == nil) || rule.Attribute != nil && *rule.Attribute == "" || rule.Operator != nil && !validRuleOperator(*rule.Operator) || rule.Value != nil && !validShippingRuleValue(*rule.Value) {
				return models.ShippingOption{}, fmt.Errorf("%w: invalid shipping option rule", domain.ErrInvalidData)
			}
		}
	}
	if in.Prices != nil && effectivePriceType == "flat" {
		for _, price := range *in.Prices {
			if price.ID == nil && ((price.CurrencyCode == nil) == (price.RegionID == nil) || price.Amount == nil) || price.CurrencyCode != nil && price.RegionID != nil {
				return models.ShippingOption{}, fmt.Errorf("%w: each price requires an id or exactly one of currency_code or region_id and an amount", domain.ErrInvalidData)
			}
			if price.Rules != nil {
				for _, rule := range *price.Rules {
					if rule.Attribute != "item_total" || !validPriceRuleOperator(rule.Operator) {
						return models.ShippingOption{}, fmt.Errorf("%w: invalid shipping option price rule", domain.ErrInvalidData)
					}
				}
			}
		}
	}
	if effectivePriceType == "calculated" {
		emptyPrices := []models.ShippingOptionUpdatePriceInput{}
		in.Prices = &emptyPrices
	}
	return s.repo.UpdateShippingOption(ctx, id, in)
}
func (s *Service) HardDeleteShippingOption(ctx context.Context, id string) error {
	return s.repo.HardDeleteShippingOption(ctx, id)
}
func (s *Service) SoftDeleteShippingOptions(ctx context.Context, ids []string) error {
	return s.repo.SoftDeleteShippingOptions(ctx, ids)
}
func (s *Service) RestoreShippingOptions(ctx context.Context, ids []string) error {
	return s.repo.RestoreShippingOptions(ctx, ids)
}
func (s *Service) CanProviderServeZone(ctx context.Context, zoneID, providerID string) (bool, error) {
	return s.repo.CanProviderServeZone(ctx, zoneID, providerID)
}
func (s *Service) BatchShippingOptionRules(ctx context.Context, optionID string, input models.ShippingOptionRuleBatchInput) (models.ShippingOptionRuleBatchResult, error) {
	if _, err := s.repo.GetShippingOption(ctx, optionID); err != nil {
		return models.ShippingOptionRuleBatchResult{}, err
	}
	for _, rule := range input.Create {
		if rule.Attribute == "" || !validRuleOperator(rule.Operator) || !validShippingRuleValue(rule.Value) {
			return models.ShippingOptionRuleBatchResult{}, fmt.Errorf("%w: invalid shipping option rule", domain.ErrInvalidData)
		}
	}
	for _, rule := range input.Update {
		if rule.ID == "" || (rule.Attribute != nil && *rule.Attribute == "") || (rule.Operator != nil && !validRuleOperator(*rule.Operator)) || (rule.Value != nil && !validShippingRuleValue(*rule.Value)) {
			return models.ShippingOptionRuleBatchResult{}, fmt.Errorf("%w: invalid shipping option rule update", domain.ErrInvalidData)
		}
	}
	return s.repo.BatchShippingOptionRules(ctx, optionID, input)
}
func (s *Service) CompensateShippingOptionRuleBatch(ctx context.Context, result models.ShippingOptionRuleBatchResult) error {
	return s.repo.CompensateShippingOptionRuleBatch(ctx, result)
}
func validRuleOperator(v string) bool {
	switch v {
	case "in", "eq", "ne", "gt", "gte", "lt", "lte", "nin":
		return true
	}
	return false
}
func validPriceRuleOperator(v string) bool {
	switch v {
	case "gt", "gte", "lt", "lte":
		return true
	}
	return false
}
func validShippingRuleValue(raw json.RawMessage) bool {
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return true
	}
	var multiple []string
	return json.Unmarshal(raw, &multiple) == nil
}

func (s *Service) ListShippingOptionTypes(ctx context.Context, f models.ShippingOptionTypeFilter, limit, offset int) ([]models.ShippingOptionType, int, error) {
	return s.repo.ListShippingOptionTypes(ctx, f, limit, offset)
}
func (s *Service) GetShippingOptionType(ctx context.Context, id string) (models.ShippingOptionType, error) {
	return s.repo.GetShippingOptionType(ctx, id)
}
func (s *Service) CreateShippingOptionTypes(ctx context.Context, in []models.ShippingOptionTypeInput) ([]models.ShippingOptionType, error) {
	return s.repo.CreateShippingOptionTypes(ctx, in)
}
func (s *Service) UpdateShippingOptionType(ctx context.Context, id string, in models.ShippingOptionTypeUpdate) (models.ShippingOptionType, error) {
	return s.repo.UpdateShippingOptionType(ctx, id, in)
}
func (s *Service) SoftDeleteShippingOptionTypes(ctx context.Context, ids []string) error {
	return s.repo.SoftDeleteShippingOptionTypes(ctx, ids)
}
func (s *Service) RestoreShippingOptionTypes(ctx context.Context, ids []string) error {
	return s.repo.RestoreShippingOptionTypes(ctx, ids)
}
func (s *Service) HardDeleteShippingOptionTypes(ctx context.Context, ids []string) error {
	return s.repo.HardDeleteShippingOptionTypes(ctx, ids)
}
func (s *Service) ShippingOptionTypeInUse(ctx context.Context, id string) (bool, error) {
	return s.repo.ShippingOptionTypeInUse(ctx, id)
}

func New(r domain.Repository) *Service { return &Service{repo: r} }
func (s *Service) ListShippingProfiles(ctx context.Context, f models.ShippingProfileFilter, limit, offset int) ([]models.ShippingProfile, int, error) {
	return s.repo.ListShippingProfiles(ctx, f, limit, offset)
}
func (s *Service) GetShippingProfile(ctx context.Context, id string) (models.ShippingProfile, error) {
	return s.repo.GetShippingProfile(ctx, id)
}
func (s *Service) CreateShippingProfiles(ctx context.Context, in []models.ShippingProfileInput) ([]models.ShippingProfile, error) {
	return s.repo.CreateShippingProfiles(ctx, in)
}
func (s *Service) UpdateShippingProfile(ctx context.Context, id string, fields map[string]any) (models.ShippingProfile, error) {
	allowed := map[string]bool{"name": true, "type": true, "metadata": true}
	for key := range fields {
		if !allowed[key] {
			return models.ShippingProfile{}, fmt.Errorf("unsupported shipping profile field %q", key)
		}
	}
	return s.repo.UpdateShippingProfile(ctx, id, fields)
}
func (s *Service) SoftDeleteShippingProfiles(ctx context.Context, ids []string) error {
	return s.repo.SoftDeleteShippingProfiles(ctx, ids)
}
func (s *Service) RestoreShippingProfiles(ctx context.Context, ids []string) error {
	return s.repo.RestoreShippingProfiles(ctx, ids)
}
func (s *Service) HardDeleteShippingProfiles(ctx context.Context, ids []string) error {
	return s.repo.HardDeleteShippingProfiles(ctx, ids)
}
func NewWithProviders(r domain.Repository, providers *ProviderService) *Service {
	return &Service{repo: r, providers: providers}
}

func (s *Service) providerService() (*ProviderService, error) {
	if s.providers == nil {
		return nil, fmt.Errorf("fulfillment provider service is not configured")
	}
	return s.providers, nil
}
func (s *Service) GetFulfillmentOptions(ctx context.Context, id string) ([]models.FulfillmentOption, error) {
	p, e := s.providerService()
	if e != nil {
		return nil, e
	}
	return p.GetFulfillmentOptions(ctx, id)
}
func (s *Service) ListFulfillmentProviders() ([]models.FulfillmentProvider, error) {
	p, e := s.providerService()
	if e != nil {
		return nil, e
	}
	return p.List(), nil
}
func (s *Service) ValidateFulfillmentData(ctx context.Context, id string, optionData, data, contextData map[string]any) (map[string]any, error) {
	p, e := s.providerService()
	if e != nil {
		return nil, e
	}
	return p.ValidateFulfillmentData(ctx, id, optionData, data, contextData)
}
func (s *Service) CalculatePrice(ctx context.Context, id string, optionData, data, contextData map[string]any) (models.ShippingPriceResult, error) {
	p, e := s.providerService()
	if e != nil {
		return models.ShippingPriceResult{}, e
	}
	return p.CalculatePrice(ctx, id, optionData, data, contextData)
}
func (s *Service) CanCalculate(ctx context.Context, id string, data map[string]any) (bool, error) {
	p, e := s.providerService()
	if e != nil {
		return false, e
	}
	return p.CanCalculate(ctx, id, data)
}
func (s *Service) ValidateOption(ctx context.Context, id string, data map[string]any) (bool, error) {
	p, e := s.providerService()
	if e != nil {
		return false, e
	}
	return p.ValidateOption(ctx, id, data)
}
func (s *Service) CreateFulfillment(ctx context.Context, id string, input models.CreateFulfillmentInput) (models.FulfillmentProviderResult, error) {
	p, e := s.providerService()
	if e != nil {
		return models.FulfillmentProviderResult{}, e
	}
	return p.CreateFulfillment(ctx, id, input)
}
func (s *Service) CancelFulfillment(ctx context.Context, id string, fulfillment map[string]any) (map[string]any, error) {
	p, e := s.providerService()
	if e != nil {
		return nil, e
	}
	return p.CancelFulfillment(ctx, id, fulfillment)
}
func (s *Service) CreateReturnFulfillment(ctx context.Context, id string, fulfillment map[string]any) (models.FulfillmentProviderResult, error) {
	p, e := s.providerService()
	if e != nil {
		return models.FulfillmentProviderResult{}, e
	}
	return p.CreateReturnFulfillment(ctx, id, fulfillment)
}
func (s *Service) CreateSet(c context.Context, in models.CreateSetInput) (models.FulfillmentSet, error) {
	if in.Name == "" || in.Type == "" {
		return models.FulfillmentSet{}, fmt.Errorf("name and type are required")
	}
	return s.repo.CreateSet(c, in)
}
func (s *Service) GetSet(c context.Context, id string) (models.FulfillmentSet, error) {
	return s.repo.GetSet(c, id)
}
func (s *Service) DeleteSet(c context.Context, id string) error  { return s.repo.DeleteSet(c, id) }
func (s *Service) RestoreSet(c context.Context, id string) error { return s.repo.RestoreSet(c, id) }
func (s *Service) CreateServiceZone(c context.Context, id string, in models.ServiceZoneInput) (models.FulfillmentSet, error) {
	if in.Name == "" {
		return models.FulfillmentSet{}, fmt.Errorf("service zone name is required")
	}
	for i := range in.GeoZones {
		if in.GeoZones[i].CountryCode == "" {
			return models.FulfillmentSet{}, fmt.Errorf("geo zone country_code is required")
		}
		if in.GeoZones[i].Type == "" {
			in.GeoZones[i].Type = "country"
		}
		switch in.GeoZones[i].Type {
		case "country", "province", "city", "zip":
		default:
			return models.FulfillmentSet{}, fmt.Errorf("invalid geo zone type %q", in.GeoZones[i].Type)
		}
	}
	return s.repo.CreateServiceZone(c, id, in)
}
func (s *Service) GetServiceZone(c context.Context, id string) (models.ServiceZone, error) {
	return s.repo.GetServiceZone(c, id)
}
func (s *Service) UpdateServiceZone(c context.Context, id string, in models.ServiceZoneUpdate) (models.FulfillmentSet, error) {
	return s.repo.UpdateServiceZone(c, id, in)
}
func (s *Service) DeleteServiceZone(c context.Context, id string) (models.FulfillmentSet, error) {
	return s.repo.DeleteServiceZone(c, id)
}
