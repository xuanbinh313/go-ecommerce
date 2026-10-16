package workflows

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/rbac/domain/models"
)

const CreateRbacPoliciesWorkflowID = "create-rbac-policies"
const CreateRbacPoliciesStepID = "create-rbac-policies"

type PolicyCreator interface {
	CreatePolicies(context.Context, []models.PolicyInput) ([]models.Policy, error)
	DeletePolicies(context.Context, []string) error
}

func CreateRbacPolicies(ctx context.Context, service PolicyCreator, policies []models.PolicyInput) ([]models.Policy, error) {
	return service.CreatePolicies(ctx, policies)
}
