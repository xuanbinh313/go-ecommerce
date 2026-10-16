package application

import (
	"context"
	"errors"
	"github.com/example/medusa-go-clone/internal/modules/user/domain"
	"github.com/example/medusa-go-clone/internal/modules/user/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"time"
)

const DefaultInviteDuration = 24 * time.Hour

type Service struct {
	repository     domain.Repository
	tokens         domain.InviteTokenManager
	inviteDuration time.Duration
}

func NewService(repository domain.Repository, tokens domain.InviteTokenManager, duration time.Duration) *Service {
	if duration <= 0 {
		duration = DefaultInviteDuration
	}
	return &Service{repository: repository, tokens: tokens, inviteDuration: duration}
}

func (s *Service) CreateUser(ctx context.Context, in models.UserInput) (models.User, error) {
	return s.repository.CreateUser(ctx, in)
}
func (s *Service) GetUser(ctx context.Context, id string) (models.User, error) {
	return s.repository.GetUser(ctx, id)
}
func (s *Service) ListUsers(ctx context.Context, ids []string) ([]models.User, error) {
	return s.repository.ListUsers(ctx, ids)
}
func (s *Service) FindUsersByEmail(ctx context.Context, emails []string) ([]models.User, error) {
	return s.repository.FindUsersByEmail(ctx, emails)
}
func (s *Service) UpdateUser(ctx context.Context, id string, in models.UserInput) (models.User, error) {
	return s.repository.UpdateUser(ctx, id, in)
}
func (s *Service) SoftDeleteUsers(ctx context.Context, ids []string) error {
	return s.repository.SoftDeleteUsers(ctx, ids)
}
func (s *Service) RestoreUsers(ctx context.Context, ids []string) error {
	return s.repository.RestoreUsers(ctx, ids)
}

func (s *Service) CreateInvite(ctx context.Context, in models.InviteInput) (models.Invite, error) {
	users, err := s.repository.FindUsersByEmail(ctx, []string{in.Email})
	if err != nil {
		return models.Invite{}, err
	}
	if len(users) > 0 {
		return models.Invite{}, errors.New("user account for this email already exists")
	}
	id := entityid.New("invite")
	if in.ID != nil {
		id = *in.ID
	}
	createdAt := time.Now()
	token, expires, err := s.tokens.Issue(domain.InviteClaims{ID: id, Email: in.Email}, s.inviteDuration)
	if err != nil {
		return models.Invite{}, err
	}
	return s.repository.CreateInvite(ctx, models.Invite{ID: id, Email: in.Email, Token: token, ExpiresAt: expires, Metadata: in.Metadata, CreatedAt: createdAt})
}

func (s *Service) ValidateInviteToken(ctx context.Context, token string) (models.Invite, error) {
	claims, err := s.tokens.Parse(token)
	if err != nil {
		return models.Invite{}, err
	}
	invite, err := s.repository.GetInvite(ctx, claims.ID)
	if err != nil {
		return models.Invite{}, err
	}
	if invite.Token != token {
		return models.Invite{}, errors.New("the invite token is invalid")
	}
	if invite.ExpiresAt.Before(time.Now()) {
		return models.Invite{}, errors.New("the invite has expired")
	}
	return invite, nil
}

func (s *Service) RefreshInviteTokens(ctx context.Context, ids []string) ([]models.Invite, error) {
	invites, err := s.repository.ListInvites(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(invites) != len(ids) {
		return nil, errors.New("one or more invites do not exist")
	}
	out := make([]models.Invite, 0, len(invites))
	for _, invite := range invites {
		invite.Token, invite.ExpiresAt, err = s.tokens.Issue(domain.InviteClaims{ID: invite.ID, Email: invite.Email}, s.inviteDuration)
		if err != nil {
			return nil, err
		}
		updated, err := s.repository.UpdateInvite(ctx, invite)
		if err != nil {
			return nil, err
		}
		out = append(out, updated)
	}
	return out, nil
}

func (s *Service) ListInvitesPage(ctx context.Context, ids, emails []string, limit, offset int) ([]models.Invite, int, error) {
	return s.repository.ListInvitesPage(ctx, ids, emails, limit, offset)
}
func (s *Service) GetInvite(ctx context.Context, id string) (models.Invite, error) {
	return s.repository.GetInvite(ctx, id)
}
func (s *Service) SoftDeleteInvites(ctx context.Context, ids []string) error {
	return s.repository.SoftDeleteInvites(ctx, ids)
}
func (s *Service) RestoreInvites(ctx context.Context, ids []string) error {
	return s.repository.RestoreInvites(ctx, ids)
}
func (s *Service) DeleteInvites(ctx context.Context, ids []string) error {
	return s.repository.DeleteInvites(ctx, ids)
}
