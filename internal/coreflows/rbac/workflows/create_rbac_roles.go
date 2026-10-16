package workflows

import (
	"context"
	"errors"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/rbac/domain/models"
)

const CreateRbacRolesWorkflowID = "create-rbac-roles"
const CreateRbacRolesStepID = "create-rbac-roles"
const CreateRbacRoleParentsStepID = "create-rbac-role-parents"
const CreateRbacRolePoliciesStepID = "create-rbac-role-policies"
const ValidateUserPermissionsStepID = "validate-user-permissions"

type PermissionValidator interface {
	ValidateUserPermissions(context.Context, string, string, []string) error
}
type RoleWorkflowService interface {
	CreateRoles(context.Context, []models.RoleInput) ([]models.Role, error)
	CreateRoleParents(context.Context, []models.RoleParentInput) ([]models.RoleParent, error)
	CreateRolePolicies(context.Context, []models.RolePolicyInput) ([]models.RolePolicy, error)
	DeleteRoles(context.Context, []string) error
	DeleteRoleParents(context.Context, []string) error
	DeleteRolePolicies(context.Context, []string) error
}
type CreateRoleInput struct {
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Metadata    []byte   `json:"metadata,omitempty"`
	ParentIDs   []string `json:"parent_ids,omitempty"`
	PolicyIDs   []string `json:"policy_ids,omitempty"`
}

// CreateRbacRoles follows the source order: permission check, create roles,
// create parent relationships, then attach policies; failed later steps remove
// the records created by earlier steps.
func CreateRbacRoles(ctx context.Context, service RoleWorkflowService, validator PermissionValidator, actorID, actor string, inputs []CreateRoleInput) ([]models.Role, error) {
	policyIDs := []string{}
	seen := map[string]bool{}
	for _, in := range inputs {
		for _, id := range in.PolicyIDs {
			if !seen[id] {
				seen[id] = true
				policyIDs = append(policyIDs, id)
			}
		}
	}
	if actorID != "" && len(policyIDs) > 0 {
		if validator == nil {
			return nil, errors.New("RBAC actor permission validation is not configured")
		}
		if err := validator.ValidateUserPermissions(ctx, actorID, actor, policyIDs); err != nil {
			return nil, err
		}
	}
	roleInputs := make([]models.RoleInput, len(inputs))
	for i, in := range inputs {
		roleInputs[i] = models.RoleInput{Name: in.Name, Description: in.Description, Metadata: in.Metadata}
	}
	roles, err := service.CreateRoles(ctx, roleInputs)
	if err != nil {
		return nil, err
	}
	roleIDs := make([]string, len(roles))
	for i, r := range roles {
		roleIDs[i] = r.ID
	}
	compensate := func(cause error, parents, links []string) error {
		if len(links) > 0 {
			if e := service.DeleteRolePolicies(ctx, links); e != nil {
				cause = errors.Join(cause, fmt.Errorf("delete created role policies: %w", e))
			}
		}
		if len(parents) > 0 {
			if e := service.DeleteRoleParents(ctx, parents); e != nil {
				cause = errors.Join(cause, fmt.Errorf("delete created role parents: %w", e))
			}
		}
		if e := service.DeleteRoles(ctx, roleIDs); e != nil {
			cause = errors.Join(cause, fmt.Errorf("delete created roles: %w", e))
		}
		return cause
	}
	parentInputs := []models.RoleParentInput{}
	linkInputs := []models.RolePolicyInput{}
	for i, in := range inputs {
		for _, id := range in.ParentIDs {
			parentInputs = append(parentInputs, models.RoleParentInput{RoleID: roles[i].ID, ParentID: id})
		}
		for _, id := range in.PolicyIDs {
			linkInputs = append(linkInputs, models.RolePolicyInput{RoleID: roles[i].ID, PolicyID: id})
		}
	}
	parents, err := service.CreateRoleParents(ctx, parentInputs)
	if err != nil {
		return nil, compensate(err, nil, nil)
	}
	parentIDs := make([]string, len(parents))
	for i, p := range parents {
		parentIDs[i] = p.ID
	}
	links, err := service.CreateRolePolicies(ctx, linkInputs)
	if err != nil {
		return nil, compensate(err, parentIDs, nil)
	}
	_ = links
	return roles, nil
}
