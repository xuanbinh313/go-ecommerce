package workflows

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/pricing/domain/models"
)

const CreatePricePreferencesWorkflowID = "create-price-preferences"
const UpdatePricePreferencesWorkflowID = "update-price-preferences"
const DeletePricePreferencesWorkflowID = "delete-price-preferences"

type PreferenceService interface {
	CreatePreference(context.Context, models.PreferenceInput) (models.PricePreference, error)
	UpdatePreference(context.Context, string, models.PreferenceUpdate) (models.PricePreference, error)
	DeletePreference(context.Context, string) error
}

func CreatePricePreference(ctx context.Context, svc PreferenceService, in models.PreferenceInput) (models.PricePreference, error) {
	return svc.CreatePreference(ctx, in)
}
func UpdatePricePreference(ctx context.Context, svc PreferenceService, id string, in models.PreferenceUpdate) (models.PricePreference, error) {
	return svc.UpdatePreference(ctx, id, in)
}
func DeletePricePreference(ctx context.Context, svc PreferenceService, id string) error {
	return svc.DeletePreference(ctx, id)
}
