package workflows

import (
	"context"
	"fmt"

	fulfillmentmodels "github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
)

const (
	CreateShippingProfilesWorkflowID    = "create-shipping-profiles-workflow"
	UpdateShippingProfilesWorkflowID    = "update-shipping-profiles-workflow"
	DeleteShippingProfileWorkflowID     = "delete-shipping-profile-workflow"
	CreateShippingProfilesStepID        = "create-shipping-profiles"
	UpdateShippingProfilesStepID        = "update-shipping-profiles"
	DeleteShippingProfilesStepID        = "delete-shipping-profiles"
	ValidateShippingProfileDeleteStepID = "validate-step-shipping-profile-delete"
)

type ShippingProfileService interface {
	ListShippingProfiles(context.Context, fulfillmentmodels.ShippingProfileFilter, int, int) ([]fulfillmentmodels.ShippingProfile, int, error)
	GetShippingProfile(context.Context, string) (fulfillmentmodels.ShippingProfile, error)
	CreateShippingProfiles(context.Context, []fulfillmentmodels.ShippingProfileInput) ([]fulfillmentmodels.ShippingProfile, error)
	UpdateShippingProfile(context.Context, string, map[string]any) (fulfillmentmodels.ShippingProfile, error)
	SoftDeleteShippingProfiles(context.Context, []string) error
	RestoreShippingProfiles(context.Context, []string) error
	HardDeleteShippingProfiles(context.Context, []string) error
}
type ShippingProfileLinkService interface {
	List(context.Context, string, map[string]any) ([]linkmodels.Link, error)
}

func CreateShippingProfiles(ctx context.Context, s ShippingProfileService, in []fulfillmentmodels.ShippingProfileInput) ([]fulfillmentmodels.ShippingProfile, error) {
	return s.CreateShippingProfiles(ctx, in)
}
func UpdateShippingProfile(ctx context.Context, s ShippingProfileService, id string, fields map[string]any) (fulfillmentmodels.ShippingProfile, error) {
	old, err := s.GetShippingProfile(ctx, id)
	if err != nil {
		return fulfillmentmodels.ShippingProfile{}, err
	}
	updated, err := s.UpdateShippingProfile(ctx, id, fields)
	if err != nil {
		_, _ = s.UpdateShippingProfile(ctx, id, map[string]any{"name": old.Name, "type": old.Type, "metadata": old.Metadata})
		return fulfillmentmodels.ShippingProfile{}, err
	}
	return updated, nil
}
func DeleteShippingProfile(ctx context.Context, s ShippingProfileService, links ShippingProfileLinkService, id string) error {
	found := false
	for _, candidate := range linkmodels.Definitions {
		if candidate.Name == "ProductShippingProfile" {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("missing ProductShippingProfile link definition")
	}
	linked, err := links.List(ctx, "ProductShippingProfile", map[string]any{"shipping_profile_id": []string{id}})
	if err != nil {
		return err
	}
	if len(linked) > 0 {
		productIDs := make([]string, 0, len(linked))
		for _, link := range linked {
			if pid, ok := link.Fields["product_id"].(string); ok {
				productIDs = append(productIDs, pid)
			}
		}
		return fmt.Errorf("shipping profile is linked to products: %v", productIDs)
	}
	if err = s.SoftDeleteShippingProfiles(ctx, []string{id}); err != nil {
		return err
	}
	return nil
}
