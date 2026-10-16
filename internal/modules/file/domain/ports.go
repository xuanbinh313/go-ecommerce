package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/file/domain/models"
)

type Provider interface {
	Upload(context.Context, models.UploadInput) (models.File, error)
	Delete(context.Context, []string) error
	PresignedDownloadURL(context.Context, string) (string, error)
	PresignedUploadURL(context.Context, string, string, string) (models.File, error)
}
