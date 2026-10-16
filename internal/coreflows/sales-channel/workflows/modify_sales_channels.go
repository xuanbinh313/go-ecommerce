package workflows

import (
	"context"
	"errors"
	"fmt"

	linkdomain "github.com/example/medusa-go-clone/internal/modules/links/domain"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	salesdomain "github.com/example/medusa-go-clone/internal/modules/sales-channel/domain"
	salesmodels "github.com/example/medusa-go-clone/internal/modules/sales-channel/domain/models"
	storemodels "github.com/example/medusa-go-clone/internal/modules/store/domain/models"
)

const (
	UpdateSalesChannelsWorkflowID         = "update-sales-channels"
	UpdateSalesChannelsStepID             = "update-sales-channels"
	DeleteSalesChannelsWorkflowID         = "delete-sales-channels"
	CanDeleteSalesChannelsStepID          = "can-delete-sales-channels-or-throw-step"
	DeleteSalesChannelsStepID             = "delete-sales-channels"
	RemoveSalesChannelLinksStepID         = "remove-remote-link-step"
	SalesChannelUpdatedEvent              = "sales-channel.updated"
	SalesChannelDeletedEvent              = "sales-channel.deleted"
	LinkProductsToSalesChannelWorkflowID  = "link-products-to-sales-channel"
	AssociateProductsWithChannelsStepID   = "associate-products-with-channels"
	DetachProductsFromSalesChannelsStepID = "detach-products-from-sales-channels-step"
	ProductUpdatedEvent                   = "product.updated"
)

type SalesChannelEventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}
type StoreReader interface {
	List(context.Context) ([]storemodels.Store, error)
}

func UpdateSalesChannel(ctx context.Context, service salesdomain.Repository, events SalesChannelEventPublisher, id string, input salesmodels.Input) (salesmodels.SalesChannel, error) {
	previous, err := service.Get(ctx, id)
	if err != nil {
		return salesmodels.SalesChannel{}, err
	}
	updated, err := service.Update(ctx, id, input)
	if err != nil {
		return salesmodels.SalesChannel{}, err
	}
	if events == nil {
		err = errors.New("sales channel event publisher is required")
	} else {
		err = events.Publish(ctx, SalesChannelUpdatedEvent, []map[string]any{{"id": id}})
	}
	if err != nil {
		name, description, disabled := previous.Name, previous.Description, previous.IsDisabled
		rollback := salesmodels.Input{Name: &name, Description: description, DescriptionSet: true, IsDisabled: &disabled, Metadata: previous.Metadata, MetadataSet: true}
		if _, restoreErr := service.Update(ctx, id, rollback); restoreErr != nil {
			err = errors.Join(err, fmt.Errorf("compensate %s: %w", UpdateSalesChannelsStepID, restoreErr))
		}
		return salesmodels.SalesChannel{}, err
	}
	return updated, nil
}

