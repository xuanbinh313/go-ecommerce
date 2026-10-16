package workflows

import (
	"context"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
)

const CreateLocationFulfillmentSetWorkflowID = "create-location-fulfillment-set"

type FulfillmentSets interface {
	CreateSet(context.Context, models.CreateSetInput) (models.FulfillmentSet, error)
	DeleteSet(context.Context, string) error
}

type FulfillmentSetLinks interface {
	Create(context.Context, string, []map[string]any) ([]linkmodels.Link, error)
}

// CreateLocationFulfillmentSet follows Medusa's create-then-associate workflow.
func CreateLocationFulfillmentSet(ctx context.Context, sets FulfillmentSets, links FulfillmentSetLinks, locationID string, input models.CreateSetInput) (models.FulfillmentSet, error) {
	set, err := sets.CreateSet(ctx, input)
	if err != nil {
		return models.FulfillmentSet{}, err
	}
	_, err = links.Create(ctx, "LocationFulfillmentSet", []map[string]any{{"stock_location_id": locationID, "fulfillment_set_id": set.ID}})
	if err != nil {
		if compensationErr := sets.DeleteSet(ctx, set.ID); compensationErr != nil {
			return models.FulfillmentSet{}, compensationErr
		}
		return models.FulfillmentSet{}, err
	}
	return set, nil
}
