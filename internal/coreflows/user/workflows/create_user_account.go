package workflows

import (
	"context"
	"errors"

	authflows "github.com/example/medusa-go-clone/internal/coreflows/auth/workflows"
	authdomain "github.com/example/medusa-go-clone/internal/modules/auth/domain"
	linkdomain "github.com/example/medusa-go-clone/internal/modules/links/domain"
	rbacdomain "github.com/example/medusa-go-clone/internal/modules/rbac/domain"
	userdomain "github.com/example/medusa-go-clone/internal/modules/user/domain"
	usermodels "github.com/example/medusa-go-clone/internal/modules/user/domain/models"
)

const CreateUserAccountWorkflowID = "create-user-account"

type CreateUserAccountInput struct {
	AuthIdentityID string
	UserData       usermodels.UserInput
}

// CreateUserAccount follows Medusa's nested create-users workflow, then sets
// the auth identity app metadata using the shared source step.
func CreateUserAccount(ctx context.Context, users userdomain.Repository, rbac rbacdomain.Repository, links linkdomain.Repository, auth authdomain.IdentityMetadata, events EventPublisher, input CreateUserAccountInput) (usermodels.User, error) {
	if input.AuthIdentityID == "" {
		return usermodels.User{}, errors.New("auth identity ID is required")
	}
	created, err := CreateUsers(ctx, users, rbac, links, events, []CreateUserInput{{UserInput: input.UserData}})
	if err != nil {
		return usermodels.User{}, err
	}
	user := created[0]
	if err = authflows.SetAuthAppMetadataStep(ctx, auth, input.AuthIdentityID, "user", user.ID); err != nil {
		if rollbackErr := users.DeleteUsers(ctx, []string{user.ID}); rollbackErr != nil {
			err = errors.Join(err, rollbackErr)
		}
		return usermodels.User{}, err
	}
	return user, nil
}
