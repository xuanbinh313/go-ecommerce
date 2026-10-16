package application

import (
	"context"
	"errors"
	"github.com/example/medusa-go-clone/internal/modules/rbac/domain"
	"github.com/example/medusa-go-clone/internal/modules/rbac/domain/models"
	"strings"
)

type Service struct{ repository domain.Repository }

func NewService(repository domain.Repository) *Service { return &Service{repository: repository} }
func (s *Service) ListRoles(ctx context.Context, limit, offset int) ([]models.Role, int, error) {
	return s.repository.ListRoles(ctx, limit, offset)
}
func (s *Service) ListPolicies(ctx context.Context, limit, offset int) ([]models.Policy, int, error) {
	return s.repository.ListPolicies(ctx, limit, offset)
}
func (s *Service) GetRole(ctx context.Context, id string) (models.Role, error) {
	return s.repository.GetRole(ctx, id)
}
func (s *Service) GetPolicy(ctx context.Context, id string) (models.Policy, error) {
	return s.repository.GetPolicy(ctx, id)
}
func (s *Service) UpdateRole(ctx context.Context, id string, in models.RoleUpdate) (models.Role, error) {
	return s.repository.UpdateRole(ctx, id, in)
}
func (s *Service) UpdatePolicy(ctx context.Context, id string, in models.PolicyUpdate) (models.Policy, error) {
	if in.Resource != nil {
		value := strings.ToLower(*in.Resource)
		in.Resource = &value
	}
	if in.Operation != nil {
		value := strings.ToLower(*in.Operation)
		in.Operation = &value
	}
	return s.repository.UpdatePolicy(ctx, id, in)
}
func (s *Service) CreateRoles(ctx context.Context, in []models.RoleInput) ([]models.Role, error) {
	return s.repository.CreateRoles(ctx, in)
}
func (s *Service) CreatePolicies(ctx context.Context, in []models.PolicyInput) ([]models.Policy, error) {
	for i := range in {
		in[i].Resource = strings.ToLower(in[i].Resource)
		in[i].Operation = strings.ToLower(in[i].Operation)
		if in[i].Key == "" {
			in[i].Key = in[i].Resource + ":" + in[i].Operation
		}
	}
	return s.repository.CreatePolicies(ctx, in)
}
func (s *Service) CreateRolePolicies(ctx context.Context, in []models.RolePolicyInput) ([]models.RolePolicy, error) {
	if len(in) == 0 {
		return []models.RolePolicy{}, nil
	}
	ids := make([]string, 0, len(in))
	for _, v := range in {
		ids = append(ids, v.PolicyID)
	}
	ok, err := s.repository.PolicyIDsExist(ctx, ids)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("one or more policies do not exist")
	}
	return s.repository.CreateRolePolicies(ctx, in)
}
func (s *Service) ListRolePolicies(ctx context.Context, roleID string, limit, offset int) ([]models.RolePolicy, int, error) {
	return s.repository.ListRolePolicies(ctx, roleID, limit, offset)
}
func (s *Service) CreateRoleParents(ctx context.Context, in []models.RoleParentInput) ([]models.RoleParent, error) {
	for _, v := range in {
		if v.RoleID == v.ParentID {
			return nil, errors.New("a role cannot be its own parent")
		}
		cycle, err := s.repository.WouldCreateCycle(ctx, v.RoleID, v.ParentID)
		if err != nil {
			return nil, err
		}
		if cycle {
			return nil, errors.New("role parent relationship would create a circular dependency")
		}
	}
	return s.repository.CreateRoleParents(ctx, in)
}
func (s *Service) ListPoliciesForRoles(ctx context.Context, ids []string) (map[string][]models.Policy, error) {
	return s.repository.ListPoliciesForRoles(ctx, ids)
}
func (s *Service) ExistingRoleIDs(ctx context.Context, ids []string) ([]string, error) {
	return s.repository.ExistingRoleIDs(ctx, ids)
}
func (s *Service) PolicyIDsExist(ctx context.Context, ids []string) (bool, error) {
	return s.repository.PolicyIDsExist(ctx, ids)
}
func (s *Service) RoleIDsExist(ctx context.Context, ids []string) (bool, error) {
	return s.repository.RoleIDsExist(ctx, ids)
}
func (s *Service) WouldCreateCycle(ctx context.Context, roleID, parentID string) (bool, error) {
	return s.repository.WouldCreateCycle(ctx, roleID, parentID)
}
func (s *Service) DeleteRoles(ctx context.Context, ids []string) error {
	return s.repository.DeleteRoles(ctx, ids)
}
func (s *Service) DeletePolicies(ctx context.Context, ids []string) error {
	return s.repository.DeletePolicies(ctx, ids)
}
func (s *Service) SoftDeleteRoles(ctx context.Context, ids []string) error {
	return s.repository.SoftDeleteRoles(ctx, ids)
}
func (s *Service) RestoreRoles(ctx context.Context, ids []string) error {
	return s.repository.RestoreRoles(ctx, ids)
}
func (s *Service) SoftDeletePolicies(ctx context.Context, ids []string) error {
	return s.repository.SoftDeletePolicies(ctx, ids)
}
func (s *Service) RestorePolicies(ctx context.Context, ids []string) error {
	return s.repository.RestorePolicies(ctx, ids)
}
func (s *Service) DeleteRolePolicies(ctx context.Context, ids []string) error {
	return s.repository.DeleteRolePolicies(ctx, ids)
}
func (s *Service) DeleteRoleParents(ctx context.Context, ids []string) error {
	return s.repository.DeleteRoleParents(ctx, ids)
}
