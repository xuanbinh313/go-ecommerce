package workflows

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/pricing/domain/models"
)

const CreatePriceListsWorkflowID = "create-price-lists"
const UpdatePriceListsWorkflowID = "update-price-lists"
const DeletePriceListsWorkflowID = "delete-price-lists"

type Service interface {
	CreatePriceList(context.Context, models.PriceList) (models.PriceList, error)
	UpdatePriceList(context.Context, string, models.PriceList) (models.PriceList, error)
	DeletePriceList(context.Context, string) error
}

func CreatePriceList(ctx context.Context, svc Service, in models.PriceList) (models.PriceList, error) {
	return svc.CreatePriceList(ctx, in)
}
func UpdatePriceList(ctx context.Context, svc Service, id string, in models.PriceList) (models.PriceList, error) {
	return svc.UpdatePriceList(ctx, id, in)
}
func DeletePriceList(ctx context.Context, svc Service, id string) error {
	return svc.DeletePriceList(ctx, id)
}
