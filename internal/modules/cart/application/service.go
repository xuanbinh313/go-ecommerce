package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/cart/domain"
	"github.com/example/medusa-go-clone/internal/modules/cart/domain/models"
	regionmodels "github.com/example/medusa-go-clone/internal/modules/region/domain/models"
)

func jsonAddress(country string) ([]byte, error) {
	return json.Marshal(map[string]string{"country_code": country})
}

type Service struct {
	repo     domain.Repository
	regions  domain.RegionLookup
	stores   domain.StoreLookup
	channels domain.SalesChannelLookup
}

func NewService(repo domain.Repository, regions domain.RegionLookup, stores domain.StoreLookup, channels domain.SalesChannelLookup) *Service {
	return &Service{repo: repo, regions: regions, stores: stores, channels: channels}
}
func (s *Service) Get(ctx context.Context, id string) (models.Cart, error) {
	return s.repo.Get(ctx, id)
}
func (s *Service) CreateLineItems(ctx context.Context, cartID string, items []models.LineItemInput) ([]models.LineItem, error) {
	if cartID == "" {
		return nil, fmt.Errorf("cart_id is required")
	}
	for _, item := range items {
		if item.Title == "" || item.Quantity <= 0 {
			return nil, fmt.Errorf("line item title and a quantity greater than zero are required")
		}
		if item.UnitPrice == "" || len(item.RawUnitPrice) == 0 {
			return nil, fmt.Errorf("line item price is required")
		}
	}
	return s.repo.CreateLineItems(ctx, cartID, items)
}
func (s *Service) GetLineItem(ctx context.Context, cartID, itemID string) (models.LineItem, error) {
	return s.repo.GetLineItem(ctx, cartID, itemID)
}
func (s *Service) UpdateLineItem(ctx context.Context, cartID, itemID string, in models.LineItemUpdate) (models.LineItem, error) {
	if cartID == "" || itemID == "" {
		return models.LineItem{}, fmt.Errorf("cart_id and item_id are required")
	}
	if in.Quantity != nil && *in.Quantity < 0 {
		return models.LineItem{}, fmt.Errorf("quantity cannot be negative")
	}
	return s.repo.UpdateLineItem(ctx, cartID, itemID, in)
}
func (s *Service) SoftDeleteLineItems(ctx context.Context, cartID string, ids []string) ([]models.LineItem, error) {
	return s.repo.SoftDeleteLineItems(ctx, cartID, ids)
}
func (s *Service) RestoreLineItems(ctx context.Context, ids []string) error {
	return s.repo.RestoreLineItems(ctx, ids)
}
func (s *Service) Create(ctx context.Context, in models.CreateInput) (models.Cart, error) {
	var regionID string
	if in.RegionID != nil && *in.RegionID != "" {
		regionID = *in.RegionID
	} else {
		stores, err := s.stores.List(ctx)
		if err != nil {
			return models.Cart{}, err
		}
		if len(stores) == 0 {
			return models.Cart{}, fmt.Errorf("store not found")
		}
		if stores[0].DefaultRegionID == nil {
			return models.Cart{}, fmt.Errorf("no regions found")
		}
		regionID = *stores[0].DefaultRegionID
	}
	region, err := s.regions.Get(ctx, regionID)
	if err != nil {
		return models.Cart{}, fmt.Errorf("no regions found")
	}
	currency := region.CurrencyCode
	if in.CurrencyCode != nil && !strings.EqualFold(*in.CurrencyCode, currency) {
		return models.Cart{}, fmt.Errorf("currency code %s does not match the region's currency code %s", *in.CurrencyCode, currency)
	}
	if in.SalesChannelID == nil || *in.SalesChannelID == "" {
		stores, e := s.stores.List(ctx)
		if e != nil {
			return models.Cart{}, e
		}
		if len(stores) == 0 || stores[0].DefaultSalesChannelID == nil {
			return models.Cart{}, fmt.Errorf("no sales channel found")
		}
		in.SalesChannelID = stores[0].DefaultSalesChannelID
	}
	channel, err := s.channels.Get(ctx, *in.SalesChannelID)
	if err != nil {
		return models.Cart{}, err
	}
	if channel.IsDisabled {
		return models.Cart{}, fmt.Errorf("unable to assign cart to disabled Sales Channel: %s", channel.Name)
	}
	in.RegionID = &region.ID
	in.CurrencyCode = &currency
	if len(in.ShippingAddress) == 0 && len(region.Countries) == 1 {
		in.ShippingAddress, _ = jsonAddress(region.Countries[0].ISO2)
	}
	return s.repo.Create(ctx, in)
}
func (s *Service) Update(ctx context.Context, id string, in models.UpdateInput) (models.Cart, error) {
	cart, err := s.repo.Get(ctx, id)
	if err != nil {
		return models.Cart{}, err
	}
	regionID := cart.RegionID
	if in.RegionID != nil {
		regionID = in.RegionID
	}
	var region *regionmodels.Region
	if regionID != nil && *regionID != "" {
		loaded, loadErr := s.regions.Get(ctx, *regionID)
		if loadErr != nil {
			return models.Cart{}, fmt.Errorf("region not found")
		}
		region = &loaded
		if in.RegionID != nil {
			in.CurrencyCode = &loaded.CurrencyCode
		}
	}
	regionChanged := in.RegionID != nil && (cart.RegionID == nil || *cart.RegionID != *in.RegionID)
	if in.ShippingAddress != nil && string(in.ShippingAddress) != "null" && region != nil {
		var address struct {
			CountryCode *string `json:"country_code"`
		}
		if err := json.Unmarshal(in.ShippingAddress, &address); err == nil && address.CountryCode != nil && *address.CountryCode != "" {
			matched := false
			for _, country := range region.Countries {
				if strings.EqualFold(country.ISO2, *address.CountryCode) {
					address.CountryCode = &country.ISO2
					matched = true
					break
				}
			}
			if !matched {
				return models.Cart{}, fmt.Errorf("country with code %s is not within region %s", *address.CountryCode, region.Name)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(in.ShippingAddress, &fields); err != nil {
				return models.Cart{}, err
			}
			countryJSON, _ := json.Marshal(*address.CountryCode)
			fields["country_code"] = countryJSON
			in.ShippingAddress, err = json.Marshal(fields)
			if err != nil {
				return models.Cart{}, err
			}
		}
	}
	if regionChanged && region != nil {
		if len(region.Countries) == 1 {
			in.ShippingAddress, err = json.Marshal(map[string]string{"country_code": region.Countries[0].ISO2})
			if err != nil {
				return models.Cart{}, err
			}
		} else {
			var address struct {
				CountryCode *string `json:"country_code"`
			}
			if len(in.ShippingAddress) == 0 || json.Unmarshal(in.ShippingAddress, &address) != nil || address.CountryCode == nil || *address.CountryCode == "" {
				in.ShippingAddress = json.RawMessage("null")
			}
		}
	}
	if in.SalesChannelID != nil {
		channel, err := s.channels.Get(ctx, *in.SalesChannelID)
		if err != nil {
			return models.Cart{}, err
		}
		if channel.IsDisabled {
			return models.Cart{}, fmt.Errorf("unable to assign cart to disabled Sales Channel: %s", channel.Name)
		}
	}
	return s.repo.Update(ctx, id, in)
}
