package workflows

import "context"

const SetLayoutConfigurationWorkflowID = "set-layout-configuration"
const SetLayoutConfigurationStepID = "set-layout-configuration"
const ClearLayoutConfigurationWorkflowID = "clear-layout-configuration"
const ClearLayoutConfigurationStepID = "clear-layout-configuration"

type LayoutConfigurationService interface {
	SetLayoutConfiguration(context.Context, string, string, bool, []byte) error
	ClearLayoutConfiguration(context.Context, string, string) error
}

// SetLayoutConfiguration follows Medusa's set-layout-configuration flow.
func SetLayoutConfiguration(ctx context.Context, service LayoutConfigurationService, zone, userID string, isDefault bool, configuration []byte) error {
	return service.SetLayoutConfiguration(ctx, zone, userID, isDefault, configuration)
}

// ClearLayoutConfiguration follows Medusa's clear-layout-configuration flow.
func ClearLayoutConfiguration(ctx context.Context, service LayoutConfigurationService, zone, userID string) error {
	return service.ClearLayoutConfiguration(ctx, zone, userID)
}