func DeleteSalesChannels(ctx context.Context, service salesdomain.Repository, stores StoreReader, links linkdomain.Repository, events SalesChannelEventPublisher, ids []string) error {
	storeRows, err := stores.List(ctx)
	if err != nil {
		return err
	}
	for _, store := range storeRows {
		if store.DefaultSalesChannelID != nil && containsString(ids, *store.DefaultSalesChannelID) {
			return fmt.Errorf("%w: cannot delete default sales channel %s", salesdomain.ErrInvalidData, *store.DefaultSalesChannelID)
		}
	}
	dismissed := []struct {
		definition linkmodels.Definition
		fields     map[string]any
	}{}
	restore := func(cause error) error {
		if e := service.Restore(ctx, ids); e != nil {
			cause = errors.Join(cause, fmt.Errorf("compensate %s: %w", DeleteSalesChannelsStepID, e))
		}
		for i := len(dismissed) - 1; i >= 0; i-- {
			item := dismissed[i]
			if _, e := links.Restore(ctx, item.definition, []map[string]any{item.fields}); e != nil {
				cause = errors.Join(cause, fmt.Errorf("restore %s link: %w", item.definition.Name, e))
			}
		}
		return cause
	}
	for _, id := range ids {
		if err = service.Delete(ctx, id); err != nil {
			return restore(err)
		}
	}
	for _, id := range ids {
		for _, definition := range linkmodels.Definitions {
			if !hasForeignKey(definition, "sales_channel_id") {
				continue
			}
			filter := map[string]any{"sales_channel_id": id}
			current, e := links.List(ctx, definition, filter)
			if e != nil {
				return restore(e)
			}
			if len(current) == 0 {
				continue
			}
			if _, e = links.Dismiss(ctx, definition, []map[string]any{filter}); e != nil {
				return restore(e)
			}
			for _, link := range current {
				dismissed = append(dismissed, struct {
					definition linkmodels.Definition
					fields     map[string]any
				}{definition: definition, fields: link.Fields})
			}
		}
	}
	data := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		data = append(data, map[string]any{"id": id})
	}
	if events == nil {
		return restore(errors.New("sales channel event publisher is required"))
	}
	if err = events.Publish(ctx, SalesChannelDeletedEvent, data); err != nil {
		return restore(fmt.Errorf("emit %s: %w", SalesChannelDeletedEvent, err))
	}
	return nil
}

func hasForeignKey(definition linkmodels.Definition, key string) bool {
	for _, field := range definition.ForeignKeys {
		if field == key {
			return true
		}
	}
	return false
}
func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// LinkProductsToSalesChannel follows Medusa's associate-then-detach workflow
// and restores completed link steps if a later action fails.
func LinkProductsToSalesChannel(ctx context.Context, links linkdomain.Repository, events SalesChannelEventPublisher, channelID string, add, remove []string) error {
	definition, ok := linkDefinition("ProductSalesChannel")
	if !ok {
		return fmt.Errorf("link definition ProductSalesChannel is not registered")
	}
	addFields := make([]map[string]any, 0, len(add))
	removeFields := make([]map[string]any, 0, len(remove))
	for _, id := range add {
		addFields = append(addFields, map[string]any{"sales_channel_id": channelID, "product_id": id})
	}
	for _, id := range remove {
		removeFields = append(removeFields, map[string]any{"sales_channel_id": channelID, "product_id": id})
	}
	if _, err := links.Create(ctx, definition, addFields); err != nil {
		return err
	}
	rollbackAdded := func(cause error) error {
		if len(addFields) > 0 {
			if _, e := links.Dismiss(ctx, definition, addFields); e != nil {
				cause = errors.Join(cause, fmt.Errorf("compensate %s: %w", AssociateProductsWithChannelsStepID, e))
			}
		}
		return cause
	}
	if len(removeFields) > 0 {
		if _, err := links.Dismiss(ctx, definition, removeFields); err != nil {
			return rollbackAdded(err)
		}
	}
	ids := make([]map[string]any, 0, len(add)+len(remove))
	seen := map[string]bool{}
	for _, id := range append(append([]string{}, add...), remove...) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, map[string]any{"id": id})
		}
	}
	var err error
	if events == nil {
		err = errors.New("sales channel product event publisher is required")
	} else {
		err = events.Publish(ctx, ProductUpdatedEvent, ids)
	}
	if err != nil {
		if len(removeFields) > 0 {
			if _, e := links.Create(ctx, definition, removeFields); e != nil {
				err = errors.Join(err, fmt.Errorf("compensate %s: %w", DetachProductsFromSalesChannelsStepID, e))
			}
		}
		return rollbackAdded(err)
	}
	return nil
}

func linkDefinition(name string) (linkmodels.Definition, bool) {
	for _, definition := range linkmodels.Definitions {
		if definition.Name == name {
			return definition, true
		}
	}
	return linkmodels.Definition{}, false
}
