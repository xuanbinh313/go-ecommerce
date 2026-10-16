package workflows

import (
	"context"
	"encoding/json"
	"errors"

	authdomain "github.com/example/medusa-go-clone/internal/modules/auth/domain"
	"github.com/example/medusa-go-clone/internal/modules/customer/domain/models"
)

type CustomerMutations interface {
	Create(context.Context, models.Input) (models.Customer, error)
	Get(context.Context, string) (models.Customer, error)
	Update(context.Context, string, models.Input) (models.Customer, error)
	Delete(context.Context, string) error
}

type CustomerGroupLinks interface {
	AddCustomerToGroup(context.Context, string, string, *string) (string, error)
	RemoveCustomerFromGroup(context.Context, string, string) error
}

type CustomerAccountRemoval interface {
	CustomerMutations
	CustomerGroupLinks
	RestoreCustomer(context.Context, string) error
}

type CustomerAuthIdentity interface {
	FindIdentityByActor(context.Context, string, string) (authdomain.AuthIdentityRecord, error)
	UnsetActorMetadata(context.Context, string, string) error
	DeleteIdentity(context.Context, string) error
}

const (
	UpdateCustomersWorkflowID              = "update-customers"
	UpdateCustomersStepID                  = "update-customer"
	DeleteCustomersWorkflowID              = "delete-customers"
	DeleteCustomersStepID                  = "delete-customers"
	CustomerUpdatedEvent                   = "customer.updated"
	CustomerDeletedEvent                   = "customer.deleted"
	RemoveCustomerAccountWorkflowID        = "remove-customer-account"
	GetCustomerAuthIdentityStepID          = "get-customer-auth-identity"
	CreateCustomerGroupsWorkflowID         = "create-customer-groups"
	CreateCustomerGroupsStepID             = "create-customer-groups"
	UpdateCustomerGroupsWorkflowID         = "update-customer-groups"
	UpdateCustomerGroupStepID              = "update-customer-groups"
	DeleteCustomerGroupsWorkflowID         = "delete-customer-groups"
	DeleteCustomerGroupsStepID             = "delete-customer-groups"
	LinkCustomersToCustomerGroupWorkflowID = "link-customers-to-customer-group"
	LinkCustomersToCustomerGroupStepID     = "link-customers-to-customer-group"
)

type CustomerGroupMutations interface {
	CreateCustomerGroups(context.Context, string, json.RawMessage, *string) (models.CustomerGroup, error)
	GetCustomerGroup(context.Context, string) (models.CustomerGroup, error)
	UpdateCustomerGroup(context.Context, string, models.CustomerGroupUpdate) (models.CustomerGroup, error)
	DeleteCustomerGroup(context.Context, string) error
	RestoreCustomerGroup(context.Context, string) error
}

func CreateCustomerGroup(ctx context.Context, groups CustomerGroupMutations, name string, metadata json.RawMessage, createdBy *string) (models.CustomerGroup, error) {
	return groups.CreateCustomerGroups(ctx, name, metadata, createdBy)
}

func UpdateCustomerGroup(ctx context.Context, groups CustomerGroupMutations, id string, update models.CustomerGroupUpdate) (models.CustomerGroup, error) {
	previous, err := groups.GetCustomerGroup(ctx, id)
	if err != nil {
		return models.CustomerGroup{}, err
	}
	updated, err := groups.UpdateCustomerGroup(ctx, id, update)
	if err != nil {
		return models.CustomerGroup{}, err
	}
	_ = previous // The source workflow's update step only needs its snapshot if a later step fails.
	return updated, nil
}

func DeleteCustomerGroup(ctx context.Context, groups CustomerGroupMutations, id string) error {
	if _, err := groups.GetCustomerGroup(ctx, id); err != nil {
		return err
	}
	return groups.DeleteCustomerGroup(ctx, id)
}

func LinkGroupCustomers(ctx context.Context, links CustomerGroupLinks, groupID string, add, remove []string, createdBy *string) error {
	added := make([]string, 0, len(add))
	for _, customerID := range add {
		if _, err := links.AddCustomerToGroup(ctx, customerID, groupID, createdBy); err != nil {
			for _, rollbackID := range added {
				if rollbackErr := links.RemoveCustomerFromGroup(ctx, rollbackID, groupID); rollbackErr != nil {
					err = errors.Join(err, rollbackErr)
				}
			}
			return err
		}
		added = append(added, customerID)
	}
	removed := make([]string, 0, len(remove))
	for _, customerID := range remove {
		if err := links.RemoveCustomerFromGroup(ctx, customerID, groupID); err != nil {
			for _, rollbackID := range removed {
				if _, rollbackErr := links.AddCustomerToGroup(ctx, rollbackID, groupID, createdBy); rollbackErr != nil {
					err = errors.Join(err, rollbackErr)
				}
			}
			for _, rollbackID := range added {
				if rollbackErr := links.RemoveCustomerFromGroup(ctx, rollbackID, groupID); rollbackErr != nil {
					err = errors.Join(err, rollbackErr)
				}
			}
			return err
		}
		removed = append(removed, customerID)
	}
	return nil
}

