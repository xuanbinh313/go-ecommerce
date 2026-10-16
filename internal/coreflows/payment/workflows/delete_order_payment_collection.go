package workflows

import (
	"context"
	"errors"
	"fmt"

	linkModels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	paymentModels "github.com/example/medusa-go-clone/internal/modules/payment/domain/models"
)

const DeleteOrderPaymentCollectionsWorkflowID = "delete-order-payment-collectionworkflow"

type PaymentCollectionReader interface {
	GetPaymentCollection(context.Context, string) (*paymentModels.PaymentCollection, error)
}
type PaymentRemoteLinks interface {
	Dismiss(context.Context, string, []map[string]any) ([]linkModels.Link, error)
	Restore(context.Context, string, []map[string]any) ([]linkModels.Link, error)
}

// DeleteOrderPaymentCollection removes the remote associations, as Medusa does;
// it does not delete the Payment module's collection record.
func DeleteOrderPaymentCollection(ctx context.Context, payments PaymentCollectionReader, links PaymentRemoteLinks, id string) error {
	collection, err := payments.GetPaymentCollection(ctx, id)
	if err != nil {
		return err
	}
	if collection.Status != paymentModels.PaymentCollectionNotPaid {
		return fmt.Errorf("can only delete payment collections where status is not_paid")
	}
	definitions := []string{"CartPaymentCollection", "OrderClaimPaymentCollection", "OrderExchangePaymentCollection", "OrderPaymentCollection"}
	dismissed := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		_, err = links.Dismiss(ctx, definition, []map[string]any{{"payment_collection_id": id}})
		if err != nil {
			var compensationErrors []error
			for i := len(dismissed) - 1; i >= 0; i-- {
				if _, restoreErr := links.Restore(ctx, dismissed[i], []map[string]any{{"payment_collection_id": id}}); restoreErr != nil {
					compensationErrors = append(compensationErrors, restoreErr)
				}
			}
			return errors.Join(append([]error{err}, compensationErrors...)...)
		}
		dismissed = append(dismissed, definition)
	}
	return nil
}
