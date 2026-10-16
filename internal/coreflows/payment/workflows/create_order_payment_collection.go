package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	linkModels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	orderModels "github.com/example/medusa-go-clone/internal/modules/order/domain/models"
	paymentModels "github.com/example/medusa-go-clone/internal/modules/payment/domain/models"
)

const (
	CreateOrderPaymentCollectionWorkflowID = "create-order-payment-collection"
	GetOrderStepID                         = "get-order"
	CreatePaymentCollectionsStepID         = "create-payment-collections"
	OrderPaymentCollectionLinkStepID       = "order-payment-collection-link"
)

type OrderReader interface {
	Get(context.Context, string, *int) (orderModels.Order, error)
}
type PaymentCollections interface {
	CreatePaymentCollections(context.Context, []paymentModels.CreatePaymentCollectionInput) ([]paymentModels.PaymentCollection, error)
	DeletePaymentCollections(context.Context, []string) error
}
type PaymentCollectionLinks interface {
	Create(context.Context, string, []map[string]any) ([]linkModels.Link, error)
}

type CreateOrderPaymentCollectionInput struct {
	OrderID string               `json:"order_id"`
	Amount  paymentModels.Amount `json:"amount"`
}

// CreateOrderPaymentCollection follows Medusa's get-order -> create-payment-collections
// -> remote-link sequence. If linking fails, it compensates the created collection.
func CreateOrderPaymentCollection(ctx context.Context, orders OrderReader, payments PaymentCollections, links PaymentCollectionLinks, input CreateOrderPaymentCollectionInput) ([]paymentModels.PaymentCollection, error) {
	if input.OrderID == "" {
		return nil, fmt.Errorf("order_id is required")
	}
	order, err := orders.Get(ctx, input.OrderID, nil)
	if err != nil {
		return nil, err
	}
	rawAmount, err := json.Marshal(map[string]any{"value": string(input.Amount), "precision": 20})
	if err != nil {
		return nil, err
	}
	collections, err := payments.CreatePaymentCollections(ctx, []paymentModels.CreatePaymentCollectionInput{{CurrencyCode: order.CurrencyCode, Amount: input.Amount, RawAmount: rawAmount}})
	if err != nil {
		return nil, err
	}
	if len(collections) == 0 {
		return nil, fmt.Errorf("payment module created no payment collection")
	}
	_, err = links.Create(ctx, "OrderPaymentCollection", []map[string]any{{"order_id": order.ID, "payment_collection_id": collections[0].ID}})
	if err != nil {
		if compensationErr := payments.DeletePaymentCollections(ctx, []string{collections[0].ID}); compensationErr != nil {
			return nil, errors.Join(err, fmt.Errorf("compensate create-payment-collections: %w", compensationErr))
		}
		return nil, err
	}
	return collections, nil
}
