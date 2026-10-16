package workflows

import (
	"context"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/cart/domain/models"
	customerdomain "github.com/example/medusa-go-clone/internal/modules/customer/domain"
)

const CreateCartWorkflowID = "create-cart"
const CreateCartsStepID = "create-carts"
const UpdateCartWorkflowID = "update-cart"
const UpdateCartStepID = "update-cart"
const TransferCartCustomerWorkflowID = "transfer-cart-customer"
const TransferCartCustomerStepID = "update-carts"
const CartCustomerTransferredEvent = "cart.customer_transferred"

type Service interface {
	Create(context.Context, models.CreateInput) (models.Cart, error)
	Update(context.Context, string, models.UpdateInput) (models.Cart, error)
}

func CreateCart(ctx context.Context, svc Service, in models.CreateInput) (models.Cart, error) {
	return svc.Create(ctx, in)
}
func UpdateCart(ctx context.Context, svc Service, id string, in models.UpdateInput) (models.Cart, error) {
	return svc.Update(ctx, id, in)
}

type CartCustomerEventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}

type CartTransferService interface {
	Get(context.Context, string) (models.Cart, error)
	Update(context.Context, string, models.UpdateInput) (models.Cart, error)
}

func TransferCartCustomer(ctx context.Context, carts CartTransferService, customers customerdomain.CartCustomerResolver, events CartCustomerEventPublisher, cartID, customerID string) (models.Cart, error) {
	cart, err := carts.Get(ctx, cartID)
	if err != nil {
		return models.Cart{}, err
	}
	customer, err := customers.Get(ctx, customerID)
	if err != nil {
		return models.Cart{}, err
	}
	if cart.CustomerID != nil && *cart.CustomerID == customer.ID {
		return cart, nil
	}
	updated, err := carts.Update(ctx, cartID, models.UpdateInput{CustomerID: &customer.ID, CustomerIDSet: true, Email: customer.Email, EmailSet: true})
	if err != nil {
		return models.Cart{}, err
	}
	if events != nil {
		if err = events.Publish(ctx, CartCustomerTransferredEvent, []map[string]any{{"id": cartID, "customer_id": customer.ID}}); err != nil {
			_, rollbackErr := carts.Update(ctx, cartID, models.UpdateInput{CustomerID: cart.CustomerID, CustomerIDSet: true, Email: cart.Email, EmailSet: true})
			if rollbackErr != nil {
				return models.Cart{}, fmt.Errorf("publish cart transfer event: %w (compensation failed: %v)", err, rollbackErr)
			}
			return models.Cart{}, err
		}
	}
	return updated, nil
}
