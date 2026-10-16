package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/sales-channel/domain/models"
)

type Repository interface {
	Create(context.Context, models.Input) (models.SalesChannel, error)
	Get(context.Context, string) (models.SalesChannel, error)
	List(context.Context) ([]models.SalesChannel, error)
	Update(context.Context, string, models.Input) (models.SalesChannel, error)
	Delete(context.Context, string) error
	Restore(context.Context, []string) error
	DeleteMany(context.Context, []string) error
}
