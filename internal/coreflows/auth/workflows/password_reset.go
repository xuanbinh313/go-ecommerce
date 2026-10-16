package workflows

import (
	"context"
	"errors"
	"time"
)

const GenerateResetPasswordTokenWorkflowID = "generate-reset-password-token"
const CreatePasswordResetTokenStepID = "create-password-reset-token"
const AuthPasswordResetEvent = "auth.password_reset"

type PasswordResetService interface {
	CreatePasswordResetToken(context.Context, string, string, time.Duration) (string, time.Time, error)
	ConsumePasswordResetToken(context.Context, string, string, string) error
	IssuePasswordResetToken(string, string, string, string, time.Time) (string, error)
}
type PasswordResetEventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}

// GenerateResetPasswordToken stores a one-use JTI, signs Medusa's reset JWT claims, and emits auth.password_reset.
func GenerateResetPasswordToken(ctx context.Context, auth PasswordResetService, events PasswordResetEventPublisher, entityID, actorType, provider string, metadata map[string]any) (string, error) {
	const ttl = 15 * time.Minute
	jti, expires, err := auth.CreatePasswordResetToken(ctx, provider, entityID, ttl)
	if err != nil {
		return "", err
	}
	rollback := func(cause error) error {
		if e := auth.ConsumePasswordResetToken(ctx, jti, provider, entityID); e != nil {
			cause = errors.Join(cause, e)
		}
		return cause
	}
	token, err := auth.IssuePasswordResetToken(entityID, provider, actorType, jti, expires)
	if err != nil {
		return "", rollback(err)
	}
	if events == nil {
		return "", rollback(errors.New("auth event publisher is required"))
	}
	if err = events.Publish(ctx, AuthPasswordResetEvent, []map[string]any{{"entity_id": entityID, "actor_type": actorType, "token": token, "metadata": metadata}}); err != nil {
		return "", rollback(err)
	}
	return token, nil
}
