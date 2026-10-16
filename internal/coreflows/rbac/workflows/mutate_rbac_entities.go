package workflows

import (
	"context"

	"github.com/example/medusa-go-clone/internal/modules/rbac/domain/models"
)

const (
	UpdateRbacRolesWorkflowID        = "update-rbac-roles"
	UpdateRbacRolesStepID            = "update-rbac-roles"
	UpdateRbacPoliciesWorkflowID     = "update-rbac-policies"
	UpdateRbacPoliciesStepID         = "update-rbac-policies"
	DeleteRbacRolesWorkflowID        = "delete-rbac-roles"
	DeleteRbacRolesStepID            = "delete-rbac-roles"
	DeleteRbacPoliciesWorkflowID     = "delete-rbac-policies"
	DeleteRbacPoliciesStepID         = "delete-rbac-policies"
	DeleteRbacRolePoliciesWorkflowID = "delete-rbac-role-policies"
	DeleteRbacRolePoliciesStepID     = "delete-rbac-role-policies"
)

type RbacEntityMutator interface {
	GetRole(context.Context, string) (models.Role, error)
	GetPolicy(context.Context, string) (models.Policy, error)
	UpdateRole(context.Context, string, models.RoleUpdate) (models.Role, error)
	UpdatePolicy(context.Context, string, models.PolicyUpdate) (models.Policy, error)
	SoftDeleteRoles(context.Context, []string) error
	SoftDeletePolicies(context.Context, []string) error
	DeleteRolePolicies(context.Context, []string) error
}

func UpdateRbacRole(ctx context.Context, service RbacEntityMutator, id string, update models.RoleUpdate) (models.Role, error) {
	if _, err := service.GetRole(ctx, id); err != nil {
		return models.Role{}, err
	}
	return service.UpdateRole(ctx, id, update)
}

func UpdateRbacPolicy(ctx context.Context, service RbacEntityMutator, id string, update models.PolicyUpdate) (models.Policy, error) {
	if _, err := service.GetPolicy(ctx, id); err != nil {
		return models.Policy{}, err
	}
	return service.UpdatePolicy(ctx, id, update)
}

func DeleteRbacRoles(ctx context.Context, service RbacEntityMutator, ids []string) error {
	return service.SoftDeleteRoles(ctx, ids)
}

func DeleteRbacPolicies(ctx context.Context, service RbacEntityMutator, ids []string) error {
	return service.SoftDeletePolicies(ctx, ids)
}

func DeleteRbacRolePolicies(ctx context.Context, service RbacEntityMutator, ids []string) error {
	return service.DeleteRolePolicies(ctx, ids)
}
