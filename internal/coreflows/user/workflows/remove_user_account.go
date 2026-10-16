package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	authdomain "github.com/example/medusa-go-clone/internal/modules/auth/domain"
	linkdomain "github.com/example/medusa-go-clone/internal/modules/links/domain"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	usermodels "github.com/example/medusa-go-clone/internal/modules/user/domain/models"
)

const RemoveUserAccountWorkflowID = "remove-user-account"
const DeleteUsersWorkflowID = "delete-user"
const DeleteUsersStepID = "delete-users-step"
const UserDeletedEvent = "user.deleted"

type UserAccountRemover interface {
	GetUser(context.Context, string) (usermodels.User, error)
	SoftDeleteUsers(context.Context, []string) error
	RestoreUsers(context.Context, []string) error
}
type AuthIdentityManager interface {
	FindIdentityByActor(context.Context, string, string) (authdomain.AuthIdentityRecord, error)
	UnsetActorMetadata(context.Context, string, string) error
	DeleteIdentity(context.Context, string) error
}
type UserAccountEventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}

// RemoveUserAccount mirrors Medusa's delete-user step, remote-link cleanup, event, then auth-identity handling.
func RemoveUserAccount(ctx context.Context, users UserAccountRemover, auth AuthIdentityManager, links linkdomain.Repository, events UserAccountEventPublisher, id string) error {
	if _, err := users.GetUser(ctx, id); err != nil {
		return err
	}
	if err := users.SoftDeleteUsers(ctx, []string{id}); err != nil {
		return err
	}
	type removedLink struct {
		definition linkmodels.Definition
		fields     map[string]any
	}
	removed := []removedLink{}
	rollback := func(cause error) error {
		if err := users.RestoreUsers(ctx, []string{id}); err != nil {
			cause = errors.Join(cause, fmt.Errorf("compensate %s: %w", DeleteUsersStepID, err))
		}
		for i := len(removed) - 1; i >= 0; i-- {
			row := removed[i]
			if _, err := links.Restore(ctx, row.definition, []map[string]any{row.fields}); err != nil {
				cause = errors.Join(cause, fmt.Errorf("restore %s link: %w", row.definition.Name, err))
			}
		}
		return cause
	}
	for _, definition := range linkmodels.Definitions {
		if !hasUserForeignKey(definition) {
			continue
		}
		filters := map[string]any{"user_id": id}
		rows, err := links.List(ctx, definition, filters)
		if err != nil {
			return rollback(err)
		}
		if len(rows) == 0 {
			continue
		}
		if _, err = links.Dismiss(ctx, definition, []map[string]any{filters}); err != nil {
			return rollback(err)
		}
		for _, row := range rows {
			removed = append(removed, removedLink{definition, row.Fields})
		}
	}
	if events == nil {
		return rollback(errors.New("user event publisher is required"))
	}
	if err := events.Publish(ctx, UserDeletedEvent, []map[string]any{{"id": id}}); err != nil {
		return rollback(err)
	}
	identity, err := auth.FindIdentityByActor(ctx, "user", id)
	if err != nil {
		// An unlinked User has no Auth identity to clean up.
		if !errors.Is(err, authdomain.ErrIdentityNotFound) {
			return rollback(err)
		}
		return nil
	}
	metadata := map[string]any{}
	if err = json.Unmarshal(identity.AppMetadata, &metadata); err != nil {
		return rollback(err)
	}
	otherActor := false
	for key, value := range metadata {
		if key == "user_id" {
			continue
		}
		if value != nil {
			otherActor = true
			break
		}
	}
	if otherActor {
		err = auth.UnsetActorMetadata(ctx, identity.ID, "user_id")
	} else {
		err = auth.DeleteIdentity(ctx, identity.ID)
	}
	if err != nil {
		return rollback(err)
	}
	return nil
}

func hasUserForeignKey(definition linkmodels.Definition) bool {
	for _, key := range definition.ForeignKeys {
		if key == "user_id" {
			return true
		}
	}
	return false
}
