package workflows

import (
	"context"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
)

const (
	CreateServiceZonesWorkflowID    = "create-service-zones-workflow"
	CreateServiceZonesStepID        = "create-service-zones"
	UpdateServiceZonesWorkflowID    = "update-service-zones-workflow"
	UpdateServiceZonesStepID        = "update-service-zones"
	DeleteServiceZonesWorkflowID    = "delete-service-zones-workflow"
	DeleteServiceZonesStepID        = "delete-service-zones"
	DeleteFulfillmentSetsWorkflowID = "delete-fulfillment-sets-workflow"
	DeleteFulfillmentSetsStepID     = "delete-fulfillment-sets"
)

type Service interface {
	CreateServiceZone(context.Context, string, models.ServiceZoneInput) (models.FulfillmentSet, error)
	UpdateServiceZone(context.Context, string, models.ServiceZoneUpdate) (models.FulfillmentSet, error)
	DeleteServiceZone(context.Context, string) (models.FulfillmentSet, error)
	DeleteSet(context.Context, string) error
	RestoreSet(context.Context, string) error
}
type Links interface {
	Dismiss(context.Context, string, []map[string]any) ([]linkmodels.Link, error)
	Restore(context.Context, string, []map[string]any) ([]linkmodels.Link, error)
}

func CreateServiceZones(ctx context.Context, service Service, setID string, input models.ServiceZoneInput) (models.FulfillmentSet, error) {
	return service.CreateServiceZone(ctx, setID, input)
}
func UpdateServiceZones(ctx context.Context, service Service, zoneID string, input models.ServiceZoneUpdate) (models.FulfillmentSet, error) {
	return service.UpdateServiceZone(ctx, zoneID, input)
}
func DeleteServiceZones(ctx context.Context, service Service, zoneID string) (models.FulfillmentSet, error) {
	return service.DeleteServiceZone(ctx, zoneID)
}
func DeleteFulfillmentSets(ctx context.Context, service Service, links Links, id string) error {
	if err := service.DeleteSet(ctx, id); err != nil {
		return err
	}
	filters := []map[string]any{{"fulfillment_set_id": id}}
	if _, err := links.Dismiss(ctx, "LocationFulfillmentSet", filters); err != nil {
		_, _ = links.Restore(ctx, "LocationFulfillmentSet", filters)
		if restoreErr := service.RestoreSet(ctx, id); restoreErr != nil {
			return restoreErr
		}
		return err
	}
	return nil
}
