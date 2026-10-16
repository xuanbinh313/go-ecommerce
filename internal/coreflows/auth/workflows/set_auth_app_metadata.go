package workflows

import (
	"context"

	"github.com/example/medusa-go-clone/internal/modules/auth/domain"
)

const SetAuthAppMetadataStepID = "set-auth-app-metadata"

func SetAuthAppMetadataStep(ctx context.Context, auth domain.IdentityMetadata, identityID, actorType, value string) error {
	if auth == nil {
		return domain.ErrIdentityNotFound
	}
	_, err := auth.SetActorMetadata(ctx, identityID, actorType, value)
	return err
}
