package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/customer/domain"
	"github.com/example/medusa-go-clone/internal/modules/customer/domain/models"
)

type Service struct{ repository domain.Repository }

type CustomerService interface {
	Get(context.Context, string) (models.Customer, error)
	List(context.Context, int, int) ([]models.Customer, error)
	Count(context.Context) (int64, error)
	Update(context.Context, string, models.Input) (models.Customer, error)
	CreateAddress(context.Context, string, models.AddressInput) (models.Address, error)
	ListAddresses(context.Context, string, int, int) ([]models.Address, int64, error)
	GetAddress(context.Context, string, string) (models.Address, error)
	UpdateAddress(context.Context, string, string, models.AddressUpdate) (models.Customer, error)
	DeleteAddress(context.Context, string, string) error
}

func NewService(repository domain.Repository) *Service { return &Service{repository: repository} }

func (s *Service) ValidateAccountCreation(ctx context.Context, email *string, authIdentityID string) error {
	if email == nil || strings.TrimSpace(*email) == "" {
		return errors.New("Email is required to create a customer")
	}
	states, err := s.repository.AccountStatesByEmail(ctx, strings.TrimSpace(*email))
	if err != nil {
		return err
	}
	for _, hasAccount := range states {
		if hasAccount && authIdentityID != "" {
			return errors.New("Customer with this email already has an account")
		}
		if !hasAccount && authIdentityID == "" {
			return errors.New("Guest customer with this email already exists")
		}
	}
	return nil
}

func (s *Service) Create(ctx context.Context, in models.Input) (models.Customer, error) {
	return s.repository.Create(ctx, in)
}
func (s *Service) FindCustomersByEmail(ctx context.Context, email string) ([]models.Customer, error) {
	return s.repository.FindCustomersByEmail(ctx, email)
}
func (s *Service) Get(ctx context.Context, id string) (models.Customer, error) {
	return s.repository.Get(ctx, id)
}
func (s *Service) List(ctx context.Context, limit, offset int) ([]models.Customer, error) {
	return s.repository.List(ctx, limit, offset)
}
func (s *Service) Count(ctx context.Context) (int64, error) { return s.repository.Count(ctx) }
func (s *Service) Update(ctx context.Context, id string, in models.Input) (models.Customer, error) {
	return s.repository.Update(ctx, id, in)
}
func (s *Service) Delete(ctx context.Context, id string) error { return s.repository.Delete(ctx, id) }
func (s *Service) RestoreCustomer(ctx context.Context, id string) error {
	return s.repository.RestoreCustomer(ctx, id)
}
func (s *Service) CreateAddress(ctx context.Context, id string, in models.AddressInput) (models.Address, error) {
	return s.repository.CreateAddress(ctx, id, in)
}
func (s *Service) ListAddresses(ctx context.Context, id string, limit, offset int) ([]models.Address, int64, error) {
	return s.repository.ListAddresses(ctx, id, limit, offset)
}
func (s *Service) GetAddress(ctx context.Context, id, addressID string) (models.Address, error) {
	return s.repository.GetAddress(ctx, id, addressID)
}
func (s *Service) UpdateAddress(ctx context.Context, id, addressID string, in models.AddressUpdate) (models.Customer, error) {
	return s.repository.UpdateAddress(ctx, id, addressID, in)
}
func (s *Service) DeleteAddress(ctx context.Context, id, addressID string) error {
	return s.repository.DeleteAddress(ctx, id, addressID)
}
func (s *Service) CreateCustomerGroups(ctx context.Context, name string, metadata json.RawMessage, createdBy *string) (models.CustomerGroup, error) {
	return s.repository.CreateCustomerGroups(ctx, name, metadata, createdBy)
}
func (s *Service) ListCustomerGroups(ctx context.Context) ([]models.CustomerGroup, error) {
	return s.repository.ListCustomerGroups(ctx)
}
func (s *Service) GetCustomerGroup(ctx context.Context, id string) (models.CustomerGroup, error) {
	return s.repository.GetCustomerGroup(ctx, id)
}
func (s *Service) UpdateCustomerGroup(ctx context.Context, id string, input models.CustomerGroupUpdate) (models.CustomerGroup, error) {
	return s.repository.UpdateCustomerGroup(ctx, id, input)
}
func (s *Service) DeleteCustomerGroup(ctx context.Context, id string) error {
	return s.repository.DeleteCustomerGroup(ctx, id)
}
func (s *Service) RestoreCustomerGroup(ctx context.Context, id string) error {
	return s.repository.RestoreCustomerGroup(ctx, id)
}
func (s *Service) AddCustomerToGroup(ctx context.Context, customerID, groupID string, createdBy *string) (string, error) {
	return s.repository.AddCustomerToGroup(ctx, customerID, groupID, createdBy)
}
func (s *Service) RemoveCustomerFromGroup(ctx context.Context, customerID, groupID string) error {
	return s.repository.RemoveCustomerFromGroup(ctx, customerID, groupID)
}

var _ CustomerService = (*Service)(nil)
var _ domain.AccountService = (*Service)(nil)
