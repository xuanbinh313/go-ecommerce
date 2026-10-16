package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/auth/domain/models"
)

const RequestVerificationWorkflowID = "request-verification-workflow"
const RequestVerificationStepID = "request-verification"
const AuthVerificationRequestedEvent = "auth.verification_requested"

type AuthVerificationService interface {
	RequestAuthVerification(context.Context, models.RequestAuthVerificationInput, time.Duration) (models.AuthVerification, error)
	ConfirmAuthVerification(context.Context, models.ConfirmAuthVerificationInput, time.Duration) (models.AuthVerification, error)
}

func RequestAuthVerificationStep(ctx context.Context, auth AuthVerificationService, input models.RequestAuthVerificationInput, ttl time.Duration) (models.AuthVerification, error) {
	if auth == nil {
		return models.AuthVerification{}, errors.New("auth verification service is required")
	}
	return auth.RequestAuthVerification(ctx, input, ttl)
}

// RequestAuthVerification stores the token hash, publishes the source event with the raw code,
// and returns the source workflow result without the code or expiration timestamp.
func RequestAuthVerification(ctx context.Context, auth AuthVerificationService, events PasswordResetEventPublisher, input models.RequestAuthVerificationInput, ttl time.Duration) (models.AuthVerification, error) {
	result, err := RequestAuthVerificationStep(ctx, auth, input, ttl)
	if err != nil {
		return models.AuthVerification{}, err
	}
	metadata := map[string]any{}
	if len(result.Metadata) > 0 && string(result.Metadata) != "null" {
		_ = json.Unmarshal(result.Metadata, &metadata)
	}
	if events == nil {
		return models.AuthVerification{}, errors.New("auth event publisher is required")
	}
	data := map[string]any{
		"entity_id":        result.EntityID,
		"entity_type":      result.EntityType,
		"code_provider":    result.CodeProvider,
		"auth_identity_id": result.AuthIdentityID,
		"metadata":         metadata,
	}
	if result.Code != "" {
		data["code"] = result.Code
	}
	if result.ExpiresAt != nil {
		data["expires_at"] = result.ExpiresAt
	}
	if err = events.Publish(ctx, AuthVerificationRequestedEvent, []map[string]any{data}); err != nil {
		return models.AuthVerification{}, err
	}
	result.Code = ""
	result.ExpiresAt = nil
	return result, nil
}
