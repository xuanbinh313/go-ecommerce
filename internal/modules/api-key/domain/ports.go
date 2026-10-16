package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/api-key/domain/models"
)

type Repository interface {
	List(context.Context, models.Filter) ([]models.APIKey, int, error)
	Get(context.Context, string) (models.APIKey, error)
	Create(context.Context, models.APIKey) (models.APIKey, error)
	Update(context.Context, string, models.UpdateInput) error
	Revoke(context.Context, string, models.RevokeInput) error
	Delete(context.Context, string) error
	DeleteSalesChannelLinks(context.Context, string, []string) error
	CreateSalesChannelLinks(context.Context, string, []string) error
	SalesChannelsExist(context.Context, []string) (bool, error)
}
