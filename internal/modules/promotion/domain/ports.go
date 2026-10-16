package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/promotion/domain/models"
)

type Repository interface {
	ListCampaigns(context.Context, models.CampaignFilter) ([]models.Campaign, int, error)
	GetCampaign(context.Context, string) (models.Campaign, error)
	CreateCampaign(context.Context, models.CampaignInput) (models.Campaign, error)
	UpdateCampaign(context.Context, string, models.CampaignUpdate) (models.Campaign, error)
	DeleteCampaign(context.Context, string) error
	LinkCampaignPromotions(context.Context, string, []string, []string) error
	List(context.Context, models.Filter) ([]models.Promotion, int, error)
	Get(context.Context, string) (models.Promotion, error)
	Create(context.Context, models.CreateInput) (models.Promotion, error)
	Update(context.Context, string, models.UpdateInput) (models.Promotion, error)
	Delete(context.Context, string) error
}
