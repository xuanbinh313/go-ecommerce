package workflows

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/product/domain"
	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
)

const CreateProductOptionsWorkflowID = "create-product-options"
const CreateProductOptionsStepID = "create-product-options"
const UpdateProductOptionsWorkflowID = "update-product-options"
const UpdateProductOptionsStepID = "update-product-options"
const DeleteProductOptionsWorkflowID = "delete-product-options"
const DeleteProductOptionsStepID = "delete-product-options"

func CreateProductOptions(ctx context.Context, service domain.Repository, events EventPublisher, input []models.CreateProductOptionInput) ([]models.ProductOption, error) {
	items, err := service.CreateProductOptions(ctx, input)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	if events == nil {
		err = errors.New("product option event publisher is required")
	} else {
		err = events.Publish(ctx, "product-option.created", idsToEvents(ids))
	}
	if err != nil {
		if rollback := service.HardDeleteProductOptions(ctx, ids); rollback != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", CreateProductOptionsStepID, rollback))
		}
		return nil, err
	}
	return items, nil
}

func UpdateProductOption(ctx context.Context, service domain.Repository, events EventPublisher, id string, input map[string]any) (models.ProductOption, error) {
	previous, err := service.GetProductOption(ctx, id)
	if err != nil {
		return models.ProductOption{}, err
	}
	updated, err := service.UpdateProductOption(ctx, id, input)
	if err != nil {
		return models.ProductOption{}, err
	}
	if events == nil {
		err = errors.New("product option event publisher is required")
	} else {
		err = events.Publish(ctx, "product-option.updated", idsToEvents([]string{id}))
	}
	if err != nil {
		if rollback := service.RestoreProductOptionSnapshot(ctx, previous); rollback != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", UpdateProductOptionsStepID, rollback))
		}
		return models.ProductOption{}, err
	}
	return updated, nil
}

func DeleteProductOptions(ctx context.Context, service domain.Repository, events EventPublisher, ids []string) error {
	previous := make([]models.ProductOption, 0, len(ids))
	for _, id := range ids {
		item, err := service.GetProductOption(ctx, id)
		if err != nil {
			return err
		}
		previous = append(previous, item)
	}
	if err := service.SoftDeleteProductOptions(ctx, ids); err != nil {
		return err
	}
	restore := func(cause error) error {
		if err := service.RestoreProductOptions(ctx, ids); err != nil {
			cause = errors.Join(cause, fmt.Errorf("compensate %s: %w", DeleteProductOptionsStepID, err))
			return cause
		}
		for _, item := range previous {
			if err := service.RestoreProductOptionSnapshot(ctx, item); err != nil {
				cause = errors.Join(cause, fmt.Errorf("restore option %s values: %w", item.ID, err))
			}
		}
		return cause
	}
	if events == nil {
		err := errors.New("product option event publisher is required")
		return restore(err)
	}
	if err := events.Publish(ctx, "product-option.deleted", idsToEvents(ids)); err != nil {
		return restore(err)
	}
	return nil
}
