package domain

import (
	"context"
	"encoding/json"

	"github.com/example/medusa-go-clone/internal/modules/customer/domain/models"
)

// Repository is the customer persistence port implemented by infrastructure.
type Repository interface {
	AccountStatesByEmail(context.Context, string) ([]bool, error)
	FindCustomersByEmail(context.Context, string) ([]models.Customer, error)
	Create(context.Context, models.Input) (models.Customer, error)
	Get(context.Context, string) (models.Customer, error)
	List(context.Context, int, int) ([]models.Customer, error)
	Count(context.Context) (int64, error)
	Update(context.Context, string, models.Input) (models.Customer, error)
	Delete(context.Context, string) error
	RestoreCustomer(context.Context, string) error
	CreateAddress(context.Context, string, models.AddressInput) (models.Address, error)
	ListAddresses(context.Context, string, int, int) ([]models.Address, int64, error)
	GetAddress(context.Context, string, string) (models.Address, error)
	UpdateAddress(context.Context, string, string, models.AddressUpdate) (models.Customer, error)
	DeleteAddress(context.Context, string, string) error
	CreateCustomerGroups(context.Context, string, json.RawMessage, *string) (models.CustomerGroup, error)
	ListCustomerGroups(context.Context) ([]models.CustomerGroup, error)
	GetCustomerGroup(context.Context, string) (models.CustomerGroup, error)
	UpdateCustomerGroup(context.Context, string, models.CustomerGroupUpdate) (models.CustomerGroup, error)
	DeleteCustomerGroup(context.Context, string) error
	RestoreCustomerGroup(context.Context, string) error
	AddCustomerToGroup(context.Context, string, string, *string) (string, error)
	RemoveCustomerFromGroup(context.Context, string, string) error
}

// AccountService is the workflow-facing port for customer account operations.
type AccountService interface {
	ValidateAccountCreation(context.Context, *string, string) error
	Create(context.Context, models.Input) (models.Customer, error)
	Delete(context.Context, string) error
}

// OrderCustomerResolver is the narrow customer domain contract used by order workflows.
type OrderCustomerResolver interface {
	FindCustomersByEmail(context.Context, string) ([]models.Customer, error)
	Create(context.Context, models.Input) (models.Customer, error)
	Delete(context.Context, string) error
}

// CartCustomerResolver is the customer lookup port used by cart ownership workflows.
type CartCustomerResolver interface {
	Get(context.Context, string) (models.Customer, error)
}
