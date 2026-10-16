package workflows

import (
	"context"
	"errors"
	"github.com/example/medusa-go-clone/internal/modules/rbac/domain/models"
)

const CreateRbacRolePoliciesWorkflowID = "create-rbac-role-policies"

type RolePolicyWorkflowService interface {
	CreateRolePolicies(context.Context, []models.RolePolicyInput) ([]models.RolePolicy, error)
	DeleteRolePolicies(context.Context, []string) error
}

func CreateRbacRolePolicies(ctx context.Context, service RolePolicyWorkflowService, validator PermissionValidator, actorID, actor string, inputs []models.RolePolicyInput) ([]models.RolePolicy, error) {
	if actorID != "" {
		ids := []string{}
		seen := map[string]bool{}
		for _, in := range inputs {
			if !seen[in.PolicyID] {
				seen[in.PolicyID] = true
				ids = append(ids, in.PolicyID)
			}
		}
		if len(ids) > 0 {
			if validator == nil {
				return nil, errors.New("RBAC actor permission validation is not configured")
			}
			if err := validator.ValidateUserPermissions(ctx, actorID, actor, ids); err != nil {
				return nil, err
			}
		}
	}
	return service.CreateRolePolicies(ctx, inputs)
}
