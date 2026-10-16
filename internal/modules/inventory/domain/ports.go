package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/inventory/domain/models"
)

type Repository interface {
	ListItems(context.Context, models.Filter) ([]models.InventoryItem, int, error)
	GetItem(context.Context, string) (models.InventoryItem, error)
	CreateItem(context.Context, models.ItemInput) (models.InventoryItem, error)
	UpdateItem(context.Context, string, models.UpdateItemInput) (models.InventoryItem, error)
	DeleteItem(context.Context, string) error
	ListLevels(context.Context, string, []string, int, int) ([]models.InventoryLevel, int, error)
	CreateLevels(context.Context, []models.LevelInput) ([]models.InventoryLevel, error)
	UpdateLevels(context.Context, []models.LevelUpdate) ([]models.InventoryLevel, error)
	DeleteLevels(context.Context, []string, bool) ([]string, error)
	UpdateLevelAtLocation(context.Context, string, string, models.LevelUpdate) (models.InventoryItem, error)
	DeleteLevelAtLocation(context.Context, string, string) (string, error)
	LocationsExist(context.Context, []string) (bool, error)
	ListReservations(context.Context, models.ReservationFilter) ([]models.ReservationItem, int, error)
	GetReservation(context.Context, string) (models.ReservationItem, error)
	CreateReservations(context.Context, []models.ReservationInput, *string) ([]models.ReservationItem, error)
	UpdateReservations(context.Context, []models.ReservationUpdate) ([]models.ReservationItem, error)
	DeleteReservations(context.Context, []string) error
	RestoreReservations(context.Context, []string) error
}
