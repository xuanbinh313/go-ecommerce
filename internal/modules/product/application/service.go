package application

import (
	"context"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/product/domain"
	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
	"regexp"
	"strings"
)

type Service struct{ repository domain.Repository }

func NewService(repository domain.Repository) *Service { return &Service{repository: repository} }

func (s *Service) ListProductCollections(ctx context.Context, f models.ProductCollectionFilter, limit, offset int) ([]models.ProductCollection, int, error) {
	return s.repository.ListProductCollections(ctx, f, limit, offset)
}
func (s *Service) GetProductCollection(ctx context.Context, id string) (models.ProductCollection, error) {
	return s.repository.GetProductCollection(ctx, id)
}
func (s *Service) CreateProductCollections(ctx context.Context, in []models.ProductCollectionInput) ([]models.ProductCollection, error) {
	for i := range in {
		if in[i].Handle == nil || *in[i].Handle == "" {
			h := handle(in[i].Title)
			in[i].Handle = &h
		}
	}
	return s.repository.CreateProductCollections(ctx, in)
}
func (s *Service) UpdateProductCollection(ctx context.Context, id string, fields map[string]any) (models.ProductCollection, error) {
	allowed := map[string]bool{"title": true, "handle": true, "external_id": true, "metadata": true, "product_ids": true}
	for k := range fields {
		if !allowed[k] {
			return models.ProductCollection{}, fmt.Errorf("unsupported product collection field %q", k)
		}
	}
	return s.repository.UpdateProductCollection(ctx, id, fields)
}
func (s *Service) SoftDeleteProductCollections(ctx context.Context, ids []string) error {
	return s.repository.SoftDeleteProductCollections(ctx, ids)
}
func (s *Service) RestoreProductCollections(ctx context.Context, ids []string) error {
	return s.repository.RestoreProductCollections(ctx, ids)
}
func (s *Service) HardDeleteProductCollections(ctx context.Context, ids []string) error {
	return s.repository.HardDeleteProductCollections(ctx, ids)
}
func (s *Service) LinkProductsToCollection(ctx context.Context, id string, add, remove []string) error {
	return s.repository.LinkProductsToCollection(ctx, id, add, remove)
}
func (s *Service) ListProductIDsForCollection(ctx context.Context, id string) ([]string, error) {
	return s.repository.ListProductIDsForCollection(ctx, id)
}
func (s *Service) SetProductsOnCollection(ctx context.Context, id string, ids []string) error {
	return s.repository.SetProductsOnCollection(ctx, id, ids)
}

