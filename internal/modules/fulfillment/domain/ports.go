package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	"time"
)

type Repository interface {
	ListShippingOptions(context.Context, models.ShippingOptionFilter, int, int) ([]models.ShippingOption, int, error)
	GetShippingOption(context.Context, string) (models.ShippingOption, error)
	CreateShippingOption(context.Context, models.ShippingOptionInput) (models.ShippingOption, error)
	UpdateShippingOption(context.Context, string, models.ShippingOptionUpdateInput) (models.ShippingOption, error)
	HardDeleteShippingOption(context.Context, string) error
	SoftDeleteShippingOptions(context.Context, []string) error
	RestoreShippingOptions(context.Context, []string) error
	PersistFulfillment(context.Context, models.FulfillmentCreateInput) (models.FulfillmentRecord, error)
	GetFulfillment(context.Context, string) (models.FulfillmentRecord, error)
	UpdateFulfillmentProviderResult(context.Context, string, models.FulfillmentProviderResult) (models.FulfillmentRecord, error)
	HardDeleteFulfillment(context.Context, string) error
	MarkFulfillmentCanceled(context.Context, string) (models.FulfillmentRecord, error)
	MarkFulfillmentShipped(context.Context, string, models.CreateShipmentInput, time.Time) (models.FulfillmentRecord, models.FulfillmentRecord, error)
	RestoreFulfillment(context.Context, models.FulfillmentRecord) error
	CanProviderServeZone(context.Context, string, string) (bool, error)
	BatchShippingOptionRules(context.Context, string, models.ShippingOptionRuleBatchInput) (models.ShippingOptionRuleBatchResult, error)
	CompensateShippingOptionRuleBatch(context.Context, models.ShippingOptionRuleBatchResult) error
	ListShippingOptionTypes(context.Context, models.ShippingOptionTypeFilter, int, int) ([]models.ShippingOptionType, int, error)
	GetShippingOptionType(context.Context, string) (models.ShippingOptionType, error)
	CreateShippingOptionTypes(context.Context, []models.ShippingOptionTypeInput) ([]models.ShippingOptionType, error)
	UpdateShippingOptionType(context.Context, string, models.ShippingOptionTypeUpdate) (models.ShippingOptionType, error)
	SoftDeleteShippingOptionTypes(context.Context, []string) error
	RestoreShippingOptionTypes(context.Context, []string) error
	HardDeleteShippingOptionTypes(context.Context, []string) error
	ShippingOptionTypeInUse(context.Context, string) (bool, error)
	ListShippingProfiles(context.Context, models.ShippingProfileFilter, int, int) ([]models.ShippingProfile, int, error)
	GetShippingProfile(context.Context, string) (models.ShippingProfile, error)
	CreateShippingProfiles(context.Context, []models.ShippingProfileInput) ([]models.ShippingProfile, error)
	UpdateShippingProfile(context.Context, string, map[string]any) (models.ShippingProfile, error)
	SoftDeleteShippingProfiles(context.Context, []string) error
	RestoreShippingProfiles(context.Context, []string) error
	HardDeleteShippingProfiles(context.Context, []string) error
	CreateSet(context.Context, models.CreateSetInput) (models.FulfillmentSet, error)
	GetSet(context.Context, string) (models.FulfillmentSet, error)
	DeleteSet(context.Context, string) error
	RestoreSet(context.Context, string) error
	CreateServiceZone(context.Context, string, models.ServiceZoneInput) (models.FulfillmentSet, error)
	GetServiceZone(context.Context, string) (models.ServiceZone, error)
	UpdateServiceZone(context.Context, string, models.ServiceZoneUpdate) (models.FulfillmentSet, error)
	DeleteServiceZone(context.Context, string) (models.FulfillmentSet, error)
}

type Provider interface {
	Identifier() string
	GetFulfillmentOptions(context.Context) ([]models.FulfillmentOption, error)
	ValidateFulfillmentData(context.Context, map[string]any, map[string]any, map[string]any) (map[string]any, error)
	CalculatePrice(context.Context, map[string]any, map[string]any, map[string]any) (models.ShippingPriceResult, error)
	CanCalculate(context.Context, map[string]any) (bool, error)
	ValidateOption(context.Context, map[string]any) (bool, error)
	CreateFulfillment(context.Context, models.CreateFulfillmentInput) (models.FulfillmentProviderResult, error)
	CancelFulfillment(context.Context, map[string]any) (map[string]any, error)
	CreateReturnFulfillment(context.Context, map[string]any) (models.FulfillmentProviderResult, error)
}
