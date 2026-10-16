package workflows

import (
	"context"

	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
)

const (
	GetOrdersListWorkflowID  = "get-orders-list"
	GetOrderDetailWorkflowID = "get-order-detail"
)

type Service interface {
	List(context.Context, models.Filter) (models.ListResult, error)
	Get(context.Context, string, *int) (models.Order, error)
}

func GetOrdersList(ctx context.Context, service Service, input models.Filter) (models.ListResult, error) {
	return service.List(ctx, input)
}
func GetOrderDetail(ctx context.Context, service Service, id string, version *int) (models.Order, error) {
	return service.Get(ctx, id, version)
}
