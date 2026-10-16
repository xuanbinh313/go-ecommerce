package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/translation/domain/models"
)

type Repository interface {
	ListLocales(context.Context, models.LocaleFilter, int, int) ([]models.Locale, int, error)
	GetLocaleByCode(context.Context, string) (models.Locale, error)
	List(context.Context, models.Filter) ([]models.Translation, int, error)
	Batch(context.Context, models.BatchInput) ([]models.Translation, []models.Translation, error)
	ListSettings(context.Context, models.TranslationSettingsFilter) ([]models.TranslationSetting, error)
	CreateSettings(context.Context, []models.CreateTranslationSettingInput) ([]models.TranslationSetting, error)
	UpdateSettings(context.Context, []models.UpdateTranslationSettingInput) ([]models.TranslationSetting, error)
	DeleteSettings(context.Context, []string) error
	RestoreSettings(context.Context, []models.TranslationSetting) error
}
