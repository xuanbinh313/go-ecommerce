package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	linkdomain "github.com/example/medusa-go-clone/internal/modules/links/domain"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	productdomain "github.com/example/medusa-go-clone/internal/modules/product/domain"
	productmodels "github.com/example/medusa-go-clone/internal/modules/product/domain/models"
)

const UpdateProductsWorkflowID = "update-products"
const UpdateProductsStepID = "update-products"
const ProductUpdatedEvent = "product.updated"

// UpdateProduct implements the Admin selector form of Medusa's update-products
// flow. Scalar product data is updated first, selected remote links are
// replaced, and the updated event is emitted last.
func UpdateProduct(ctx context.Context, products productdomain.Repository, links linkdomain.Repository, events EventPublisher, id string, input map[string]any) error {
	if events == nil {
		return errors.New("product event publisher is required")
	}
	previous, err := products.GetProduct(ctx, id)
	if err != nil {
		return err
	}
	fields := make(map[string]any, len(input))
	for key, value := range input {
		if key == "additional_data" {
			continue
		}
		if key == "sales_channels" || key == "shipping_profile_id" {
			continue
		}
		if key == "variants" || key == "images" || key == "categories" || key == "tags" || key == "option_ids" {
			return fmt.Errorf("product update field %q requires its Medusa workflow step", key)
		}
		fields[key] = value
	}
	if _, err := products.UpdateProduct(ctx, id, fields); err != nil {
		return err
	}
	type linkChange struct {
		name    string
		old     []linkmodels.Link
		created []map[string]any
	}
	changes := []linkChange{}
	rollback := func(cause error) error {
		for i := len(changes) - 1; i >= 0; i-- {
			change := changes[i]
			definition, ok := productLinkDefinition(change.name)
			if !ok {
				cause = errors.Join(cause, fmt.Errorf("missing %s link definition for compensation", change.name))
				continue
			}
			if len(change.created) > 0 {
				if _, e := links.Dismiss(ctx, definition, change.created); e != nil {
					cause = errors.Join(cause, fmt.Errorf("dismiss new %s links: %w", change.name, e))
				}
			}
			for _, old := range change.old {
				if _, e := links.Restore(ctx, definition, []map[string]any{old.Fields}); e != nil {
					cause = errors.Join(cause, fmt.Errorf("restore previous %s link: %w", change.name, e))
				}
			}
		}
		if len(fields) > 0 {
			if _, e := products.UpdateProduct(ctx, id, productSnapshot(previous)); e != nil {
				cause = errors.Join(cause, fmt.Errorf("restore product fields: %w", e))
			}
		}
		return cause
	}
	if raw, exists := input["sales_channels"]; exists {
		created, e := replaceProductLinks(ctx, links, "ProductSalesChannel", "product_id", id, raw)
		if e != nil {
			return rollback(e)
		}
		changes = append(changes, created)
	}
	if raw, exists := input["shipping_profile_id"]; exists {
		created, e := replaceProductLinks(ctx, links, "ProductShippingProfile", "product_id", id, raw)
		if e != nil {
			return rollback(e)
		}
		changes = append(changes, created)
	}
	if err := events.Publish(ctx, ProductUpdatedEvent, []map[string]any{{"id": id}}); err != nil {
		return rollback(fmt.Errorf("emit %s: %w", ProductUpdatedEvent, err))
	}
	return nil
}

func replaceProductLinks(ctx context.Context, links linkdomain.Repository, name, productField string, id string, raw any) (struct {
	name    string
	old     []linkmodels.Link
	created []map[string]any
}, error) {
	result := struct {
		name    string
		old     []linkmodels.Link
		created []map[string]any
	}{name: name}
	definition, ok := productLinkDefinition(name)
	if !ok {
		return result, fmt.Errorf("missing %s link definition", name)
	}
	current, err := links.List(ctx, definition, map[string]any{productField: id})
	if err != nil {
		return result, err
	}
	result.old = current
	if len(current) > 0 {
		if _, err = links.Dismiss(ctx, definition, []map[string]any{{productField: id}}); err != nil {
			return result, err
		}
	}
	requested, err := productLinkInput(name, productField, id, raw)
	if err != nil {
		for _, old := range current {
			_, _ = links.Restore(ctx, definition, []map[string]any{old.Fields})
		}
		return result, err
	}
	if len(requested) > 0 {
		if _, err = links.Create(ctx, definition, requested); err != nil {
			for _, old := range current {
				_, _ = links.Restore(ctx, definition, []map[string]any{old.Fields})
			}
			return result, err
		}
		result.created = requested
	}
	return result, nil
}

func productLinkInput(name, productField, id string, value any) ([]map[string]any, error) {
	if name == "ProductSalesChannel" {
		items, ok := value.([]any)
		if !ok {
			return nil, errors.New("sales_channels must be an array")
		}
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			row, ok := item.(map[string]any)
			if !ok {
				return nil, errors.New("sales_channels entries must be objects")
			}
			channelID, ok := row["id"].(string)
			if !ok || channelID == "" {
				return nil, errors.New("sales_channels entries require id")
			}
			out = append(out, map[string]any{productField: id, "sales_channel_id": channelID})
		}
		return out, nil
	}
	if value == nil {
		return nil, nil
	}
	profileID, ok := value.(string)
	if !ok || profileID == "" {
		return nil, errors.New("shipping_profile_id must be a string or null")
	}
	return []map[string]any{{productField: id, "shipping_profile_id": profileID}}, nil
}

func productSnapshot(product productmodels.Product) map[string]any {
	var metadata any
	if len(product.Metadata) > 0 {
		_ = json.Unmarshal(product.Metadata, &metadata)
	}
	return map[string]any{
		"title": product.Title, "handle": product.Handle, "subtitle": product.Subtitle,
		"description": product.Description, "is_giftcard": product.IsGiftcard, "status": product.Status,
		"thumbnail": product.Thumbnail, "weight": product.Weight, "length": product.Length,
		"height": product.Height, "width": product.Width, "origin_country": product.OriginCountry,
		"hs_code": product.HSCode, "mid_code": product.MIDCode, "material": product.Material,
		"type_id": product.TypeID, "collection_id": product.CollectionID,
		"discountable": product.Discountable, "external_id": product.ExternalID, "metadata": metadata,
	}
}
