package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/cart/domain/models"
	regionmodels "github.com/example/medusa-go-clone/internal/modules/region/domain/models"
	salesmodels "github.com/example/medusa-go-clone/internal/modules/sales-channel/domain/models"
	storemodels "github.com/example/medusa-go-clone/internal/modules/store/domain/models"
)

type Repository interface {
	Create(context.Context, models.CreateInput) (models.Cart, error)
	Get(context.Context, string) (models.Cart, error)
	CreateLineItems(context.Context, string, []models.LineItemInput) ([]models.LineItem, error)
	GetLineItem(context.Context, string, string) (models.LineItem, error)
	UpdateLineItem(context.Context, string, string, models.LineItemUpdate) (models.LineItem, error)
	SoftDeleteLineItems(context.Context, string, []string) ([]models.LineItem, error)
	RestoreLineItems(context.Context, []string) error
	Update(context.Context, string, models.UpdateInput) (models.Cart, error)
}
type RegionLookup interface {
	Get(context.Context, string) (regionmodels.Region, error)
}
type StoreLookup interface {
	List(context.Context) ([]storemodels.Store, error)
}
type SalesChannelLookup interface {
	Get(context.Context, string) (salesmodels.SalesChannel, error)
}
