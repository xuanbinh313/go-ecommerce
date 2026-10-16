package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/store/domain/models"
)

type Repository interface {
	Create(context.Context, models.StoreInput) (models.Store, error)
	Get(context.Context, string) (models.Store, error)
	List(context.Context) ([]models.Store, error)
	Update(context.Context, string, models.StoreInput) (models.Store, error)
	Delete(context.Context, string) error
	DeleteStores(context.Context, []string) error
}
