package workflows

import (
	"context"
	"errors"
	"fmt"

	linkdomain "github.com/example/medusa-go-clone/internal/modules/links/domain"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	rbacdomain "github.com/example/medusa-go-clone/internal/modules/rbac/domain"
	rbacmodels "github.com/example/medusa-go-clone/internal/modules/rbac/domain/models"
	userdomain "github.com/example/medusa-go-clone/internal/modules/user/domain"
	"github.com/example/medusa-go-clone/internal/modules/user/domain/models"
)

const CreateInvitesWorkflowID = "create-invite-step"
const CreateInviteStepID = "create-invite-step"
const ValidateInviteRolesStepID = "validate-roles-exist-step"
const DeleteInvitesWorkflowID = "delete-invites-workflow"
const DeleteInvitesStepID = "delete-invites-step"
const RefreshInviteTokensWorkflowID = "refresh-invite-tokens-workflow"
const RefreshInviteTokensStepID = "refresh-invite-tokens-step"
const InviteCreatedEvent = "invite.created"
const InviteDeletedEvent = "invite.deleted"
const InviteResentEvent = "invite.resent"
const AcceptInviteWorkflowID = "accept-invite-workflow"
const ValidateInviteTokenStepID = "validate-invite-token-step"
const GetInviteRolesStepID = "get-invite-roles-step"
const InviteAcceptedEvent = "invite.accepted"

type InviteCreator interface {
	CreateInvite(context.Context, models.InviteInput) (models.Invite, error)
	GetInvite(context.Context, string) (models.Invite, error)
	DeleteInvites(context.Context, []string) error
	SoftDeleteInvites(context.Context, []string) error
	RestoreInvites(context.Context, []string) error
	RefreshInviteTokens(context.Context, []string) ([]models.Invite, error)
}
type InviteRoleCatalog interface {
	GetRole(context.Context, string) (rbacmodels.Role, error)
}
type InviteEventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}
type CreateInviteInput struct {
	models.InviteInput
	Roles []string `json:"roles,omitempty"`
}

type InviteTokenValidator interface {
	ValidateInviteToken(context.Context, string) (models.Invite, error)
	SoftDeleteInvites(context.Context, []string) error
	RestoreInvites(context.Context, []string) error
}
type InviteAuthMetadata interface {
	SetActorMetadata(context.Context, string, string, string) (string, error)
	UnsetActorMetadata(context.Context, string, string) error
}
type AcceptInviteInput struct {
	InviteToken    string
	AuthIdentityID string
	Email          *string
	FirstName      *string
	LastName       *string
}

func AcceptInvite(ctx context.Context, inviteService InviteTokenValidator, users userdomain.Repository, catalog rbacdomain.Repository, auth InviteAuthMetadata, links linkdomain.Repository, events InviteEventPublisher, input AcceptInviteInput) (models.User, error) {
	invite, err := inviteService.ValidateInviteToken(ctx, input.InviteToken)
	if err != nil {
		return models.User{}, err
	}
	def, ok := inviteRoleDefinition()
	if !ok {
		return models.User{}, errors.New("InviteRBACRole link definition is missing")
	}
	roleLinks, err := links.List(ctx, def, map[string]any{"invite_id": invite.ID})
	if err != nil {
		return models.User{}, err
	}
	roles := make([]string, 0, len(roleLinks))
	for _, link := range roleLinks {
		if id, ok := link.Fields["rbac_role_id"].(string); ok {
			roles = append(roles, id)
		}
	}
	email := invite.Email
	if input.Email != nil {
		email = *input.Email
	}
	created, err := CreateUsers(ctx, users, catalog, links, events, []CreateUserInput{{UserInput: models.UserInput{Email: &email, FirstName: input.FirstName, LastName: input.LastName}, Roles: roles}})
	if err != nil {
		return models.User{}, err
	}
	if len(created) == 0 {
		return models.User{}, errors.New("invite user creation returned no user")
	}
	user := created[0]
	metadataSet := false
	cleanup := func(cause error) error {
		if metadataSet {
			if e := auth.UnsetActorMetadata(ctx, input.AuthIdentityID, "user_id"); e != nil {
				cause = errors.Join(cause, fmt.Errorf("restore auth metadata: %w", e))
			}
		}
		userDef, exists := userRoleDefinition()
		if exists {
			if _, e := links.Dismiss(ctx, userDef, []map[string]any{{"user_id": user.ID}}); e != nil {
				cause = errors.Join(cause, fmt.Errorf("remove created user-role links: %w", e))
			}
		}
		if e := users.DeleteUsers(ctx, []string{user.ID}); e != nil {
			cause = errors.Join(cause, fmt.Errorf("delete created user: %w", e))
		}
		return cause
	}
	if _, err = auth.SetActorMetadata(ctx, input.AuthIdentityID, "user", user.ID); err != nil {
		return models.User{}, cleanup(err)
	}
	metadataSet = true
	if err = inviteService.SoftDeleteInvites(ctx, []string{invite.ID}); err != nil {
		return models.User{}, cleanup(err)
	}
	if events == nil {
		return models.User{}, cleanup(errors.New("invite event publisher is required"))
	}
	if err = events.Publish(ctx, InviteAcceptedEvent, []map[string]any{{"id": invite.ID}}); err != nil {
		if e := inviteService.RestoreInvites(ctx, []string{invite.ID}); e != nil {
			err = errors.Join(err, e)
		}
		return models.User{}, cleanup(err)
	}
	return user, nil
}

