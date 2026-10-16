package workflows

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	customerdomain "github.com/example/medusa-go-clone/internal/modules/customer/domain"
	eventmodels "github.com/example/medusa-go-clone/internal/modules/event-bus-local/domain/models"
	orderdomain "github.com/example/medusa-go-clone/internal/modules/order/domain"
	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
)

const (
	UpdateOrderWorkflowID       = "update-order-workflow"
	UpdateOrderValidationStepID = "update-order-validation"
	UpdateOrdersStepID          = "update-orders"
	RegisterOrderChangesStepID  = "register-order-change"
	PreviewOrderChangeStepID    = "preview-order-change"
	OrderUpdatedEvent           = "order.updated"
)

type Updater interface {
	Get(context.Context, string, *int) (models.Order, error)
	Update(context.Context, models.UpdateInput) (models.Order, error)
}

func UpdateOrder(ctx context.Context, service Updater, customers customerdomain.OrderCustomerResolver, products ProductReader, translations TranslationReader, translationEnabled bool, events EventPublisher, input models.UpdateInput) (models.Order, error) {
	current, err := service.Get(ctx, input.ID, nil)
	if err != nil {
		return models.Order{}, err
	}
	if current.Status == "canceled" {
		return models.Order{}, orderdomain.InvalidData(fmt.Sprintf("Order with id %s has been canceled.", input.ID))
	}
	if input.Email != nil {
		parsed, e := mail.ParseAddress(*input.Email)
		if e != nil || parsed.Address != *input.Email {
			return models.Order{}, orderdomain.InvalidData("Invalid email address")
		}
	}
	if input.ShippingAddress != nil && input.ShippingAddress.CountryCode != nil && *input.ShippingAddress.CountryCode != "" && (current.ShippingAddress == nil || current.ShippingAddress.CountryCode == nil || *current.ShippingAddress.CountryCode != *input.ShippingAddress.CountryCode) {
		return models.Order{}, orderdomain.InvalidData("Country code cannot be changed")
	}
	if input.BillingAddress != nil && input.BillingAddress.CountryCode != nil && *input.BillingAddress.CountryCode != "" && (current.BillingAddress == nil || current.BillingAddress.CountryCode == nil || *current.BillingAddress.CountryCode != *input.BillingAddress.CountryCode) {
		return models.Order{}, orderdomain.InvalidData("Country code cannot be changed")
	}
	groupID := entityid.New("eventgroup")
	var createdCustomerID *string
	if (current.Email == nil || *current.Email == "") && input.Email != nil && *input.Email != "" {
		customer, created, resolveErr := FindOrCreateCustomerByEmail(ctx, customers, strings.ToLower(*input.Email))
		if resolveErr != nil {
			return models.Order{}, resolveErr
		}
		input.CustomerID = &customer.ID
		if created {
			createdCustomerID = &customer.ID
			if err = events.Emit(ctx, []eventmodels.Event{{Name: "customer.created", Data: map[string]any{"id": customer.ID}, Metadata: map[string]any{"eventGroupId": groupID}}}, eventmodels.Options{}); err != nil {
				events.ClearGroupedEvents(groupID, []string{"customer.created"})
				if compensationErr := customers.Delete(ctx, customer.ID); compensationErr != nil {
					return models.Order{}, compensationErr
				}
				return models.Order{}, err
			}
		}
	}
	if translationEnabled&&input.Locale!=nil&&*input.Locale!=""&&(current.Locale==nil||*current.Locale!=*input.Locale){
		itemTranslations,translationErr:=PrepareOrderItemTranslations(ctx,current.Items,*input.Locale,products,translations)
		if translationErr!=nil{if createdCustomerID!=nil{if compErr:=customers.Delete(ctx,*createdCustomerID);compErr!=nil{return models.Order{},compErr}};return models.Order{},translationErr}
		input.ItemTranslations=itemTranslations
	}
	updated, err := service.Update(ctx, input)
	if err != nil {
		events.ClearGroupedEvents(groupID, []string{"customer.created"})
		if createdCustomerID != nil {
			if compensationErr := customers.Delete(ctx, *createdCustomerID); compensationErr != nil {
				return models.Order{}, compensationErr
			}
		}
		return models.Order{}, err
	}
	if err = events.Emit(ctx, []eventmodels.Event{{Name: OrderUpdatedEvent, Data: map[string]any{"id": input.ID}, Metadata: map[string]any{"eventGroupId": groupID}}}, eventmodels.Options{}); err != nil {
		events.ClearGroupedEvents(groupID, []string{OrderUpdatedEvent, "customer.created"})
		return models.Order{}, err
	}
	if err = events.ReleaseGroupedEvents(ctx, groupID); err != nil {
		events.ClearGroupedEvents(groupID, []string{OrderUpdatedEvent, "customer.created"})
		return models.Order{}, err
	}
	return updated, nil
}
