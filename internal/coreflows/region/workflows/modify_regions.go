package workflows

import (
	"context"
	"errors"
	"fmt"

	linkdomain "github.com/example/medusa-go-clone/internal/modules/links/domain"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	"github.com/example/medusa-go-clone/internal/modules/region/domain/models"
)

type RegionModifier interface {
	Get(context.Context, string) (models.Region, error)
	Update(context.Context, string, models.RegionInput) (models.Region, error)
	SoftDelete(context.Context, []string) error
	RestoreRegions(context.Context, []string) error
}
type RegionTaxPreferenceUpdater interface {
	SetRegionTaxInclusive(context.Context, string, bool) error
}
type RegionUpdateEventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}

// UpdateRegion keeps module update, provider links, tax preference, and event behavior in Medusa workflow order.
func UpdateRegion(ctx context.Context, service RegionModifier, links linkdomain.Repository, payment PaymentProviderValidator, prices RegionTaxPreferenceUpdater, events RegionUpdateEventPublisher, id string, input models.RegionInput) (models.Region, error) {
	previous, err := service.Get(ctx, id)
	if err != nil {
		return models.Region{}, err
	}
	updated, err := service.Update(ctx, id, input)
	if err != nil {
		return models.Region{}, err
	}
	var def linkmodels.Definition
	if input.PaymentProviders != nil {
		if err = payment.ValidatePaymentProviders(ctx, unique(input.PaymentProviders)); err != nil {
			return rollbackRegion(ctx, service, id, previous, err)
		}
		def, err = regionPaymentProviderDefinition()
		if err != nil {
			return rollbackRegion(ctx, service, id, previous, err)
		}
		current, e := links.List(ctx, def, map[string]any{"region_id": id})
		if e != nil {
			return rollbackRegion(ctx, service, id, previous, e)
		}
		wanted := map[string]bool{}
		for _, providerID := range input.PaymentProviders {
			wanted[providerID] = true
		}
		currentIDs := map[string]bool{}
		remove := []map[string]any{}
		for _, row := range current {
			providerID := fmt.Sprint(row.Fields["payment_provider_id"])
			currentIDs[providerID] = true
			if !wanted[providerID] {
				remove = append(remove, row.Fields)
			}
		}
		add := []map[string]any{}
		for providerID := range wanted {
			if !currentIDs[providerID] {
				add = append(add, map[string]any{"region_id": id, "payment_provider_id": providerID})
			}
		}
		removed := []map[string]any{}
		if len(remove) > 0 {
			dismissed, dismissErr := links.Dismiss(ctx, def, remove)
			e = dismissErr
			if e != nil {
				return rollbackRegion(ctx, service, id, previous, e)
			}
			for _, link := range dismissed {
				removed = append(removed, link.Fields)
			}
		}
		if len(add) > 0 {
			_, e = links.Create(ctx, def, add)
			if e != nil {
				if len(removed) > 0 {
					_, e = links.Restore(ctx, def, removed)
					if e != nil {
						return rollbackRegion(ctx, service, id, previous, errors.Join(fmt.Errorf("create provider links: %w", e), e))
					}
				}
				return rollbackRegion(ctx, service, id, previous, e)
			}
		}
	}
	if input.IsTaxInclusive != nil {
		if prices == nil {
			return rollbackRegion(ctx, service, id, previous, errors.New("region price preference updater is required"))
		}
		if err = prices.SetRegionTaxInclusive(ctx, id, *input.IsTaxInclusive); err != nil {
			return rollbackRegion(ctx, service, id, previous, err)
		}
	}
	if events == nil {
		return rollbackRegion(ctx, service, id, previous, errors.New("region event publisher is required"))
	}
	if err = events.Publish(ctx, RegionUpdatedEvent, []map[string]any{{"id": id}}); err != nil {
		return rollbackRegion(ctx, service, id, previous, err)
	}
	return updated, nil
}

// DeleteRegion soft deletes the module record, removes every remote link, then emits region.deleted.
func DeleteRegion(ctx context.Context, service RegionModifier, links linkdomain.Repository, events RegionUpdateEventPublisher, id string) error {
	if _, err := service.Get(ctx, id); err != nil {
		return err
	}
	if err := service.SoftDelete(ctx, []string{id}); err != nil {
		return err
	}
	type removedLink struct {
		definition linkmodels.Definition
		fields     map[string]any
	}
	removed := []removedLink{}
	rollback := func(cause error) error {
		if e := service.RestoreRegions(ctx, []string{id}); e != nil {
			cause = errors.Join(cause, fmt.Errorf("compensate %s: %w", DeleteRegionsStepID, e))
		}
		for i := len(removed) - 1; i >= 0; i-- {
			r := removed[i]
			if _, e := links.Restore(ctx, r.definition, []map[string]any{r.fields}); e != nil {
				cause = errors.Join(cause, fmt.Errorf("restore region link: %w", e))
			}
		}
		return cause
	}
	for _, def := range linkmodels.Definitions {
		if !hasField(def, "region_id") {
			continue
		}
		filters := map[string]any{"region_id": id}
		rows, err := links.List(ctx, def, filters)
		if err != nil {
			return rollback(err)
		}
		if len(rows) == 0 {
			continue
		}
		if _, err = links.Dismiss(ctx, def, []map[string]any{filters}); err != nil {
			return rollback(err)
		}
		for _, row := range rows {
			removed = append(removed, removedLink{def, row.Fields})
		}
	}
	if events == nil {
		return rollback(errors.New("region event publisher is required"))
	}
	if err := events.Publish(ctx, RegionDeletedEvent, []map[string]any{{"id": id}}); err != nil {
		return rollback(err)
	}
	return nil
}

func rollbackRegion(ctx context.Context, service RegionModifier, id string, previous models.Region, cause error) (models.Region, error) {
	name, currency, taxes := previous.Name, previous.CurrencyCode, previous.AutomaticTaxes
	input := models.RegionInput{Name: &name, CurrencyCode: &currency, AutomaticTaxes: &taxes, Countries: make([]string, len(previous.Countries)), Metadata: previous.Metadata}
	for i, country := range previous.Countries {
		input.Countries[i] = country.ISO2
	}
	if _, err := service.Update(ctx, id, input); err != nil {
		cause = errors.Join(cause, fmt.Errorf("compensate %s: %w", UpdateRegionsStepID, err))
	}
	return models.Region{}, cause
}
func regionPaymentProviderDefinition() (linkmodels.Definition, error) {
	for _, def := range linkmodels.Definitions {
		if def.Name == "RegionPaymentProvider" {
			return def, nil
		}
	}
	return linkmodels.Definition{}, errors.New("RegionPaymentProvider link definition is missing")
}
func hasField(def linkmodels.Definition, name string) bool {
	for _, field := range def.ForeignKeys {
		if field == name {
			return true
		}
	}
	return false
}
