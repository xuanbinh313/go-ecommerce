package workflows

import (
	"context"
	"errors"
	"fmt"
	"strings"

	linkdomain "github.com/example/medusa-go-clone/internal/modules/links/domain"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	productdomain "github.com/example/medusa-go-clone/internal/modules/product/domain"
	productmodels "github.com/example/medusa-go-clone/internal/modules/product/domain/models"
)

const CreateProductsWorkflowID = "create-products"
const ValidateProductInputStepID = "validate-product-input"
const CreateProductsStepID = "create-products"
const CreateRemoteLinksStepID = "create-remote-links"
const EmitEventStepID = "emit-event-step"
const ProductCreatedEvent = "product.created"

type VariantWorkflow interface {
	CreateProductVariants(context.Context, []productmodels.ProductVariantInput) ([]productmodels.ProductVariant, error)
}
type EventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}

// CreateProducts preserves the Medusa create-products step order. Inventory,
// pricing, and variant follow-up belong to the nested variant workflow port.
func CreateProducts(ctx context.Context, products productdomain.Repository, links linkdomain.Repository, variants VariantWorkflow, events EventPublisher, input productmodels.CreateProductsInput) ([]productmodels.Product, error) {
	if events == nil {
		return nil, errors.New("product event publisher is required")
	}
	productInputs := make([]productmodels.ProductInput, len(input.Products))
	missingOptions := []string{}
	for i, product := range input.Products {
		if len(product.Options) == 0 {
			missingOptions = append(missingOptions, product.Title)
		}
		if len(product.Variants) > 0 && variants == nil {
			return nil, errors.New("product variants require the create-product-variants workflow")
		}
		product.Variants = nil
		product.SalesChannels = nil
		product.ShippingProfileID = nil
		productInputs[i] = product
	}
	if len(missingOptions) > 0 {
		return nil, fmt.Errorf("Product options are not provided for: [%s].", strings.Join(missingOptions, ", "))
	}
	created, err := products.CreateProducts(ctx, productInputs)
	if err != nil {
		return nil, err
	}
	productIDs := make([]string, len(created))
	for i, p := range created {
		productIDs[i] = p.ID
	}
	linkRequests := map[string][]map[string]any{"ProductSalesChannel": {}, "ProductShippingProfile": {}}
	for i, in := range input.Products {
		for _, channel := range in.SalesChannels {
			linkRequests["ProductSalesChannel"] = append(linkRequests["ProductSalesChannel"], map[string]any{"product_id": created[i].ID, "sales_channel_id": channel.ID})
		}
		if in.ShippingProfileID != nil && *in.ShippingProfileID != "" {
			linkRequests["ProductShippingProfile"] = append(linkRequests["ProductShippingProfile"], map[string]any{"product_id": created[i].ID, "shipping_profile_id": *in.ShippingProfileID})
		}
	}
	createdLinks := []createdLink{}
	compensate := func(cause error) error {
		for i := len(createdLinks) - 1; i >= 0; i-- {
			_, e := links.Dismiss(ctx, createdLinks[i].definition, createdLinks[i].fields)
			if e != nil {
				cause = errors.Join(cause, fmt.Errorf("dismiss product link: %w", e))
			}
		}
		if e := products.DeleteProducts(ctx, productIDs); e != nil {
			cause = errors.Join(cause, fmt.Errorf("delete created products: %w", e))
		}
		return cause
	}
	for _, name := range []string{"ProductSalesChannel", "ProductShippingProfile"} {
		fields := linkRequests[name]
		if len(fields) == 0 {
			continue
		}
		definition, ok := findDefinition(name)
		if !ok {
			return nil, compensate(fmt.Errorf("missing %s link definition", name))
		}
		if _, err = links.Create(ctx, definition, fields); err != nil {
			return nil, compensate(err)
		}
		createdLinks = append(createdLinks, createdLink{definition: definition, fields: fields})
	}
	variantInputs := []productmodels.ProductVariantInput{}
	for i, in := range input.Products {
		for _, variant := range in.Variants {
			variant.ProductID = created[i].ID
			variantInputs = append(variantInputs, variant)
		}
	}
	if len(variantInputs) > 0 {
		createdVariants, e := variants.CreateProductVariants(ctx, variantInputs)
		if e != nil {
			return nil, compensate(e)
		}
		byProduct := map[string][]productmodels.ProductVariant{}
		for _, variant := range createdVariants {
			if variant.ProductID != nil {
				byProduct[*variant.ProductID] = append(byProduct[*variant.ProductID], variant)
			}
		}
		for i := range created {
			created[i].Variants = byProduct[created[i].ID]
		}
	}
	eventData := make([]map[string]any, len(created))
	for i, p := range created {
		eventData[i] = map[string]any{"id": p.ID}
	}
	if err = events.Publish(ctx, ProductCreatedEvent, eventData); err != nil {
		return nil, compensate(fmt.Errorf("emit %s: %w", ProductCreatedEvent, err))
	}
	return created, nil
}

type createdLink struct {
	definition linkmodels.Definition
	fields     []map[string]any
}

func findDefinition(name string) (linkmodels.Definition, bool) {
	for _, d := range linkmodels.Definitions {
		if d.Name == name {
			return d, true
		}
	}
	return linkmodels.Definition{}, false
}
