package workflows

import (
	"context"
	"errors"

	"github.com/example/medusa-go-clone/internal/modules/cart/domain/models"
)

const (
	CreateLineItemsStepID = "create-line-items-step"
	UpdateLineItemsStepID = "update-line-items-step"
	DeleteLineItemsStepID = "delete-line-items"
)

type LineItemService interface {
	CreateLineItems(context.Context, string, []models.LineItemInput) ([]models.LineItem, error)
	GetLineItem(context.Context, string, string) (models.LineItem, error)
	UpdateLineItem(context.Context, string, string, models.LineItemUpdate) (models.LineItem, error)
	SoftDeleteLineItems(context.Context, string, []string) ([]models.LineItem, error)
	RestoreLineItems(context.Context, []string) error
}

// CreateLineItemsStep follows Medusa's create-line-items-step action.
func CreateLineItemsStep(ctx context.Context, carts LineItemService, cartID string, items []models.LineItemInput) ([]models.LineItem, error) {
	return carts.CreateLineItems(ctx, cartID, items)
}

func CompensateCreatedLineItems(ctx context.Context, carts LineItemService, items []models.LineItem) error {
	if len(items)==0 { return nil }
	ids:=make([]string,0,len(items)); for _,item:=range items { ids=append(ids,item.ID) }
	_,err:=carts.SoftDeleteLineItems(ctx,items[0].CartID,ids)
	return err
}

// UpdateLineItemsStep follows Medusa's update-line-items-step action. It
// snapshots each old row so callers can compensate later workflow failures.
func UpdateLineItemsStep(ctx context.Context, carts LineItemService, cartID string, updates map[string]models.LineItemUpdate) ([]models.LineItem, []models.LineItem, error) {
	updated := make([]models.LineItem, 0, len(updates))
	previous := make([]models.LineItem, 0, len(updates))
	for id, input := range updates {
		old, err := carts.GetLineItem(ctx, cartID, id)
		if err != nil {
			return nil, previous, err
		}
		previous = append(previous, old)
		item, err := carts.UpdateLineItem(ctx, cartID, id, input)
		if err != nil {
			return nil, previous, err
		}
		updated = append(updated, item)
	}
	return updated, previous, nil
}

// DeleteLineItemsStep preserves Medusa's soft-delete action and returns the
// prior rows, which the caller can restore when a later step fails.
func DeleteLineItemsStep(ctx context.Context, carts LineItemService, cartID string, ids []string) ([]models.LineItem, error) {
	return carts.SoftDeleteLineItems(ctx, cartID, ids)
}

func RestoreLineItemsCompensation(ctx context.Context, carts LineItemService, snapshot []models.LineItem) error {
	ids := make([]string,0,len(snapshot))
	for _, item := range snapshot { ids=append(ids,item.ID) }
	if err := carts.RestoreLineItems(ctx, ids); err != nil {
		return errors.Join(errors.New("restore line items compensation failed"), err)
	}
	return nil
}

func CompensateUpdatedLineItems(ctx context.Context,carts LineItemService,snapshot []models.LineItem)error{
	for _,item:=range snapshot {
		quantity:=item.Quantity
		if _,err:=carts.UpdateLineItem(ctx,item.CartID,item.ID,models.LineItemUpdate{Quantity:&quantity,Metadata:item.Metadata,MetadataSet:true});err!=nil{return errors.Join(errors.New("restore updated line item compensation failed"),err)}
	}
	return nil
}
