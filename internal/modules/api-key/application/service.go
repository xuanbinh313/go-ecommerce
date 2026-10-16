package application

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/api-key/domain"
	"github.com/example/medusa-go-clone/internal/modules/api-key/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"golang.org/x/crypto/scrypt"
)

type Service struct{ repository domain.Repository }

func NewService(repository domain.Repository) *Service { return &Service{repository: repository} }
func (s *Service) List(ctx context.Context, f models.Filter) ([]models.APIKey, int, error) {
	return s.repository.List(ctx, f)
}
func (s *Service) Get(ctx context.Context, id string) (models.APIKey, error) {
	return s.repository.Get(ctx, id)
}

// PublishableKeySalesChannels returns the sales-channel scope associated with
// a source-compatible x-publishable-api-key header.
func (s *Service) PublishableKeySalesChannels(ctx context.Context, token string) ([]string, error) {
	if token == "" {
		return []string{}, nil
	}
	key, err := s.repository.GetPublishableKeyByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(key.SalesChannels))
	for _, channel := range key.SalesChannels {
		ids = append(ids, channel.ID)
	}
	return ids, nil
}

func (s *Service) Create(ctx context.Context, in models.CreateInput) (models.Created, error) {
	if in.Title == "" || (in.Type != "publishable" && in.Type != "secret") {
		return models.Created{}, fmt.Errorf("title and valid type are required")
	}
	raw, err := randomToken(in.Type)
	if err != nil {
		return models.Created{}, err
	}
	salt := ""
	token := raw
	if in.Type == "secret" {
		saltBytes := make([]byte, 16)
		if _, err = rand.Read(saltBytes); err != nil {
			return models.Created{}, err
		}
		salt = fmt.Sprintf("%x", saltBytes)
		hashed, e := scrypt.Key([]byte(raw), []byte(salt), 16384, 8, 1, 64)
		if e != nil {
			return models.Created{}, e
		}
		token = fmt.Sprintf("%x", hashed)
	}
	key := models.APIKey{ID: entityid.New("apk"), Title: in.Title, Token: token, Salt: salt, Redacted: redact(raw), Type: in.Type, CreatedBy: in.CreatedBy}
	created, err := s.repository.Create(ctx, key)
	return models.Created{APIKey: created, RawToken: raw}, err
}
func (s *Service) Update(ctx context.Context, id string, in models.UpdateInput) (models.APIKey, error) {
	if in.Title == "" {
		return models.APIKey{}, fmt.Errorf("title is required")
	}
	if err := s.repository.Update(ctx, id, in); err != nil {
		return models.APIKey{}, err
	}
	return s.repository.Get(ctx, id)
}
func (s *Service) Revoke(ctx context.Context, id string, in models.RevokeInput) (models.APIKey, error) {
	if in.RevokedBy == "" {
		return models.APIKey{}, fmt.Errorf("revoked_by is required")
	}
	if in.RevokeIn != nil && *in.RevokeIn < 0 {
		return models.APIKey{}, fmt.Errorf("revoke_in must be greater than or equal to zero")
	}
	if err := s.repository.Revoke(ctx, id, in); err != nil {
		return models.APIKey{}, err
	}
	return s.repository.Get(ctx, id)
}
func (s *Service) Delete(ctx context.Context, id string) error { return s.repository.Delete(ctx, id) }
func (s *Service) LinkSalesChannels(ctx context.Context, id string, add, remove []string) error {
	if len(add) > 0 {
		ok, err := s.repository.SalesChannelsExist(ctx, add)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("one or more sales channels do not exist")
		}
	}
	if len(remove) > 0 {
		if err := s.repository.DeleteSalesChannelLinks(ctx, id, remove); err != nil {
			return err
		}
	}
	if len(add) > 0 {
		return s.repository.CreateSalesChannelLinks(ctx, id, add)
	}
	return nil
}
func randomToken(kind string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	prefix := "pk_"
	if kind == "secret" {
		prefix = "sk_"
	}
	return prefix + fmt.Sprintf("%x", b), nil
}
func redact(token string) string {
	if len(token) < 9 {
		return token
	}
	return strings.Join([]string{token[:6], token[len(token)-3:]}, "***")
}
func RevocationTime(in models.RevokeInput) time.Time {
	at := time.Now()
	if in.RevokeIn != nil {
		at = at.Add(time.Duration(*in.RevokeIn) * time.Second)
	}
	return at
}
