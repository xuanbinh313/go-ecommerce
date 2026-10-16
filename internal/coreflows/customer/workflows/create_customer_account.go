package workflows

import (
	"context"
	"errors"

	authdomain "github.com/example/medusa-go-clone/internal/modules/auth/domain"
	customerdomain "github.com/example/medusa-go-clone/internal/modules/customer/domain"
	"github.com/example/medusa-go-clone/internal/modules/customer/domain/models"
)

type CreateCustomerAccountInput struct {
	AuthIdentityID string       `json:"authIdentityId"`
	CustomerData   models.Input `json:"customerData"`
}

const CreateCustomerAccountWorkflowID = "create-customer-account"
const ValidateCustomerAccountCreationStepID = "validate-customer-account-creation"
const CreateCustomersWorkflowID = "create-customers"
const CreateCustomersStepID = "create-customers"
const SetAuthAppMetadataStepID = "set-auth-app-metadata"
const EmitEventStepID = "emit-event-step"
const CustomerCreatedEvent = "customer.created"

type EventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}

// CreateCustomerAccount creates the customer and links it to the auth identity.
// If linking fails, it compensates by deleting the customer it just created.
func CreateCustomerAccount(ctx context.Context, customers customerdomain.AccountService, auth authdomain.IdentityMetadata, in CreateCustomerAccountInput, publishers ...EventPublisher) (models.Customer, error) {
	if err := customers.ValidateAccountCreation(ctx, in.CustomerData.Email, in.AuthIdentityID); err != nil {
		return models.Customer{}, err
	}
	in.CustomerData.HasAccount = in.AuthIdentityID != ""
	created, err := customers.Create(ctx, in.CustomerData)
	if err != nil {
		return models.Customer{}, err
	}
	// createCustomersWorkflow emits customer.created before the auth metadata step.
	if len(publishers) > 0 && publishers[0] != nil {
		if err := publishers[0].Publish(ctx, CustomerCreatedEvent, []map[string]any{{"id": created.ID}}); err != nil {
			if rollbackErr := customers.Delete(ctx, created.ID); rollbackErr != nil {
				return models.Customer{}, errors.Join(err, rollbackErr)
			}
			return models.Customer{}, err
		}
	}
	if _, err = auth.SetActorMetadata(ctx, in.AuthIdentityID, "customer", created.ID); err != nil {
		if rollbackErr := customers.Delete(ctx, created.ID); rollbackErr != nil {
			return models.Customer{}, errors.Join(err, rollbackErr)
		}
		return models.Customer{}, err
	}
	return created, nil
}
