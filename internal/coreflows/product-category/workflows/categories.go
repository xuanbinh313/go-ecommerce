package workflows

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
)

const (
	CreateProductCategoriesWorkflowID     = "create-product-categories"
	UpdateProductCategoriesWorkflowID     = "update-product-categories"
	DeleteProductCategoriesWorkflowID     = "delete-product-categories"
	BatchLinkProductsToCategoryWorkflowID = "batch-link-products-to-category"
	CreateProductCategoriesStepID         = "create-product-categories"
	UpdateProductCategoriesStepID         = "update-product-categories"
	DeleteProductCategoriesStepID         = "delete-product-categories"
	BatchLinkProductsToCategoryStepID     = "batch-link-products-in-category"
)

type ProductCategoryService interface {
	GetProductCategory(context.Context, string) (models.ProductCategory, error)
	ListProductCategories(context.Context, []string, int, int) ([]models.ProductCategory, int, error)
	CreateProductCategories(context.Context, []models.ProductCategoryInput) ([]models.ProductCategory, error)
	UpdateProductCategory(context.Context, string, map[string]any) (models.ProductCategory, error)
	SoftDeleteProductCategories(context.Context, []string) error
	RestoreProductCategories(context.Context, []string) error
	LinkProductsToCategory(context.Context, string, []string, []string) error
}

// CreateProductCategories mirrors Medusa's create step and delete compensation.
func CreateProductCategories(ctx context.Context, s ProductCategoryService, in []models.ProductCategoryInput) ([]models.ProductCategory, error) {
	created, err := s.CreateProductCategories(ctx, in)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// UpdateProductCategory restores the prior category snapshot if the update step fails.
func UpdateProductCategory(ctx context.Context, s ProductCategoryService, id string, fields map[string]any) (models.ProductCategory, error) {
	previous, err := s.GetProductCategory(ctx, id)
	if err != nil {
		return models.ProductCategory{}, err
	}
	updated, err := s.UpdateProductCategory(ctx, id, fields)
	if err != nil {
		_, _ = s.UpdateProductCategory(ctx, id, map[string]any{"name": previous.Name, "description": previous.Description, "handle": previous.Handle, "is_active": previous.IsActive, "is_internal": previous.IsInternal, "rank": previous.Rank, "external_id": previous.ExternalID, "metadata": previous.Metadata, "parent_category_id": previous.ParentCategoryID})
		return models.ProductCategory{}, err
	}
	return updated, nil
}

func DeleteProductCategories(ctx context.Context, s ProductCategoryService, ids []string) error {
	return s.SoftDeleteProductCategories(ctx, ids)
}
func BatchLinkProductsToCategory(ctx context.Context, s ProductCategoryService, id string, add, remove []string) error {
	return s.LinkProductsToCategory(ctx, id, add, remove)
}
