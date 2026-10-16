package application

import (
	"context"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/file/domain"
	"github.com/example/medusa-go-clone/internal/modules/file/domain/models"
)

type Service struct{ provider domain.Provider }

func New(providers map[string]domain.Provider) (*Service, error) {
	if len(providers) != 1 {
		return nil, fmt.Errorf("file module should be initialized with exactly one provider")
	}
	for _, p := range providers {
		return &Service{provider: p}, nil
	}
	return nil, fmt.Errorf("file provider missing")
}
func (s *Service) Upload(ctx context.Context, in []models.UploadInput) ([]models.File, error) {
	out := make([]models.File, 0, len(in))
	for _, file := range in {
		v, err := s.provider.Upload(ctx, file)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
func (s *Service) Delete(ctx context.Context, ids []string) error { return s.provider.Delete(ctx, ids) }
func (s *Service) Get(ctx context.Context, id string) (models.File, error) {
	url, err := s.provider.PresignedDownloadURL(ctx, id)
	if err != nil {
		return models.File{}, err
	}
	return models.File{ID: id, URL: url}, nil
}
func (s *Service) PresignedUpload(ctx context.Context, in models.PresignedInput) (models.PresignedOutput, error) {
	file, err := s.provider.PresignedUploadURL(ctx, in.Filename, in.MIMEType, in.Access)
	if err != nil {
		return models.PresignedOutput{}, err
	}
	return models.PresignedOutput{URL: file.URL, Filename: file.ID, MIMEType: in.MIMEType, Size: in.Size, Extension: extension(in.MIMEType), OriginalName: in.OriginalName}, nil
}
func extension(mimeType string) string {
	for i := len(mimeType) - 1; i >= 0; i-- {
		if mimeType[i] == '/' {
			return mimeType[i+1:]
		}
	}
	return ""
}
