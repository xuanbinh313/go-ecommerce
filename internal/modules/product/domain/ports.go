package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
)

type Repository interface {
	CreateProducts(context.Context, []models.ProductInput) ([]models.Product, error)
	CreateVariants(context.Context, []models.ProductVariantInput) ([]models.ProductVariant, error)
	ListProductVariants(context.Context, string, models.ProductVariantFilter, int, int) ([]models.ProductVariant, int, error)
	GetProduct(context.Context, string) (models.Product, error)
	ListProductOptions(context.Context, models.ProductOptionFilter, int, int) ([]models.ProductOption, int, error)
	GetProductOption(context.Context, string) (models.ProductOption, error)
	CreateProductOptions(context.Context, []models.CreateProductOptionInput) ([]models.ProductOption, error)
	UpdateProductOption(context.Context, string, map[string]any) (models.ProductOption, error)
	RestoreProductOptionSnapshot(context.Context, models.ProductOption) error
	SoftDeleteProductOptions(context.Context, []string) error
	RestoreProductOptions(context.Context, []string) error
	HardDeleteProductOptions(context.Context, []string) error
	UpdateProduct(context.Context, string, map[string]any) (models.Product, error)
	ListProducts(context.Context, []string) ([]models.Product, error)
	ListAdminProducts(context.Context, []string, *bool, int, int) ([]models.Product, int, error)
	ListProductTypes(context.Context, []string, int, int) ([]models.ProductType, int, error)
	ListStoreProductTypes(context.Context, models.ProductCatalogFilter, int, int) ([]models.ProductType, int, error)
	GetProductType(context.Context, string) (models.ProductType, error)
	ListProductIDsForType(context.Context, string) ([]string, error)
	CreateProductTypes(context.Context, []models.ProductTypeInput) ([]models.ProductType, error)
	UpdateProductType(context.Context, string, map[string]any) (models.ProductType, error)
	SoftDeleteProductTypes(context.Context, []string) error
	DetachProductTypeReferences(context.Context, []string) ([]models.ProductTypeProductReference, error)
	RestoreProductTypeReferences(context.Context, []models.ProductTypeProductReference) error
	RestoreProductTypes(context.Context, []string) error
	HardDeleteProductTypes(context.Context, []string) error
	ListProductTags(context.Context, []string, int, int) ([]models.ProductTag, int, error)
	ListStoreProductTags(context.Context, models.ProductCatalogFilter, int, int) ([]models.ProductTag, int, error)
	GetProductTag(context.Context, string) (models.ProductTag, error)
	ListProductIDsForTag(context.Context, string) ([]string, error)
	CreateProductTags(context.Context, []models.ProductTagInput) ([]models.ProductTag, error)
	UpdateProductTag(context.Context, string, map[string]any) (models.ProductTag, error)
	SoftDeleteProductTags(context.Context, []string) error
	RestoreProductTags(context.Context, []string) error
	HardDeleteProductTags(context.Context, []string) error
	DeleteProducts(context.Context, []string) error
	RestoreProducts(context.Context, []string) error
	DeleteVariants(context.Context, []string) error
	ListProductCategories(context.Context, []string, int, int) ([]models.ProductCategory, int, error)
	ListStoreProductCategories(context.Context, models.StoreProductCategoryFilter, int, int) ([]models.ProductCategory, int, error)
	GetProductCategory(context.Context, string) (models.ProductCategory, error)
	ListProductIDsForCategory(context.Context, string) ([]string, error)
	CreateProductCategories(context.Context, []models.ProductCategoryInput) ([]models.ProductCategory, error)
	UpdateProductCategory(context.Context, string, map[string]any) (models.ProductCategory, error)
	SoftDeleteProductCategories(context.Context, []string) error
	RestoreProductCategories(context.Context, []string) error
	LinkProductsToCategory(context.Context, string, []string, []string) error
	ListProductCollections(context.Context, models.ProductCollectionFilter, int, int) ([]models.ProductCollection, int, error)
	GetProductCollection(context.Context, string) (models.ProductCollection, error)
	CreateProductCollections(context.Context, []models.ProductCollectionInput) ([]models.ProductCollection, error)
	UpdateProductCollection(context.Context, string, map[string]any) (models.ProductCollection, error)
	SoftDeleteProductCollections(context.Context, []string) error
	RestoreProductCollections(context.Context, []string) error
	HardDeleteProductCollections(context.Context, []string) error
	LinkProductsToCollection(context.Context, string, []string, []string) error
	ListProductIDsForCollection(context.Context, string) ([]string, error)
	SetProductsOnCollection(context.Context, string, []string) error
}
