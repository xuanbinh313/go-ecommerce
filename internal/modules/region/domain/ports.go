package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/region/domain/models"
)

type Repository interface {
	Create(context.Context, models.RegionInput) (models.Region, error)
	Get(context.Context, string) (models.Region, error)
	List(context.Context) ([]models.Region, error)
	Update(context.Context, string, models.RegionInput) (models.Region, error)
	SoftDelete(context.Context, []string) error
	RestoreRegions(context.Context, []string) error
	DeleteRegions(context.Context, []string) error
	Countries(context.Context, []string, string) ([]models.Country, error)
}

// WorkflowService is the domain boundary used by direct region workflows.
// Implementations retain country normalization and assignment validation.
type WorkflowService interface {
	Create(context.Context, models.RegionInput) (models.Region, error)
	DeleteRegions(context.Context, []string) error
}
