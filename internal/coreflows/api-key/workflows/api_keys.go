package workflows

import (
	"context"
	"errors"
	"github.com/example/medusa-go-clone/internal/modules/api-key/domain/models"
)

const CreateApiKeysWorkflowID = "create-api-keys"
const CreateApiKeysStepID = "create-api-keys"
const UpdateApiKeysWorkflowID = "update-api-keys"
const RevokeApiKeysWorkflowID = "revoke-api-keys"
const DeleteApiKeysWorkflowID = "delete-api-keys"
const LinkSalesChannelsToApiKeyWorkflowID = "link-sales-channels-to-api-key"

type Service interface {
	Create(context.Context, models.CreateInput) (models.Created, error)
	Update(context.Context, string, models.UpdateInput) (models.APIKey, error)
	Revoke(context.Context, string, models.RevokeInput) (models.APIKey, error)
	Delete(context.Context, string) error
	LinkSalesChannels(context.Context, string, []string, []string) error
	Get(context.Context, string) (models.APIKey, error)
}

func CreateApiKeys(ctx context.Context, s Service, input models.CreateInput) (models.Created, error) {
	return s.Create(ctx, input)
}
func UpdateApiKeys(ctx context.Context, s Service, id string, input models.UpdateInput) (models.APIKey, error) {
	return s.Update(ctx, id, input)
}
func RevokeApiKeys(ctx context.Context, s Service, id string, input models.RevokeInput) (models.APIKey, error) {
	return s.Revoke(ctx, id, input)
}
func DeleteApiKeys(ctx context.Context, s Service, id string) error { return s.Delete(ctx, id) }
func LinkSalesChannelsToApiKey(ctx context.Context, s Service, id string, add, remove []string) (models.APIKey, error) {
	key, err := s.Get(ctx, id)
	if err != nil {
		return key, err
	}
	if key.Type != "publishable" {
		return key, errors.New("sales channels can only be associated with publishable API keys")
	}
	if err = s.LinkSalesChannels(ctx, id, add, remove); err != nil {
		return key, err
	}
	return s.Get(ctx, id)
}
