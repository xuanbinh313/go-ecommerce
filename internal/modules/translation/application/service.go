package application

import (
	"context"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/translation/domain"
	"github.com/example/medusa-go-clone/internal/modules/translation/domain/models"
)

type Service struct{ repo domain.Repository }

func NewService(r domain.Repository) *Service { return &Service{repo: r} }
func (s *Service) ListLocales(ctx context.Context, f models.LocaleFilter, limit, offset int) ([]models.Locale, int, error) {
	return s.repo.ListLocales(ctx, f, limit, offset)
}
func (s *Service) GetLocaleByCode(ctx context.Context, code string) (models.Locale, error) {
	return s.repo.GetLocaleByCode(ctx, code)
}
func (s *Service) List(c context.Context, f models.Filter) ([]models.Translation, int, error) {
	return s.repo.List(c, f)
}
func (s *Service) Batch(c context.Context, in models.BatchInput) ([]models.Translation, []models.Translation, error) {
	for _, v := range in.Create {
		if v.ReferenceID == "" || v.Reference == "" || v.LocaleCode == "" {
			return nil, nil, fmt.Errorf("reference_id, reference, and locale_code are required")
		}
	}
	for _, v := range in.Update {
		if v.ID == "" {
			return nil, nil, fmt.Errorf("translation update id is required")
		}
	}
	return s.repo.Batch(c, in)
}

func (s *Service) ListSettings(c context.Context, filter models.TranslationSettingsFilter) ([]models.TranslationSetting, error) {
	return s.repo.ListSettings(c, filter)
}

func (s *Service) CreateSettings(c context.Context, in []models.CreateTranslationSettingInput) ([]models.TranslationSetting, error) {
	for _, setting := range in {
		if setting.EntityType == "" {
			return nil, fmt.Errorf("entity_type is required")
		}
	}
	return s.repo.CreateSettings(c, in)
}
func (s *Service) UpdateSettings(c context.Context, in []models.UpdateTranslationSettingInput) ([]models.TranslationSetting, error) {
	for _, setting := range in {
		if setting.ID == "" {
			return nil, fmt.Errorf("translation setting update id is required")
		}
	}
	return s.repo.UpdateSettings(c, in)
}
func (s *Service) DeleteSettings(c context.Context, ids []string) error {
	return s.repo.DeleteSettings(c, ids)
}
func (s *Service) RestoreSettings(c context.Context, rows []models.TranslationSetting) error {
	return s.repo.RestoreSettings(c, rows)
}
