package workflows

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
)

const (
	CreateCollectionsWorkflowID             = "create-collections"
	UpdateCollectionsWorkflowID             = "update-collections"
	DeleteCollectionsWorkflowID             = "delete-collections"
	BatchLinkProductsToCollectionWorkflowID = "batch-link-products-to-collection"
	CreateCollectionsStepID                 = "create-collections"
	UpdateCollectionsStepID                 = "update-collections"
	DeleteCollectionsStepID                 = "delete-collections"
	BatchLinkProductsToCollectionStepID     = "batch-link-products-to-collection"
)

type ProductCollectionService interface {
	ListProductCollections(context.Context, models.ProductCollectionFilter, int, int) ([]models.ProductCollection, int, error)
	GetProductCollection(context.Context, string) (models.ProductCollection, error)
	CreateProductCollections(context.Context, []models.ProductCollectionInput) ([]models.ProductCollection, error)
	UpdateProductCollection(context.Context, string, map[string]any) (models.ProductCollection, error)
	SoftDeleteProductCollections(context.Context, []string) error
	RestoreProductCollections(context.Context, []string) error
	HardDeleteProductCollections(context.Context, []string) error
	ListProductIDsForCollection(context.Context, string) ([]string, error)
	SetProductsOnCollection(context.Context, string, []string) error
	LinkProductsToCollection(context.Context, string, []string, []string) error
}

func CreateCollections(ctx context.Context, s ProductCollectionService, events EventPublisher, in []models.ProductCollectionInput) ([]models.ProductCollection, error) {
	created, err := s.CreateProductCollections(ctx, in)
	if err != nil {
		return nil, err
	}
	if events != nil {
		if err = events.Publish(ctx, "product-collection.created", collectionEvents(created)); err != nil {
			if rollbackErr := s.HardDeleteProductCollections(ctx, collectionIDs(created)); rollbackErr != nil {
				return nil, errors.Join(err, fmt.Errorf("rollback created collections: %w", rollbackErr))
			}
			return nil, err
		}
	}
	return created, nil
}
func UpdateCollection(ctx context.Context, s ProductCollectionService, events EventPublisher, id string, fields map[string]any) (models.ProductCollection, error) {
	old, err := s.GetProductCollection(ctx, id)
	if err != nil {
		return models.ProductCollection{}, err
	}
	updated, err := s.UpdateProductCollection(ctx, id, fields)
	if err != nil {
		return models.ProductCollection{}, err
	}
	if events != nil {
		if err = events.Publish(ctx, "product-collection.updated", []map[string]any{{"id": id}}); err != nil {
			_, rollbackErr := s.UpdateProductCollection(ctx, id, map[string]any{"title": old.Title, "handle": old.Handle, "external_id": old.ExternalID, "metadata": old.Metadata})
			if rollbackErr != nil {
				return models.ProductCollection{}, errors.Join(err, fmt.Errorf("restore collection: %w", rollbackErr))
			}
			return models.ProductCollection{}, err
		}
	}
	return updated, nil
}
func DeleteCollections(ctx context.Context, s ProductCollectionService, events EventPublisher, ids []string) error {
	if err := s.SoftDeleteProductCollections(ctx, ids); err != nil {
		return err
	}
	if events != nil {
		if err := events.Publish(ctx, "product-collection.deleted", idEvents(ids)); err != nil {
			if restoreErr := s.RestoreProductCollections(ctx, ids); restoreErr != nil {
				return errors.Join(err, fmt.Errorf("restore collections: %w", restoreErr))
			}
			return err
		}
	}
	return nil
}
func BatchLinkProductsToCollection(ctx context.Context, s ProductCollectionService, events EventPublisher, id string, add, remove []string) error {
	if len(add) == 0 && len(remove) == 0 {
		return nil
	}
	previous, err := s.ListProductIDsForCollection(ctx, id)
	if err != nil {
		return err
	}
	if err = s.LinkProductsToCollection(ctx, id, add, remove); err != nil {
		return err
	}
	if events != nil {
		if err = events.Publish(ctx, "product.updated", idEvents(deduplicate(append(append([]string{}, add...), remove...)))); err != nil {
			if restoreErr := s.SetProductsOnCollection(ctx, id, previous); restoreErr != nil {
				return errors.Join(err, fmt.Errorf("restore collection products: %w", restoreErr))
			}
			return err
		}
	}
	return nil
}
func collectionIDs(items []models.ProductCollection) []string {
	ids := make([]string, 0, len(items))
	for _, v := range items {
		ids = append(ids, v.ID)
	}
	return ids
}
func collectionEvents(items []models.ProductCollection) []map[string]any {
	data := make([]map[string]any, 0, len(items))
	for _, v := range items {
		data = append(data, map[string]any{"id": v.ID})
	}
	return data
}
func idEvents(ids []string) []map[string]any {
	data := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		data = append(data, map[string]any{"id": id})
	}
	return data
}
func deduplicate(ids []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
