package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/stock-location/domain/models"
)

type Repository interface {
	List(context.Context, models.Filter) ([]models.StockLocation, int, error)
	Get(context.Context, string) (models.StockLocation, error)
	Create(context.Context, models.Input) (models.StockLocation, error)
	Update(context.Context, string, models.UpdateInput) (models.StockLocation, error)
	Delete(context.Context, string) error
	LinkSalesChannels(context.Context, string, []string, []string) error
}
