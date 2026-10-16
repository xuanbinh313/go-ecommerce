package workflows

import (
	"context"
	"encoding/json"
	"time"

	inventory "github.com/example/medusa-go-clone/internal/modules/inventory/domain/models"
)

const (
	CreateReservationsWorkflowID = "create-reservations-workflow"
	CreateReservationsStepID     = "create-reservations-step"
	UpdateReservationsWorkflowID = "update-reservations-workflow"
	UpdateReservationsStepID     = "update-reservations-step"
	DeleteReservationsWorkflowID = "delete-reservations"
	DeleteReservationsStepID     = "delete-reservations"
	ReservationCreatedEvent      = "inventory.reservation-item.created"
	ReservationUpdatedEvent      = "inventory.reservation-item.updated"
	ReservationDeletedEvent      = "inventory.reservation-item.deleted"
)

type Service interface {
	ListReservations(context.Context, inventory.ReservationFilter) ([]inventory.ReservationItem, int, error)
	GetReservation(context.Context, string) (inventory.ReservationItem, error)
	CreateReservations(context.Context, []inventory.ReservationInput, *string) ([]inventory.ReservationItem, error)
	UpdateReservations(context.Context, []inventory.ReservationUpdate) ([]inventory.ReservationItem, error)
	DeleteReservations(context.Context, []string) error
	RestoreReservations(context.Context, []string) error
}

type LockExecutor interface {
	Execute(context.Context, []string, time.Duration, func(context.Context) error) error
}

type EventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}

func CreateReservations(ctx context.Context, service Service, locks LockExecutor, events EventPublisher, input []inventory.ReservationInput, actor *string) ([]inventory.ReservationItem, error) {
	keys := make([]string, 0, len(input))
	for _, item := range input {
		keys = append(keys, item.InventoryItemID)
	}
	var created []inventory.ReservationItem
	err := locks.Execute(ctx, keys, 2*time.Second, func(ctx context.Context) error {
		var err error
		created, err = service.CreateReservations(ctx, input, actor)
		return err
	})
	if err != nil {
		return nil, err
	}
	rows := reservationEvents(created)
	if err = events.Publish(ctx, ReservationCreatedEvent, rows); err != nil {
		ids := reservationIDs(created)
		_ = locks.Execute(ctx, keys, 2*time.Second, func(ctx context.Context) error { return service.DeleteReservations(ctx, ids) })
		return nil, err
	}
	return created, nil
}

func UpdateReservations(ctx context.Context, service Service, locks LockExecutor, events EventPublisher, input []inventory.ReservationUpdate) ([]inventory.ReservationItem, error) {
	before := make([]inventory.ReservationItem, 0, len(input))
	keys := make([]string, 0, len(input))
	for _, update := range input {
		current, err := service.GetReservation(ctx, update.ID)
		if err != nil {
			return nil, err
		}
		before = append(before, current)
		keys = append(keys, current.InventoryItemID)
	}
	var updated []inventory.ReservationItem
	err := locks.Execute(ctx, keys, 2*time.Second, func(ctx context.Context) error {
		var err error
		updated, err = service.UpdateReservations(ctx, input)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err = events.Publish(ctx, ReservationUpdatedEvent, reservationEvents(updated)); err != nil {
		rollback := make([]inventory.ReservationUpdate, 0, len(before))
		for _, item := range before {
			description, _ := json.Marshal(item.Description)
			rollback = append(rollback, inventory.ReservationUpdate{ID: item.ID, LocationID: &item.LocationID, Quantity: &item.Quantity, Description: description, Metadata: item.Metadata})
		}
		_ = locks.Execute(ctx, keys, 2*time.Second, func(ctx context.Context) error { _, err := service.UpdateReservations(ctx, rollback); return err })
		return nil, err
	}
	return updated, nil
}

func DeleteReservations(ctx context.Context, service Service, locks LockExecutor, events EventPublisher, ids []string) error {
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		item, err := service.GetReservation(ctx, id)
		if err != nil {
			return err
		}
		keys = append(keys, item.InventoryItemID)
	}
	if err := locks.Execute(ctx, keys, 2*time.Second, func(ctx context.Context) error { return service.DeleteReservations(ctx, ids) }); err != nil {
		return err
	}
	rows := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, map[string]any{"id": id})
	}
	if err := events.Publish(ctx, ReservationDeletedEvent, rows); err != nil {
		_ = locks.Execute(ctx, keys, 2*time.Second, func(ctx context.Context) error { return service.RestoreReservations(ctx, ids) })
		return err
	}
	return nil
}

func reservationIDs(items []inventory.ReservationItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func reservationEvents(items []inventory.ReservationItem) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{"id": item.ID})
	}
	return rows
}
