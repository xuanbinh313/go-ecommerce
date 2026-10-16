package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/rbac/domain/models"
)

type Repository interface {
	ListRoles(context.Context, int, int) ([]models.Role, int, error)
	ListPolicies(context.Context, int, int) ([]models.Policy, int, error)
	GetRole(context.Context, string) (models.Role, error)
	GetPolicy(context.Context, string) (models.Policy, error)
	CreateRoles(context.Context, []models.RoleInput) ([]models.Role, error)
	CreatePolicies(context.Context, []models.PolicyInput) ([]models.Policy, error)
	UpdateRole(context.Context, string, models.RoleUpdate) (models.Role, error)
	UpdatePolicy(context.Context, string, models.PolicyUpdate) (models.Policy, error)
	CreateRolePolicies(context.Context, []models.RolePolicyInput) ([]models.RolePolicy, error)
	ListRolePolicies(context.Context, string, int, int) ([]models.RolePolicy, int, error)
	CreateRoleParents(context.Context, []models.RoleParentInput) ([]models.RoleParent, error)
	ListPoliciesForRoles(context.Context, []string) (map[string][]models.Policy, error)
	PolicyIDsExist(context.Context, []string) (bool, error)
	RoleIDsExist(context.Context, []string) (bool, error)
	ExistingRoleIDs(context.Context, []string) ([]string, error)
	WouldCreateCycle(context.Context, string, string) (bool, error)
	DeleteRoles(context.Context, []string) error
	DeletePolicies(context.Context, []string) error
	SoftDeleteRoles(context.Context, []string) error
	RestoreRoles(context.Context, []string) error
	SoftDeletePolicies(context.Context, []string) error
	RestorePolicies(context.Context, []string) error
	DeleteRolePolicies(context.Context, []string) error
	DeleteRoleParents(context.Context, []string) error
}
