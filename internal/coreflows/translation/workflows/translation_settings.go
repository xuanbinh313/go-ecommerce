package workflows

import (
	"context"
	"errors"

	"github.com/example/medusa-go-clone/internal/modules/translation/domain/models"
)

const (
	BatchTranslationSettingsWorkflowID = "batch-translation-settings"
	CreateTranslationSettingsStepID    = "create-translation-settings"
	UpdateTranslationSettingsStepID    = "update-translation-settings"
	DeleteTranslationSettingsStepID    = "delete-translation-settings"
)

type TranslationSettingsService interface {
	ListSettings(context.Context, models.TranslationSettingsFilter) ([]models.TranslationSetting, error)
	CreateSettings(context.Context, []models.CreateTranslationSettingInput) ([]models.TranslationSetting, error)
	UpdateSettings(context.Context, []models.UpdateTranslationSettingInput) ([]models.TranslationSetting, error)
	DeleteSettings(context.Context, []string) error
	RestoreSettings(context.Context, []models.TranslationSetting) error
}

func BatchTranslationSettings(ctx context.Context, service TranslationSettingsService, input models.BatchTranslationSettingsInput) ([]models.TranslationSetting, []models.TranslationSetting, error) {
	var updatedBefore []models.TranslationSetting
	var err error
	if ids := settingUpdateIDs(input.Update); len(ids) > 0 {
		updatedBefore, err = service.ListSettings(ctx, models.TranslationSettingsFilter{IDs: ids})
		if err != nil {
			return nil, nil, err
		}
	}
	created, err := service.CreateSettings(ctx, input.Create)
	if err != nil {
		return nil, nil, err
	}
	updated, err := service.UpdateSettings(ctx, input.Update)
	if err != nil {
		return nil, nil, errors.Join(err, service.DeleteSettings(ctx, settingIDs(created)))
	}
	if err = service.DeleteSettings(ctx, input.Delete); err != nil {
		return nil, nil, errors.Join(err, service.RestoreSettings(ctx, updatedBefore), service.DeleteSettings(ctx, settingIDs(created)))
	}
	return created, updated, nil
}

func settingUpdateIDs(updates []models.UpdateTranslationSettingInput) []string {
	ids := make([]string, 0, len(updates))
	for _, input := range updates {
		ids = append(ids, input.ID)
	}
	return ids
}

func settingIDs(settings []models.TranslationSetting) []string {
	ids := make([]string, 0, len(settings))
	for _, setting := range settings {
		ids = append(ids, setting.ID)
	}
	return ids
}