// CreateCustomers follows Medusa's create-customers step and event order.
func CreateCustomers(ctx context.Context, customers CustomerMutations, events EventPublisher, in models.Input) (models.Customer, error) {
	created, err := customers.Create(ctx, in)
	if err != nil {
		return models.Customer{}, err
	}
	if events != nil {
		if err := events.Publish(ctx, CustomerCreatedEvent, []map[string]any{{"id": created.ID}}); err != nil {
			if rollbackErr := customers.Delete(ctx, created.ID); rollbackErr != nil {
				return models.Customer{}, errors.Join(err, rollbackErr)
			}
			return models.Customer{}, err
		}
	}
	return created, nil
}

// UpdateCustomers retains the previous customer values for event-failure compensation.
func UpdateCustomers(ctx context.Context, customers CustomerMutations, events EventPublisher, id string, in models.Input) (models.Customer, error) {
	previous, err := customers.Get(ctx, id)
	if err != nil {
		return models.Customer{}, err
	}
	updated, err := customers.Update(ctx, id, in)
	if err != nil {
		return models.Customer{}, err
	}
	if events != nil {
		if err := events.Publish(ctx, CustomerUpdatedEvent, []map[string]any{{"id": updated.ID}}); err != nil {
			_, rollbackErr := customers.Update(ctx, id, models.Input{CompanyName: previous.CompanyName, FirstName: previous.FirstName, LastName: previous.LastName, Email: previous.Email, Phone: previous.Phone, HasAccount: previous.HasAccount, Metadata: previous.Metadata})
			if rollbackErr != nil {
				return models.Customer{}, errors.Join(err, rollbackErr)
			}
			return models.Customer{}, err
		}
	}
	return updated, nil
}

// LinkCustomerGroups applies additions before removals and compensates completed links on failure.
func LinkCustomerGroups(ctx context.Context, links CustomerGroupLinks, customerID string, add, remove []string, createdBy *string) error {
	added := make([]string, 0, len(add))
	for _, groupID := range add {
		if _, err := links.AddCustomerToGroup(ctx, customerID, groupID, createdBy); err != nil {
			for _, rollbackID := range added {
				if rollbackErr := links.RemoveCustomerFromGroup(ctx, customerID, rollbackID); rollbackErr != nil {
					err = errors.Join(err, rollbackErr)
				}
			}
			return err
		}
		added = append(added, groupID)
	}
	removed := make([]string, 0, len(remove))
	for _, groupID := range remove {
		if err := links.RemoveCustomerFromGroup(ctx, customerID, groupID); err != nil {
			for _, rollbackID := range removed {
				if _, rollbackErr := links.AddCustomerToGroup(ctx, customerID, rollbackID, createdBy); rollbackErr != nil {
					err = errors.Join(err, rollbackErr)
				}
			}
			for _, rollbackID := range added {
				if rollbackErr := links.RemoveCustomerFromGroup(ctx, customerID, rollbackID); rollbackErr != nil {
					err = errors.Join(err, rollbackErr)
				}
			}
			return err
		}
		removed = append(removed, groupID)
	}
	return nil
}

// RemoveCustomerAccount mirrors Medusa's soft-delete, event, and auth-identity sequence.
func RemoveCustomerAccount(ctx context.Context, customers CustomerAccountRemoval, auth CustomerAuthIdentity, events EventPublisher, customerID string) error {
	previous, err := customers.Get(ctx, customerID)
	if err != nil {
		return err
	}
	if err = customers.Delete(ctx, customerID); err != nil {
		return err
	}
	if events != nil {
		if err = events.Publish(ctx, CustomerDeletedEvent, []map[string]any{{"id": customerID}}); err != nil {
			return errors.Join(err, restoreCustomer(ctx, customers, previous))
		}
	}
	if !previous.HasAccount {
		return nil
	}
	identity, err := auth.FindIdentityByActor(ctx, "customer", customerID)
	if err != nil {
		return errors.Join(err, restoreCustomer(ctx, customers, previous))
	}
	metadata := map[string]any{}
	if err = json.Unmarshal(identity.AppMetadata, &metadata); err != nil {
		return errors.Join(err, restoreCustomer(ctx, customers, previous))
	}
	shared := false
	for key, value := range metadata {
		if key != "customer_id" && value != nil {
			shared = true
			break
		}
	}
	if shared {
		err = auth.UnsetActorMetadata(ctx, identity.ID, "customer_id")
	} else {
		err = auth.DeleteIdentity(ctx, identity.ID)
	}
	if err != nil {
		return errors.Join(err, restoreCustomer(ctx, customers, previous))
	}
	return nil
}

func restoreCustomer(ctx context.Context, customers CustomerAccountRemoval, previous models.Customer) error {
	if err := customers.RestoreCustomer(ctx, previous.ID); err != nil {
		return err
	}
	var rollbackErr error
	for _, group := range previous.Groups {
		if _, err := customers.AddCustomerToGroup(ctx, previous.ID, group.ID, group.CreatedBy); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
	}
	return rollbackErr
}
