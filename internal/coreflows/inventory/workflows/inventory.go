package workflows

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/inventory/domain/models"
)

const CreateInventoryItemsWorkflowID = "create-inventory-items-workflow"
const CreateInventoryItemsStepID = "create-inventory-items"
const UpdateInventoryItemsWorkflowID = "update-inventory-items-workflow"
const UpdateInventoryItemsStepID = "update-inventory-items-step"
const DeleteInventoryItemWorkflowID = "delete-inventory-item-workflow"
const DeleteInventoryItemStepID = "delete-inventory-item-step"
const CreateInventoryLevelsWorkflowID = "create-inventory-levels-workflow"
const CreateInventoryLevelsStepID = "create-inventory-levels"
const UpdateInventoryLevelsWorkflowID = "update-inventory-levels-workflow"
const UpdateInventoryLevelsStepID = "update-inventory-levels-step"
const DeleteInventoryLevelsWorkflowID = "delete-inventory-levels-workflow"
const DeleteInventoryLevelsStepID = "delete-inventory-levels-step"
const BatchInventoryItemLevelsWorkflowID = "batch-inventory-item-levels-workflow"

type Service interface {
	CreateItem(context.Context, models.ItemInput) (models.InventoryItem, error)
	UpdateItem(context.Context, string, models.UpdateItemInput) (models.InventoryItem, error)
	DeleteItem(context.Context, string) error
	CreateLevels(context.Context, []models.LevelInput) ([]models.InventoryLevel, error)
	UpdateLevels(context.Context, []models.LevelUpdate) ([]models.InventoryLevel, error)
	DeleteLevels(context.Context, []string, bool) ([]string, error)
}

func CreateInventoryItems(c context.Context, s Service, in models.ItemInput) (models.InventoryItem, error) {
	return s.CreateItem(c, in)
}
func UpdateInventoryItems(c context.Context, s Service, id string, in models.UpdateItemInput) (models.InventoryItem, error) {
	return s.UpdateItem(c, id, in)
}
func DeleteInventoryItem(c context.Context, s Service, id string) error { return s.DeleteItem(c, id) }
func CreateInventoryLevels(c context.Context, s Service, in []models.LevelInput) ([]models.InventoryLevel, error) {
	return s.CreateLevels(c, in)
}
func UpdateInventoryLevels(c context.Context, s Service, in []models.LevelUpdate) ([]models.InventoryLevel, error) {
	return s.UpdateLevels(c, in)
}
func DeleteInventoryLevels(c context.Context, s Service, ids []string, force bool) ([]string, error) {
	return s.DeleteLevels(c, ids, force)
}
