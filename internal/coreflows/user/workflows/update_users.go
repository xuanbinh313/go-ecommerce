package workflows

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/user/domain/models"
)

const UpdateUsersWorkflowID = "update-users-workflow"
const UpdateUsersStepID = "update-users-step"
const UserUpdatedEvent = "user.updated"

type UserUpdateEventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}
type UserUpdater interface {
	GetUser(context.Context, string) (models.User, error)
	UpdateUser(context.Context, string, models.UserInput) (models.User, error)
}

func UpdateUser(ctx context.Context, users UserUpdater, events UserUpdateEventPublisher, id string, input models.UserInput) (models.User, error) {
	previous, err := users.GetUser(ctx, id)
	if err != nil {
		return models.User{}, err
	}
	updated, err := users.UpdateUser(ctx, id, input)
	if err != nil {
		return models.User{}, err
	}
	if events == nil {
		err = errors.New("user event publisher is required")
	} else {
		err = events.Publish(ctx, UserUpdatedEvent, []map[string]any{{"id": id}})
	}
	if err != nil {
		rollback := models.UserInput{FirstName: previous.FirstName, LastName: previous.LastName, Email: &previous.Email, AvatarURL: previous.AvatarURL, Metadata: previous.Metadata, FirstNameSet: true, LastNameSet: true, AvatarURLSet: true, MetadataSet: true}
		if _, e := users.UpdateUser(ctx, id, rollback); e != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", UpdateUsersStepID, e))
		}
		return models.User{}, err
	}
	return updated, nil
}
