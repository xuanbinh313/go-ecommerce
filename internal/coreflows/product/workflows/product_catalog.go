package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/product/domain"
	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
)

const (
	CreateProductTypesWorkflowID = "create-product-types"
	CreateProductTypesStepID     = "create-product-types"
	UpdateProductTypesWorkflowID = "update-product-types"
	UpdateProductTypesStepID     = "update-product-types"
	DeleteProductTypesWorkflowID = "delete-product-types"
	DeleteProductTypesStepID     = "delete-product-types"
	CreateProductTagsWorkflowID  = "create-product-tags"
	CreateProductTagsStepID      = "create-product-tags"
	UpdateProductTagsWorkflowID  = "update-product-tags"
	UpdateProductTagsStepID      = "update-product-tags"
	DeleteProductTagsWorkflowID  = "delete-product-tags"
	DeleteProductTagsStepID      = "delete-product-tags"
)

func CreateProductTypes(ctx context.Context, service domain.Repository, events EventPublisher, input []models.ProductTypeInput) ([]models.ProductType, error) {
	items, err := service.CreateProductTypes(ctx, input)
	if err != nil {
		return nil, err
	}
	ids := productTypeIDValues(items)
	if err = publishCatalogEvents(ctx, events, "product-type.created", idsToEvents(ids)); err != nil {
		if rollback := service.HardDeleteProductTypes(ctx, ids); rollback != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", CreateProductTypesStepID, rollback))
		}
		return nil, err
	}
	return items, nil
}

func UpdateProductType(ctx context.Context, service domain.Repository, events EventPublisher, id string, fields map[string]any) (models.ProductType, error) {
	previous, err := service.GetProductType(ctx, id)
	if err != nil {
		return models.ProductType{}, err
	}
	updated, err := service.UpdateProductType(ctx, id, fields)
	if err != nil {
		return models.ProductType{}, err
	}
	if err = publishCatalogEvents(ctx, events, "product-type.updated", []map[string]any{{"id": id}}); err != nil {
		if _, rollback := service.UpdateProductType(ctx, id, catalogSnapshot(previous)); rollback != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", UpdateProductTypesStepID, rollback))
		}
		return models.ProductType{}, err
	}
	return updated, nil
}

func DeleteProductTypes(ctx context.Context, service domain.Repository, events EventPublisher, ids []string) error {
	if err := service.SoftDeleteProductTypes(ctx, ids); err != nil {
		return err
	}
	references, err := service.DetachProductTypeReferences(ctx, ids)
	if err != nil {
		if rollback := service.RestoreProductTypes(ctx, ids); rollback != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", DeleteProductTypesStepID, rollback))
		}
		return err
	}
	if err := publishCatalogEvents(ctx, events, "product-type.deleted", idsToEvents(ids)); err != nil {
		if rollback := service.RestoreProductTypes(ctx, ids); rollback != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", DeleteProductTypesStepID, rollback))
		} else if rollback := service.RestoreProductTypeReferences(ctx, references); rollback != nil {
			err = errors.Join(err, fmt.Errorf("restore product type relationships: %w", rollback))
		}
		return err
	}
	return nil
}

func CreateProductTags(ctx context.Context, service domain.Repository, events EventPublisher, input []models.ProductTagInput) ([]models.ProductTag, error) {
	items, err := service.CreateProductTags(ctx, input)
	if err != nil {
		return nil, err
	}
	ids := productTagIDValues(items)
	if err = publishCatalogEvents(ctx, events, "product-tag.created", idsToEvents(ids)); err != nil {
		if rollback := service.HardDeleteProductTags(ctx, ids); rollback != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", CreateProductTagsStepID, rollback))
		}
		return nil, err
	}
	return items, nil
}

func UpdateProductTag(ctx context.Context, service domain.Repository, events EventPublisher, id string, fields map[string]any) (models.ProductTag, error) {
	previous, err := service.GetProductTag(ctx, id)
	if err != nil {
		return models.ProductTag{}, err
	}
	updated, err := service.UpdateProductTag(ctx, id, fields)
	if err != nil {
		return models.ProductTag{}, err
	}
	if err = publishCatalogEvents(ctx, events, "product-tag.updated", []map[string]any{{"id": id}}); err != nil {
		if _, rollback := service.UpdateProductTag(ctx, id, catalogSnapshot(previous)); rollback != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", UpdateProductTagsStepID, rollback))
		}
		return models.ProductTag{}, err
	}
	return updated, nil
}

func DeleteProductTags(ctx context.Context, service domain.Repository, events EventPublisher, ids []string) error {
	if err := service.SoftDeleteProductTags(ctx, ids); err != nil {
		return err
	}
	if err := publishCatalogEvents(ctx, events, "product-tag.deleted", idsToEvents(ids)); err != nil {
		if rollback := service.RestoreProductTags(ctx, ids); rollback != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", DeleteProductTagsStepID, rollback))
		}
		return err
	}
	return nil
}

func publishCatalogEvents(ctx context.Context, events EventPublisher, name string, rows []map[string]any) error {
	if events == nil {
		return errors.New("product catalog event publisher is required")
	}
	return events.Publish(ctx, name, rows)
}

func productTypeIDValues(items []models.ProductType) []string {
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, item.ID)
	}
	return rows
}
func productTagIDValues(items []models.ProductTag) []string {
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, item.ID)
	}
	return rows
}
func idsToEvents(ids []string) []map[string]any {
	rows := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, map[string]any{"id": id})
	}
	return rows
}
func catalogSnapshot[T any](item T) map[string]any {
	encoded, _ := json.Marshal(item)
	fields := map[string]any{}
	_ = json.Unmarshal(encoded, &fields)
	delete(fields, "id")
	delete(fields, "created_at")
	delete(fields, "updated_at")
	delete(fields, "deleted_at")
	return fields
}
