package workflows

import (
	"context"
	"errors"
	"fmt"
	"strings"

	linkdomain "github.com/example/medusa-go-clone/internal/modules/links/domain"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	rbacdomain "github.com/example/medusa-go-clone/internal/modules/rbac/domain"
	userdomain "github.com/example/medusa-go-clone/internal/modules/user/domain"
	usermodels "github.com/example/medusa-go-clone/internal/modules/user/domain/models"
)

const CreateUsersWorkflowID = "create-users-workflow"
const ValidateRolesExistStepID = "validate-roles-exist-step"
const CreateUsersStepID = "create-users-step"
const CreateRemoteLinksStepID = "create-remote-links"
const EmitEventStepID = "emit-event-step"

const UserCreatedEvent = "user.created"

type CreateUserInput struct {
	usermodels.UserInput
	Roles []string `json:"roles,omitempty"`
}
type EventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}

// CreateUsers follows Medusa's step order: validate all referenced role IDs,
// create users, create user-RBAC links, and emit user.created.
func CreateUsers(ctx context.Context, users userdomain.Repository, rbac rbacdomain.Repository, links linkdomain.Repository, events EventPublisher, inputs []CreateUserInput) ([]usermodels.User, error) {
	roleIDs := []string{}
	seen := map[string]bool{}
	for _, input := range inputs {
		for _, roleID := range input.Roles {
			if !seen[roleID] {
				seen[roleID] = true
				roleIDs = append(roleIDs, roleID)
			}
		}
	}
	if len(roleIDs) > 0 {
		existing, err := rbac.ExistingRoleIDs(ctx, roleIDs)
		if err != nil {
			return nil, err
		}
		existingSet := map[string]bool{}
		for _, id := range existing {
			existingSet[id] = true
		}
		missing := []string{}
		for _, id := range roleIDs {
			if !existingSet[id] {
				missing = append(missing, id)
			}
		}
		if len(missing) > 0 {
			return nil, fmt.Errorf("The following role IDs do not exist: %s", strings.Join(missing, ", "))
		}
	}

	created := make([]usermodels.User, 0, len(inputs))
	for _, input := range inputs {
		user, err := users.CreateUser(ctx, input.UserInput)
		if err != nil {
			ids := make([]string, len(created))
			for i, createdUser := range created {
				ids[i] = createdUser.ID
			}
			return nil, compensateUsers(ctx, users, ids, err)
		}
		created = append(created, user)
	}
	createdIDs := make([]string, len(created))
	for i, user := range created {
		createdIDs[i] = user.ID
	}
	definition, ok := userRoleDefinition()
	if !ok {
		return nil, compensateUsers(ctx, users, createdIDs, errors.New("UserRBACRole link definition is missing"))
	}
	roleLinks := []map[string]any{}
	for i, input := range inputs {
		for _, roleID := range input.Roles {
			roleLinks = append(roleLinks, map[string]any{"user_id": created[i].ID, "rbac_role_id": roleID})
		}
	}
	if len(roleLinks) > 0 {
		if _, err := links.Create(ctx, definition, roleLinks); err != nil {
			return nil, compensateUsers(ctx, users, createdIDs, err)
		}
	}
	if events == nil {
		return nil, compensateUserCreation(ctx, users, links, definition, roleLinks, createdIDs, errors.New("user event publisher is required"))
	}
	eventData := make([]map[string]any, len(created))
	for i, user := range created {
		eventData[i] = map[string]any{"id": user.ID}
	}
	if err := events.Publish(ctx, UserCreatedEvent, eventData); err != nil {
		return nil, compensateUserCreation(ctx, users, links, definition, roleLinks, createdIDs, fmt.Errorf("emit %s: %w", UserCreatedEvent, err))
	}
	return created, nil
}

func userRoleDefinition() (linkmodels.Definition, bool) {
	for _, definition := range linkmodels.Definitions {
		if definition.Name == "UserRBACRole" {
			return definition, true
		}
	}
	return linkmodels.Definition{}, false
}
func compensateUsers(ctx context.Context, users userdomain.Repository, ids []string, cause error) error {
	if err := users.DeleteUsers(ctx, ids); err != nil {
		return errors.Join(cause, fmt.Errorf("delete created users: %w", err))
	}
	return cause
}
func compensateUserCreation(ctx context.Context, users userdomain.Repository, links linkdomain.Repository, definition linkmodels.Definition, roleLinks []map[string]any, ids []string, cause error) error {
	if len(roleLinks) > 0 {
		if _, err := links.Dismiss(ctx, definition, roleLinks); err != nil {
			cause = errors.Join(cause, fmt.Errorf("dismiss created user-role links: %w", err))
		}
	}
	return compensateUsers(ctx, users, ids, cause)
}
