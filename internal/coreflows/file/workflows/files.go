package workflows

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/file/domain/models"
)

const UploadFilesWorkflowID = "upload-files"
const UploadFilesStepID = "upload-files"
const DeleteFilesWorkflowID = "delete-files"
const DeleteFilesStepID = "delete-files"

type Service interface {
	Upload(context.Context, []models.UploadInput) ([]models.File, error)
	Delete(context.Context, []string) error
}

func UploadFiles(ctx context.Context, s Service, in []models.UploadInput) ([]models.File, error) {
	return s.Upload(ctx, in)
}
func DeleteFiles(ctx context.Context, s Service, ids []string) error { return s.Delete(ctx, ids) }