func CreateInvites(ctx context.Context, users InviteCreator, roles InviteRoleCatalog, links linkdomain.Repository, events InviteEventPublisher, inputs []CreateInviteInput) ([]models.Invite, error) {
	seen := map[string]bool{}
	for _, in := range inputs {
		for _, roleID := range in.Roles {
			if seen[roleID] {
				continue
			}
			seen[roleID] = true
			if _, err := roles.GetRole(ctx, roleID); err != nil {
				return nil, err
			}
		}
	}
	created := make([]models.Invite, 0, len(inputs))
	for _, in := range inputs {
		row, err := users.CreateInvite(ctx, in.InviteInput)
		if err != nil {
			return nil, rollbackInvites(ctx, users, created, err)
		}
		created = append(created, row)
	}
	definition, ok := inviteRoleDefinition()
	if !ok && hasInviteRoles(inputs) {
		return nil, rollbackInvites(ctx, users, created, errors.New("InviteRBACRole link definition is missing"))
	}
	roleLinks := []map[string]any{}
	for i, in := range inputs {
		for _, roleID := range in.Roles {
			roleLinks = append(roleLinks, map[string]any{"invite_id": created[i].ID, "rbac_role_id": roleID})
		}
	}
	if len(roleLinks) > 0 {
		if _, err := links.Create(ctx, definition, roleLinks); err != nil {
			return nil, rollbackInviteCreation(ctx, users, links, definition, roleLinks, created, err)
		}
	}
	if events == nil {
		return nil, rollbackInviteCreation(ctx, users, links, definition, roleLinks, created, errors.New("invite event publisher is required"))
	}
	eventsData := make([]map[string]any, len(created))
	for i, row := range created {
		eventsData[i] = map[string]any{"id": row.ID}
	}
	if err := events.Publish(ctx, InviteCreatedEvent, eventsData); err != nil {
		return nil, rollbackInviteCreation(ctx, users, links, definition, roleLinks, created, fmt.Errorf("emit %s: %w", InviteCreatedEvent, err))
	}
	return created, nil
}

func DeleteInvite(ctx context.Context, users InviteCreator, events InviteEventPublisher, id string) error {
	if _, err := users.GetInvite(ctx, id); err != nil {
		return err
	}
	if err := users.SoftDeleteInvites(ctx, []string{id}); err != nil {
		return err
	}
	if events == nil {
		if e := users.RestoreInvites(ctx, []string{id}); e != nil {
			return errors.Join(errors.New("invite event publisher is required"), e)
		}
		return errors.New("invite event publisher is required")
	}
	if err := events.Publish(ctx, InviteDeletedEvent, []map[string]any{{"id": id}}); err != nil {
		if e := users.RestoreInvites(ctx, []string{id}); e != nil {
			return errors.Join(err, e)
		}
		return err
	}
	return nil
}

func RefreshInviteTokens(ctx context.Context, users InviteCreator, events InviteEventPublisher, id string) (models.Invite, error) {
	rows, err := users.RefreshInviteTokens(ctx, []string{id})
	if err != nil {
		return models.Invite{}, err
	}
	if events == nil {
		return models.Invite{}, errors.New("invite event publisher is required")
	}
	if err = events.Publish(ctx, InviteResentEvent, []map[string]any{{"id": id}}); err != nil {
		return models.Invite{}, err
	}
	return rows[0], nil
}

func rollbackInvites(ctx context.Context, users InviteCreator, created []models.Invite, cause error) error {
	if len(created) == 0 {
		return cause
	}
	ids := make([]string, len(created))
	for i, row := range created {
		ids[i] = row.ID
	}
	if err := users.DeleteInvites(ctx, ids); err != nil {
		return errors.Join(cause, fmt.Errorf("compensate %s: %w", CreateInviteStepID, err))
	}
	return cause
}
func rollbackInviteCreation(ctx context.Context, users InviteCreator, links linkdomain.Repository, definition linkmodels.Definition, roleLinks []map[string]any, created []models.Invite, cause error) error {
	if len(roleLinks) > 0 {
		if _, err := links.Dismiss(ctx, definition, roleLinks); err != nil {
			cause = errors.Join(cause, fmt.Errorf("dismiss invite-role links: %w", err))
		}
	}
	return rollbackInvites(ctx, users, created, cause)
}
func inviteRoleDefinition() (linkmodels.Definition, bool) {
	for _, d := range linkmodels.Definitions {
		if d.Name == "InviteRBACRole" {
			return d, true
		}
	}
	return linkmodels.Definition{}, false
}
func hasInviteRoles(inputs []CreateInviteInput) bool {
	for _, in := range inputs {
		if len(in.Roles) > 0 {
			return true
		}
	}
	return false
}
