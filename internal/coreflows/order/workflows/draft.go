package workflows

import "context"

const DeleteDraftOrderWorkflowID = "delete-draft-order"
const DeleteDraftOrdersStepID = "delete-draft-orders"

type DraftOrderDeleteService interface {
	DeleteDraftOrders(context.Context, []string) error
}

func DeleteDraftOrders(ctx context.Context, service DraftOrderDeleteService, ids []string) error {
	return service.DeleteDraftOrders(ctx, ids)
}
