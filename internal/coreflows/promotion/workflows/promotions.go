package workflows

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/promotion/domain/models"
)

const CreatePromotionsWorkflowID = "create-promotions"
const CreatePromotionsStepID = "create-promotions"
const UpdatePromotionsWorkflowID = "update-promotions"
const UpdatePromotionsStepID = "update-promotions"
const DeletePromotionsWorkflowID = "delete-promotions"
const DeletePromotionsStepID = "delete-promotions"
const UpdatePromotionsStatusWorkflowID = "update-promotions-status"
const CreateCampaignsWorkflowID = "create-campaigns"
const CreateCampaignsStepID = "create-campaigns"
const UpdateCampaignsWorkflowID = "update-campaigns"
const UpdateCampaignsStepID = "update-campaigns"
const DeleteCampaignsWorkflowID = "delete-campaigns"
const DeleteCampaignsStepID = "delete-campaigns"
const AddOrRemoveCampaignPromotionsWorkflowID = "add-or-remove-campaign-promotions"
const AddCampaignPromotionsStepID = "add-campaign-promotions"
const RemoveCampaignPromotionsStepID = "remove-campaign-promotions"

type Service interface {
	Create(context.Context, models.CreateInput) (models.Promotion, error)
	Update(context.Context, string, models.UpdateInput) (models.Promotion, error)
	Delete(context.Context, string) error
}
type CampaignService interface {
	CreateCampaign(context.Context, models.CampaignInput) (models.Campaign, error)
	UpdateCampaign(context.Context, string, models.CampaignUpdate) (models.Campaign, error)
	DeleteCampaign(context.Context, string) error
	LinkCampaignPromotions(context.Context, string, []string, []string) error
}

func CreatePromotion(c context.Context, s Service, in models.CreateInput) (models.Promotion, error) {
	return s.Create(c, in)
}
func UpdatePromotion(c context.Context, s Service, id string, in models.UpdateInput) (models.Promotion, error) {
	return s.Update(c, id, in)
}
func DeletePromotion(c context.Context, s Service, id string) error { return s.Delete(c, id) }

func CreateCampaign(c context.Context, s CampaignService, in models.CampaignInput) (models.Campaign, error) {
	return s.CreateCampaign(c, in)
}
func UpdateCampaign(c context.Context, s CampaignService, id string, in models.CampaignUpdate) (models.Campaign, error) {
	return s.UpdateCampaign(c, id, in)
}
func DeleteCampaign(c context.Context, s CampaignService, id string) error {
	return s.DeleteCampaign(c, id)
}
func AddOrRemoveCampaignPromotions(c context.Context, s CampaignService, id string, add, remove []string) error {
	return s.LinkCampaignPromotions(c, id, add, remove)
}
