package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/pricing/domain/models"
)

type Repository interface {
	ListPreferences(context.Context, []string, []string, int, int) ([]models.PricePreference, int, error)
	GetPreference(context.Context, string) (models.PricePreference, error)
	CreatePreference(context.Context, models.PreferenceInput) (models.PricePreference, error)
	UpdatePreference(context.Context, string, models.PreferenceUpdate) (models.PricePreference, error)
	DeletePreference(context.Context, string) error
	ListPriceLists(context.Context, []string, []string, []string, int, int) ([]models.PriceList, int, error)
	GetPriceList(context.Context, string) (models.PriceList, error)
	CreatePriceList(context.Context, models.PriceList) (models.PriceList, error)
	UpdatePriceList(context.Context, string, models.PriceList) (models.PriceList, error)
	DeletePriceList(context.Context, string) error
}
