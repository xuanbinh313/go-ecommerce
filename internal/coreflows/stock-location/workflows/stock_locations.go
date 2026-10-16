package workflows

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/stock-location/domain/models"
)

const CreateStockLocationsWorkflowID = "create-stock-locations-workflow"
const CreateStockLocationsStepID = "create-stock-locations"
const UpdateStockLocationsWorkflowID = "update-stock-locations-workflow"
const UpdateStockLocationsStepID = "update-stock-locations-step"
const DeleteStockLocationsWorkflowID = "delete-stock-locations-workflow"
const DeleteStockLocationsStepID = "delete-stock-locations-step"
const LinkSalesChannelsToStockLocationWorkflowID = "link-sales-channels-to-stock-location"

type Service interface {
	Create(context.Context, models.Input) (models.StockLocation, error)
	Update(context.Context, string, models.UpdateInput) (models.StockLocation, error)
	Delete(context.Context, string) error
	LinkSalesChannels(context.Context, string, []string, []string) error
}

func CreateStockLocations(c context.Context, s Service, in models.Input) (models.StockLocation, error) {
	return s.Create(c, in)
}
func UpdateStockLocations(c context.Context, s Service, id string, in models.UpdateInput) (models.StockLocation, error) {
	return s.Update(c, id, in)
}
func DeleteStockLocations(c context.Context, s Service, id string) error { return s.Delete(c, id) }
func LinkSalesChannelsToStockLocation(c context.Context, s Service, id string, add, remove []string) error {
	return s.LinkSalesChannels(c, id, add, remove)
}
