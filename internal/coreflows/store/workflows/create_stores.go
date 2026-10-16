package workflows

import (
	"context"
	"errors"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/store/domain/models"
)

const CreateStoresWorkflowID = "create-stores"
const CreateStoresStepID = "create-stores"
const UpdatePricePreferencesAsArrayStepID = "update-price-preferences-as-array"

type StoreCreator interface {
	Create(context.Context, models.StoreInput) (models.Store, error)
}
type StoreDeleter interface {
	DeleteStores(context.Context, []string) error
}
type PricePreference struct {
	Attribute      string `json:"attribute"`
	Value          string `json:"value"`
	IsTaxInclusive *bool  `json:"is_tax_inclusive,omitempty"`
}
type PricePreferenceWriter interface {
	UpsertCurrencyPreferences(context.Context, []PricePreference) error
}

// CreateStores follows Medusa's create-stores order: persist stores, then upsert
// one currency preference per distinct supported currency.
func CreateStores(ctx context.Context, stores StoreCreator, prices PricePreferenceWriter, inputs []models.StoreInput) ([]models.Store, error) {
	created := make([]models.Store, 0, len(inputs))
	compensate := func(cause error) error {
		deleter, ok := stores.(StoreDeleter)
		if !ok || len(created) == 0 {
			return cause
		}
		ids := make([]string, 0, len(created))
		for _, store := range created {
			ids = append(ids, store.ID)
		}
		if err := deleter.DeleteStores(ctx, ids); err != nil {
			return errors.Join(cause, fmt.Errorf("create-stores compensation: %w", err))
		}
		return cause
	}
	for _, input := range inputs {
		store, err := stores.Create(ctx, input)
		if err != nil {
			return nil, compensate(err)
		}
		created = append(created, store)
	}
	currencies := []PricePreference{}
	byCurrency := map[string]int{}
	for _, input := range inputs {
		for _, currency := range input.SupportedCurrencies {
			if index, exists := byCurrency[currency.CurrencyCode]; exists {
				currencies[index].IsTaxInclusive = currency.IsTaxInclusive
			} else {
				byCurrency[currency.CurrencyCode] = len(currencies)
				currencies = append(currencies, PricePreference{Attribute: "currency_code", Value: currency.CurrencyCode, IsTaxInclusive: currency.IsTaxInclusive})
			}
		}
	}
	if prices != nil && len(currencies) > 0 {
		if err := prices.UpsertCurrencyPreferences(ctx, currencies); err != nil {
			return nil, compensate(fmt.Errorf("create-stores: update price preferences: %w", err))
		}
	} else if prices == nil && len(currencies) > 0 {
		return nil, compensate(errors.New("create-stores: price preference writer is required"))
	}
	return created, nil
}
