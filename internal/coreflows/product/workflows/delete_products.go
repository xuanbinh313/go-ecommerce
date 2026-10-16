package workflows

import (
	"context"
	"errors"
	"fmt"

	linkdomain "github.com/example/medusa-go-clone/internal/modules/links/domain"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	productdomain "github.com/example/medusa-go-clone/internal/modules/product/domain"
)

const DeleteProductsWorkflowID = "delete-products"
const QueryProductsWithOptionsStepID = "query-products-with-options-step"
const RemoveProductVariantLinkStepID = "remove-product-variant-link-step"
const DeleteProductsStepID = "delete-products"
const ProductDeletedEvent = "product.deleted"

// DeleteProducts follows the source flow's relation-removal, product deletion,
// then event order. Errors after links are dismissed restore those links; event
// errors also restore the soft-deleted products.
func DeleteProducts(ctx context.Context, products productdomain.Repository, links linkdomain.Repository, events EventPublisher, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if events == nil {
		return errors.New("product event publisher is required")
	}
	type dismissed struct {
		definition linkmodels.Definition
		fields     []map[string]any
	}
	removed := []dismissed{}
	restoreLinks := func(cause error) error {
		for i := len(removed) - 1; i >= 0; i-- {
			if _, err := links.Restore(ctx, removed[i].definition, removed[i].fields); err != nil {
				cause = errors.Join(cause, fmt.Errorf("restore %s links: %w", removed[i].definition.Name, err))
			}
		}
		return cause
	}
	variantIDs := []string{}
	for _, id := range ids {
		product, err := products.GetProduct(ctx, id)
		if err != nil {
			return fmt.Errorf("query product %s: %w", id, err)
		}
		for _, variant := range product.Variants {
			variantIDs = append(variantIDs, variant.ID)
		}
	}
	linkFilters := []struct {
		name   string
		fields []map[string]any
	}{
		{name: "ProductSalesChannel", fields: productLinkFields("product_id", ids)},
		{name: "ProductShippingProfile", fields: productLinkFields("product_id", ids)},
		{name: "ProductVariantInventoryItem", fields: productLinkFields("variant_id", variantIDs)},
		{name: "ProductVariantPriceSet", fields: productLinkFields("variant_id", variantIDs)},
	}
	for _, candidate := range linkFilters {
		if len(candidate.fields) == 0 {
			continue
		}
		definition, ok := productLinkDefinition(candidate.name)
		if !ok {
			return restoreLinks(fmt.Errorf("missing %s link definition", candidate.name))
		}
		if _, err := links.Dismiss(ctx, definition, candidate.fields); err != nil {
			return restoreLinks(err)
		}
		removed = append(removed, dismissed{definition: definition, fields: candidate.fields})
	}
	if err := products.DeleteProducts(ctx, ids); err != nil {
		return restoreLinks(err)
	}
	data := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		data = append(data, map[string]any{"id": id})
	}
	if err := events.Publish(ctx, ProductDeletedEvent, data); err != nil {
		if restoreErr := products.RestoreProducts(ctx, ids); restoreErr != nil {
			err = errors.Join(err, fmt.Errorf("restore products after event failure: %w", restoreErr))
		}
		return restoreLinks(fmt.Errorf("emit %s: %w", ProductDeletedEvent, err))
	}
	return nil
}

func productLinkFields(field string, ids []string) []map[string]any {
	fields := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		fields = append(fields, map[string]any{field: id})
	}
	return fields
}

func productLinkDefinition(name string) (linkmodels.Definition, bool) {
	for _, definition := range linkmodels.Definitions {
		if definition.Name == name {
			return definition, true
		}
	}
	return linkmodels.Definition{}, false
}
