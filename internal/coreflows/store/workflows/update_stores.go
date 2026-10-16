package workflows

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/store/domain/models"
)

const UpdateStoresWorkflowID = "update-stores"
const UpdateStoresStepID = "update-stores"

type StoreUpdater interface {
	Get(context.Context, string) (models.Store, error)
	Update(context.Context, string, models.StoreInput) (models.Store, error)
}

// UpdateStores preserves Medusa's update-stores step and price preference step
// ordering. The repository update is restored if a later preference step fails.
func UpdateStores(ctx context.Context, stores StoreUpdater, prices PricePreferenceWriter, id string, input models.StoreInput) ([]models.Store, error) {
	previous, err := stores.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	updated, err := stores.Update(ctx, id, input)
	if err != nil {
		return nil, err
	}
	if len(input.SupportedCurrencies) == 0 {
		return []models.Store{updated}, nil
	}
	if prices == nil {
		return nil, compensateStore(ctx, stores, id, previous, errors.New("update-stores: price preference writer is required"))
	}
	preferences := make([]PricePreference, 0, len(input.SupportedCurrencies))
	for _, currency := range input.SupportedCurrencies {
		preferences = append(preferences, PricePreference{Attribute: "currency_code", Value: currency.CurrencyCode, IsTaxInclusive: currency.IsTaxInclusive})
	}
	if err := prices.UpsertCurrencyPreferences(ctx, preferences); err != nil {
		return nil, compensateStore(ctx, stores, id, previous, fmt.Errorf("update-stores: update price preferences: %w", err))
	}
	return []models.Store{updated}, nil
}

func compensateStore(ctx context.Context, stores StoreUpdater, id string, previous models.Store, cause error) error {
	input := models.StoreInput{ID: &previous.ID, Name: &previous.Name, DefaultSalesChannelID: previous.DefaultSalesChannelID, DefaultRegionID: previous.DefaultRegionID, DefaultLocationID: previous.DefaultLocationID, Metadata: previous.Metadata}
	for _, currency := range previous.SupportedCurrencies {
		input.SupportedCurrencies = append(input.SupportedCurrencies, models.StoreCurrencyInput{CurrencyCode: currency.CurrencyCode, IsDefault: currency.IsDefault})
	}
	for _, locale := range previous.SupportedLocales {
		input.SupportedLocales = append(input.SupportedLocales, models.StoreLocaleInput{LocaleCode: locale.LocaleCode})
	}
	if _, err := stores.Update(ctx, id, input); err != nil {
		return errors.Join(cause, fmt.Errorf("update-stores compensation: %w", err))
	}
	return cause
}
