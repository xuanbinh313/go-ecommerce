package workflows

import (
	"context"
	"errors"
	"fmt"
	"sync"

	linkdomain "github.com/example/medusa-go-clone/internal/modules/links/domain"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	regiondomain "github.com/example/medusa-go-clone/internal/modules/region/domain"
	regionmodels "github.com/example/medusa-go-clone/internal/modules/region/domain/models"
)

const CreateRegionsWorkflowID = "create-regions"
const CreateRegionsStepID = "create-regions"
const SetRegionsPaymentProvidersStepID = "add-region-payment-providers-step"
const EmitEventStepID = "emit-event-step"
const RegionCreatedEvent = "region.created"
const RegionUpdatedEvent = "region.updated"
const RegionDeletedEvent = "region.deleted"
const UpdateRegionsWorkflowID = "update-regions"
const UpdateRegionsStepID = "update-region"
const DeleteRegionsWorkflowID = "delete-regions"
const DeleteRegionsStepID = "delete-regions"

type PaymentProviderValidator interface {
	ValidatePaymentProviders(context.Context, []string) error
}
type RegionPricePreference struct {
	Attribute      string `json:"attribute"`
	Value          string `json:"value"`
	IsTaxInclusive *bool  `json:"is_tax_inclusive,omitempty"`
}
type RegionPricePreferenceWriter interface {
	CreateRegionPricePreferences(context.Context, []RegionPricePreference) error
	DeleteRegionPricePreferences(context.Context, []string) error
}
type EventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}

// CreateRegions follows Medusa's create-regions step and then runs payment
// links, regional price preferences, and event emission in parallel.
func CreateRegions(ctx context.Context, regions regiondomain.WorkflowService, links linkdomain.Repository, payment PaymentProviderValidator, prices RegionPricePreferenceWriter, events EventPublisher, inputs []regionmodels.RegionInput) ([]regionmodels.Region, error) {
	if prices == nil || events == nil {
		return nil, errors.New("region pricing and event ports are required")
	}
	created := make([]regionmodels.Region, 0, len(inputs))
	for _, input := range inputs {
		region, err := regions.Create(ctx, input)
		if err != nil {
			ids := regionIDs(created)
			if len(ids) > 0 {
				if e := regions.DeleteRegions(ctx, ids); e != nil {
					return nil, errors.Join(err, fmt.Errorf("delete created regions: %w", e))
				}
			}
			return nil, err
		}
		created = append(created, region)
	}
	ids := regionIDs(created)
	regionData := make([]RegionPricePreference, len(created))
	eventData := make([]map[string]any, len(created))
	for i, r := range created {
		regionData[i] = RegionPricePreference{Attribute: "region_id", Value: r.ID, IsTaxInclusive: inputs[i].IsTaxInclusive}
		eventData[i] = map[string]any{"id": r.ID}
	}
	providerLinks := []map[string]any{}
	providerIDs := []string{}
	for i, in := range inputs {
		for _, providerID := range in.PaymentProviders {
			providerIDs = append(providerIDs, providerID)
			providerLinks = append(providerLinks, map[string]any{"region_id": created[i].ID, "payment_provider_id": providerID})
		}
	}
	var providerDefinition linkmodels.Definition
	for _, d := range linkmodels.Definitions {
		if d.Name == "RegionPaymentProvider" {
			providerDefinition = d
			break
		}
	}
	var mu sync.Mutex
	errs := []error{}
	record := func(err error) {
		if err != nil {
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
		}
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if len(providerLinks) == 0 {
			return
		}
		if payment == nil {
			record(errors.New("payment provider validator is required"))
			return
		}
		if err := payment.ValidatePaymentProviders(ctx, unique(providerIDs)); err != nil {
			record(err)
			return
		}
		if providerDefinition.Name == "" {
			record(errors.New("RegionPaymentProvider link definition is missing"))
			return
		}
		if _, err := links.Create(ctx, providerDefinition, providerLinks); err != nil {
			record(err)
		}
	}()
	go func() { defer wg.Done(); record(prices.CreateRegionPricePreferences(ctx, regionData)) }()
	// Medusa's event step is a third parallel branch after module creation.
	wg.Add(1)
	go func() { defer wg.Done(); record(events.Publish(ctx, RegionCreatedEvent, eventData)) }()
	wg.Wait()
	if len(errs) > 0 {
		cause := errors.Join(errs...)
		if len(providerLinks) > 0 && providerDefinition.Name != "" {
			if _, err := links.Dismiss(ctx, providerDefinition, providerLinks); err != nil {
				cause = errors.Join(cause, fmt.Errorf("dismiss payment-provider links: %w", err))
			}
		}
		if err := prices.DeleteRegionPricePreferences(ctx, ids); err != nil {
			cause = errors.Join(cause, fmt.Errorf("delete regional price preferences: %w", err))
		}
		if err := regions.DeleteRegions(ctx, ids); err != nil {
			cause = errors.Join(cause, fmt.Errorf("delete created regions: %w", err))
		}
		return nil, cause
	}
	return created, nil
}
func regionIDs(regions []regionmodels.Region) []string {
	ids := make([]string, len(regions))
	for i, r := range regions {
		ids[i] = r.ID
	}
	return ids
}
func unique(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