func (s *Service) ListProductCategories(ctx context.Context, ids []string, limit, offset int) ([]models.ProductCategory, int, error) {
	return s.repository.ListProductCategories(ctx, ids, limit, offset)
}
func (s *Service) ListStoreProductCategories(ctx context.Context, filter models.StoreProductCategoryFilter, limit, offset int) ([]models.ProductCategory, int, error) {
	return s.repository.ListStoreProductCategories(ctx, filter, limit, offset)
}
func (s *Service) ListProductIDsForCategory(ctx context.Context, id string) ([]string, error) {
	return s.repository.ListProductIDsForCategory(ctx, id)
}
func (s *Service) GetProductCategory(ctx context.Context, id string) (models.ProductCategory, error) {
	return s.repository.GetProductCategory(ctx, id)
}
func (s *Service) CreateProductCategories(ctx context.Context, inputs []models.ProductCategoryInput) ([]models.ProductCategory, error) {
	return s.repository.CreateProductCategories(ctx, inputs)
}
func (s *Service) UpdateProductCategory(ctx context.Context, id string, fields map[string]any) (models.ProductCategory, error) {
	allowed := map[string]bool{"name": true, "description": true, "handle": true, "is_active": true, "is_internal": true, "rank": true, "external_id": true, "metadata": true, "parent_category_id": true}
	for key := range fields {
		if !allowed[key] {
			return models.ProductCategory{}, fmt.Errorf("unsupported product category field %q", key)
		}
	}
	return s.repository.UpdateProductCategory(ctx, id, fields)
}
func (s *Service) SoftDeleteProductCategories(ctx context.Context, ids []string) error {
	return s.repository.SoftDeleteProductCategories(ctx, ids)
}
func (s *Service) RestoreProductCategories(ctx context.Context, ids []string) error {
	return s.repository.RestoreProductCategories(ctx, ids)
}
func (s *Service) LinkProductsToCategory(ctx context.Context, id string, add, remove []string) error {
	return s.repository.LinkProductsToCategory(ctx, id, add, remove)
}
func (s *Service) CreateProducts(ctx context.Context, inputs []models.ProductInput) ([]models.Product, error) {
	for i := range inputs {
		if inputs[i].Handle == nil || *inputs[i].Handle == "" {
			h := handle(inputs[i].Title)
			inputs[i].Handle = &h
		}
		if inputs[i].Status == nil {
			v := "draft"
			inputs[i].Status = &v
		}
		if inputs[i].IsGiftcard == nil {
			v := false
			inputs[i].IsGiftcard = &v
		}
		if inputs[i].Discountable == nil {
			v := true
			inputs[i].Discountable = &v
		}
	}
	return s.repository.CreateProducts(ctx, inputs)
}
func (s *Service) CreateVariants(ctx context.Context, inputs []models.ProductVariantInput) ([]models.ProductVariant, error) {
	for i := range inputs {
		if inputs[i].AllowBackorder == nil {
			v := false
			inputs[i].AllowBackorder = &v
		}
		if inputs[i].ManageInventory == nil {
			v := true
			inputs[i].ManageInventory = &v
		}
	}
	return s.repository.CreateVariants(ctx, inputs)
}
func (s *Service) ListProductVariants(ctx context.Context, productID string, filter models.ProductVariantFilter, limit, offset int) ([]models.ProductVariant, int, error) {
	return s.repository.ListProductVariants(ctx, productID, filter, limit, offset)
}
func (s *Service) GetProduct(ctx context.Context, id string) (models.Product, error) {
	return s.repository.GetProduct(ctx, id)
}
func (s *Service) ListProductOptions(ctx context.Context, filter models.ProductOptionFilter, limit, offset int) ([]models.ProductOption, int, error) {
	return s.repository.ListProductOptions(ctx, filter, limit, offset)
}
func (s *Service) GetProductOption(ctx context.Context, id string) (models.ProductOption, error) {
	return s.repository.GetProductOption(ctx, id)
}
func (s *Service) CreateProductOptions(ctx context.Context, inputs []models.CreateProductOptionInput) ([]models.ProductOption, error) {
	return s.repository.CreateProductOptions(ctx, inputs)
}
func (s *Service) UpdateProductOption(ctx context.Context, id string, fields map[string]any) (models.ProductOption, error) {
	allowed := map[string]bool{"title": true, "is_exclusive": true, "metadata": true, "values": true, "ranks": true}
	for key := range fields {
		if !allowed[key] {
			return models.ProductOption{}, fmt.Errorf("%w: unsupported product option field %q", domain.ErrInvalidData, key)
		}
	}
	return s.repository.UpdateProductOption(ctx, id, fields)
}
func (s *Service) RestoreProductOptionSnapshot(ctx context.Context, previous models.ProductOption) error {
	return s.repository.RestoreProductOptionSnapshot(ctx, previous)
}
func (s *Service) SoftDeleteProductOptions(ctx context.Context, ids []string) error {
	return s.repository.SoftDeleteProductOptions(ctx, ids)
}
func (s *Service) RestoreProductOptions(ctx context.Context, ids []string) error {
	return s.repository.RestoreProductOptions(ctx, ids)
}
func (s *Service) HardDeleteProductOptions(ctx context.Context, ids []string) error {
	return s.repository.HardDeleteProductOptions(ctx, ids)
}
func (s *Service) UpdateProduct(ctx context.Context, id string, fields map[string]any) (models.Product, error) {
	allowed := map[string]bool{"title": true, "handle": true, "subtitle": true, "description": true, "is_giftcard": true, "status": true, "thumbnail": true, "weight": true, "length": true, "height": true, "width": true, "origin_country": true, "hs_code": true, "mid_code": true, "material": true, "type_id": true, "collection_id": true, "discountable": true, "external_id": true, "metadata": true}
	for field := range fields {
		if !allowed[field] {
			return models.Product{}, fmt.Errorf("product update field %q requires its Medusa workflow step", field)
		}
	}
	if status, ok := fields["status"].(string); ok {
		switch status {
		case "draft", "proposed", "published", "rejected":
		default:
			return models.Product{}, fmt.Errorf("invalid product status %q", status)
		}
	}
	return s.repository.UpdateProduct(ctx, id, fields)
}
func (s *Service) ListProducts(ctx context.Context, ids []string) ([]models.Product, error) {
	return s.repository.ListProducts(ctx, ids)
}
func (s *Service) ListAdminProducts(ctx context.Context, ids []string, isGiftcard *bool, limit, offset int) ([]models.Product, int, error) {
	return s.repository.ListAdminProducts(ctx, ids, isGiftcard, limit, offset)
}
func (s *Service) ListProductTypes(ctx context.Context, ids []string, limit, offset int) ([]models.ProductType, int, error) {
	return s.repository.ListProductTypes(ctx, ids, limit, offset)
}
func (s *Service) ListStoreProductTypes(ctx context.Context, filter models.ProductCatalogFilter, limit, offset int) ([]models.ProductType, int, error) {
	return s.repository.ListStoreProductTypes(ctx, filter, limit, offset)
}
func (s *Service) ListProductIDsForType(ctx context.Context, id string) ([]string, error) {
	return s.repository.ListProductIDsForType(ctx, id)
}
func (s *Service) GetProductType(ctx context.Context, id string) (models.ProductType, error) {
	return s.repository.GetProductType(ctx, id)
}
func (s *Service) CreateProductTypes(ctx context.Context, inputs []models.ProductTypeInput) ([]models.ProductType, error) {
	for _, input := range inputs {
		if strings.TrimSpace(input.Value) == "" {
			return nil, fmt.Errorf("product type value is required")
		}
	}
	return s.repository.CreateProductTypes(ctx, inputs)
}
func (s *Service) UpdateProductType(ctx context.Context, id string, fields map[string]any) (models.ProductType, error) {
	if err := validateCatalogUpdate(fields); err != nil {
		return models.ProductType{}, err
	}
	return s.repository.UpdateProductType(ctx, id, fields)
}
func (s *Service) SoftDeleteProductTypes(ctx context.Context, ids []string) error {
	return s.repository.SoftDeleteProductTypes(ctx, ids)
}
func (s *Service) DetachProductTypeReferences(ctx context.Context, ids []string) ([]models.ProductTypeProductReference, error) {
	return s.repository.DetachProductTypeReferences(ctx, ids)
}
func (s *Service) RestoreProductTypeReferences(ctx context.Context, refs []models.ProductTypeProductReference) error {
	return s.repository.RestoreProductTypeReferences(ctx, refs)
}
func (s *Service) RestoreProductTypes(ctx context.Context, ids []string) error {
	return s.repository.RestoreProductTypes(ctx, ids)
}
func (s *Service) HardDeleteProductTypes(ctx context.Context, ids []string) error {
	return s.repository.HardDeleteProductTypes(ctx, ids)
}
func (s *Service) ListProductTags(ctx context.Context, ids []string, limit, offset int) ([]models.ProductTag, int, error) {
	return s.repository.ListProductTags(ctx, ids, limit, offset)
}
func (s *Service) ListStoreProductTags(ctx context.Context, filter models.ProductCatalogFilter, limit, offset int) ([]models.ProductTag, int, error) {
	return s.repository.ListStoreProductTags(ctx, filter, limit, offset)
}
func (s *Service) ListProductIDsForTag(ctx context.Context, id string) ([]string, error) {
	return s.repository.ListProductIDsForTag(ctx, id)
}
func (s *Service) GetProductTag(ctx context.Context, id string) (models.ProductTag, error) {
	return s.repository.GetProductTag(ctx, id)
}
func (s *Service) CreateProductTags(ctx context.Context, inputs []models.ProductTagInput) ([]models.ProductTag, error) {
	for _, input := range inputs {
		if strings.TrimSpace(input.Value) == "" {
			return nil, fmt.Errorf("product tag value is required")
		}
	}
	return s.repository.CreateProductTags(ctx, inputs)
}
func (s *Service) UpdateProductTag(ctx context.Context, id string, fields map[string]any) (models.ProductTag, error) {
	if err := validateCatalogUpdate(fields); err != nil {
		return models.ProductTag{}, err
	}
	return s.repository.UpdateProductTag(ctx, id, fields)
}
func (s *Service) SoftDeleteProductTags(ctx context.Context, ids []string) error {
	return s.repository.SoftDeleteProductTags(ctx, ids)
}
func (s *Service) RestoreProductTags(ctx context.Context, ids []string) error {
	return s.repository.RestoreProductTags(ctx, ids)
}
func (s *Service) HardDeleteProductTags(ctx context.Context, ids []string) error {
	return s.repository.HardDeleteProductTags(ctx, ids)
}
func (s *Service) DeleteProducts(ctx context.Context, ids []string) error {
	return s.repository.DeleteProducts(ctx, ids)
}
func (s *Service) RestoreProducts(ctx context.Context, ids []string) error {
	return s.repository.RestoreProducts(ctx, ids)
}
func (s *Service) DeleteVariants(ctx context.Context, ids []string) error {
	return s.repository.DeleteVariants(ctx, ids)
}

var nonHandle = regexp.MustCompile(`[^a-z0-9]+`)

func validateCatalogUpdate(fields map[string]any) error {
	for key, value := range fields {
		if key != "value" && key != "external_id" && key != "metadata" {
			return fmt.Errorf("unsupported product catalog field %q", key)
		}
		if key == "value" {
			if text, ok := value.(string); !ok || strings.TrimSpace(text) == "" {
				return fmt.Errorf("product catalog value must be a non-empty string")
			}
		}
	}
	return nil
}

func handle(value string) string {
	return strings.Trim(nonHandle.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "-"), "-")
}
