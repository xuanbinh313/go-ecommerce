package domain

import (
	"context"

	"github.com/example/medusa-go-clone/internal/modules/settings/domain/models"
)

type Repository interface {
	ListPropertyLabels(context.Context, models.PropertyLabelFilter, int, int) ([]models.PropertyLabel, int, error)
	GetPropertyLabel(context.Context, string) (models.PropertyLabel, error)
	CreatePropertyLabels(context.Context, []models.PropertyLabelInput) ([]models.PropertyLabel, error)
	UpdatePropertyLabel(context.Context, string, map[string]any) (models.PropertyLabel, error)
	DeletePropertyLabels(context.Context, []string) ([]models.PropertyLabel, error)
	ListLayoutConfigurations(context.Context, string, models.LayoutConfigurationFilter, int, int) ([]models.LayoutConfiguration, int, error)
	GetLayoutConfiguration(context.Context, string, string) (models.LayoutConfigurations, error)
	SetLayoutConfiguration(context.Context, string, string, bool, []byte) error
	ClearLayoutConfiguration(context.Context, string, string) error
}
