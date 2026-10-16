package application

import (
	"context"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/inventory/domain"
	"github.com/example/medusa-go-clone/internal/modules/inventory/domain/models"
)

type Service struct{ repository domain.Repository }

func NewService(r domain.Repository) *Service { return &Service{repository: r} }
func (s *Service) ListItems(c context.Context, f models.Filter) ([]models.InventoryItem, int, error) {
	return s.repository.ListItems(c, f)
}
func (s *Service) GetItem(c context.Context, id string) (models.InventoryItem, error) {
	return s.repository.GetItem(c, id)
}
func (s *Service) CreateItem(c context.Context, in models.ItemInput) (models.InventoryItem, error) {
	if in.RequiresShipping == nil {
		v := true
		in.RequiresShipping = &v
	}
	locationIDs := []string{}
	for _, level := range in.LocationLevels {
		locationIDs = append(locationIDs, level.LocationID)
		if level.StockedQuantity != nil && *level.StockedQuantity < 0 {
			return models.InventoryItem{}, fmt.Errorf("stocked_quantity must be non-negative")
		}
		if level.IncomingQuantity != nil && *level.IncomingQuantity < 0 {
			return models.InventoryItem{}, fmt.Errorf("incoming_quantity must be non-negative")
		}
	}
	if len(locationIDs) > 0 {
		ok, err := s.repository.LocationsExist(c, locationIDs)
		if err != nil {
			return models.InventoryItem{}, err
		}
		if !ok {
			return models.InventoryItem{}, fmt.Errorf("one or more inventory locations do not exist")
		}
	}
	return s.repository.CreateItem(c, in)
}
func (s *Service) UpdateItem(c context.Context, id string, in models.UpdateItemInput) (models.InventoryItem, error) {
	return s.repository.UpdateItem(c, id, in)
}
func (s *Service) DeleteItem(c context.Context, id string) error {
	return s.repository.DeleteItem(c, id)
}
func (s *Service) ListLevels(c context.Context, itemID string, locations []string, limit, offset int) ([]models.InventoryLevel, int, error) {
	return s.repository.ListLevels(c, itemID, locations, limit, offset)
}
func (s *Service) CreateLevels(c context.Context, in []models.LevelInput) ([]models.InventoryLevel, error) {
	locationIDs := []string{}
	for _, v := range in {
		locationIDs = append(locationIDs, v.LocationID)
		if v.StockedQuantity != nil && *v.StockedQuantity < 0 {
			return nil, fmt.Errorf("stocked_quantity must be non-negative")
		}
		if v.IncomingQuantity != nil && *v.IncomingQuantity < 0 {
			return nil, fmt.Errorf("incoming_quantity must be non-negative")
		}
	}
	if len(locationIDs) > 0 {
		ok, err := s.repository.LocationsExist(c, locationIDs)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("one or more inventory locations do not exist")
		}
	}
	return s.repository.CreateLevels(c, in)
}
func (s *Service) UpdateLevels(c context.Context, in []models.LevelUpdate) ([]models.InventoryLevel, error) {
	for _, v := range in {
		if v.StockedQuantity != nil && *v.StockedQuantity < 0 {
			return nil, fmt.Errorf("stocked_quantity must be non-negative")
		}
		if v.IncomingQuantity != nil && *v.IncomingQuantity < 0 {
			return nil, fmt.Errorf("incoming_quantity must be non-negative")
		}
	}
	return s.repository.UpdateLevels(c, in)
}
func (s *Service) DeleteLevels(c context.Context, ids []string, force bool) ([]string, error) {
	return s.repository.DeleteLevels(c, ids, force)
}
func (s *Service) UpdateLevelAtLocation(c context.Context, id, location string, in models.LevelUpdate) (models.InventoryItem, error) {
	if in.StockedQuantity != nil && *in.StockedQuantity < 0 {
		return models.InventoryItem{}, fmt.Errorf("stocked_quantity must be non-negative")
	}
	if in.IncomingQuantity != nil && *in.IncomingQuantity < 0 {
		return models.InventoryItem{}, fmt.Errorf("incoming_quantity must be non-negative")
	}
	return s.repository.UpdateLevelAtLocation(c, id, location, in)
}
func (s *Service) DeleteLevelAtLocation(c context.Context, id, location string) (string, error) {
	return s.repository.DeleteLevelAtLocation(c, id, location)
}

func (s *Service) ListReservations(c context.Context, f models.ReservationFilter) ([]models.ReservationItem, int, error) {
	return s.repository.ListReservations(c, f)
}
func (s *Service) GetReservation(c context.Context, id string) (models.ReservationItem, error) {
	return s.repository.GetReservation(c, id)
}
func (s *Service) CreateReservations(c context.Context, input []models.ReservationInput, actor *string) ([]models.ReservationItem, error) {
	for _, reservation := range input {
		if reservation.InventoryItemID == "" || reservation.LocationID == "" {
			return nil, fmt.Errorf("inventory_item_id and location_id are required")
		}
		if reservation.Quantity < 0 {
			return nil, fmt.Errorf("quantity must be non-negative")
		}
	}
	return s.repository.CreateReservations(c, input, actor)
}
func (s *Service) UpdateReservations(c context.Context, input []models.ReservationUpdate) ([]models.ReservationItem, error) {
	return s.repository.UpdateReservations(c, input)
}
func (s *Service) DeleteReservations(c context.Context, ids []string) error {
	return s.repository.DeleteReservations(c, ids)
}
func (s *Service) RestoreReservations(c context.Context, ids []string) error {
	return s.repository.RestoreReservations(c, ids)
}
