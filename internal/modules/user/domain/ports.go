package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/user/domain/models"
	"time"
)

type Repository interface {
	CreateUser(context.Context, models.UserInput) (models.User, error)
	GetUser(context.Context, string) (models.User, error)
	ListUsers(context.Context, []string) ([]models.User, error)
	FindUsersByEmail(context.Context, []string) ([]models.User, error)
	UpdateUser(context.Context, string, models.UserInput) (models.User, error)
	SoftDeleteUsers(context.Context, []string) error
	RestoreUsers(context.Context, []string) error
	DeleteUsers(context.Context, []string) error
	CreateInvite(context.Context, models.Invite) (models.Invite, error)
	GetInvite(context.Context, string) (models.Invite, error)
	ListInvites(context.Context, []string) ([]models.Invite, error)
	ListInvitesPage(context.Context, []string, []string, int, int) ([]models.Invite, int, error)
	UpdateInvite(context.Context, models.Invite) (models.Invite, error)
	SoftDeleteInvites(context.Context, []string) error
	RestoreInvites(context.Context, []string) error
	DeleteInvites(context.Context, []string) error
}

type InviteClaims struct {
	ID    string
	Email string
}
type InviteTokenManager interface {
	Issue(InviteClaims, time.Duration) (string, time.Time, error)
	Parse(string) (InviteClaims, error)
}
