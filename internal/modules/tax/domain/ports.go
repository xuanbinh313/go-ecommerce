package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/tax/domain/models"
)

type Repository interface {
	ListProviders(context.Context, []string, *bool, int, int) ([]models.TaxProvider, int, error)
	ListRegions(context.Context, []string, []string, []string, int, int) ([]models.TaxRegion, int, error)
	GetRegion(context.Context, string) (models.TaxRegion, error)
	CreateRegion(context.Context, models.RegionInput) (models.TaxRegion, error)
	UpdateRegion(context.Context, string, models.RegionUpdate) (models.TaxRegion, error)
	DeleteRegion(context.Context, string) error
	ListRates(context.Context, []string, []string, []string, int, int) ([]models.TaxRate, int, error)
	GetRate(context.Context, string) (models.TaxRate, error)
	CreateRate(context.Context, models.RateInput, string, *string) (models.TaxRate, error)
	UpdateRate(context.Context, string, models.RateUpdate, *string) (models.TaxRate, error)
	DeleteRate(context.Context, string) error
	CreateRateRules(context.Context, []models.RuleCreateInput) error
	DeleteRateRules(context.Context, []string) error
}
