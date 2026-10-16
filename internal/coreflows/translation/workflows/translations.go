package workflows

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/translation/domain/models"
)

const BatchTranslationsWorkflowID = "batch-translations"
const CreateTranslationsWorkflowID = "create-translations"
const CreateTranslationsStepID = "create-translations"
const UpdateTranslationsWorkflowID = "update-translations"
const UpdateTranslationsStepID = "update-translations"
const DeleteTranslationsWorkflowID = "delete-translations"
const DeleteTranslationsStepID = "delete-translations"

type Service interface {
	Batch(context.Context, models.BatchInput) ([]models.Translation, []models.Translation, error)
}

func BatchTranslations(ctx context.Context, s Service, in models.BatchInput) ([]models.Translation, []models.Translation, error) {
	return s.Batch(ctx, in)
}
