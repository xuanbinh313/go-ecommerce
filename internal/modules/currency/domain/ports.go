package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/currency/domain/models"
)

type Repository interface {
	Retrieve(context.Context, string) (models.Currency, error)
	List(context.Context, []string) ([]models.Currency, error)
	Upsert(context.Context, []models.Currency) ([]models.Currency, error)
}
