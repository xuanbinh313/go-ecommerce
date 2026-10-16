package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/links/domain/models"
)

type Repository interface {
	Create(context.Context, models.Definition, []map[string]any) ([]models.Link, error)
	Update(context.Context, models.Definition, map[string]any, map[string]any) ([]models.Link, error)
	List(context.Context, models.Definition, map[string]any) ([]models.Link, error)
	Dismiss(context.Context, models.Definition, []map[string]any) ([]models.Link, error)
	Delete(context.Context, models.Definition, []map[string]any) error
	Restore(context.Context, models.Definition, []map[string]any) ([]models.Link, error)
}
