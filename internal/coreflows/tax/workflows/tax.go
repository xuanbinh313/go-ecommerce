package workflows

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/tax/domain/models"
)

const CreateTaxRegionsWorkflowID = "create-tax-regions"
const CreateTaxRegionsStepID = "create-tax-regions"
const UpdateTaxRegionsWorkflowID = "update-tax-regions"
const UpdateTaxRegionsStepID = "update-tax-regions"
const DeleteTaxRegionsWorkflowID = "delete-tax-regions"
const DeleteTaxRegionsStepID = "delete-tax-regions"
const CreateTaxRatesWorkflowID = "create-tax-rates"
const CreateTaxRatesStepID = "create-tax-rates"
const UpdateTaxRatesWorkflowID = "update-tax-rates"
const UpdateTaxRatesStepID = "update-tax-rates"
const DeleteTaxRatesWorkflowID = "delete-tax-rates"
const DeleteTaxRatesStepID = "delete-tax-rates"
const CreateTaxRateRulesWorkflowID = "create-tax-rate-rules"
const CreateTaxRateRulesStepID = "create-tax-rate-rules"
const DeleteTaxRateRulesWorkflowID = "delete-tax-rate-rules"
const DeleteTaxRateRulesStepID = "delete-tax-rate-rules"

type Service interface {
	CreateRegion(context.Context, models.RegionInput) (models.TaxRegion, error)
	UpdateRegion(context.Context, string, models.RegionUpdate) (models.TaxRegion, error)
	DeleteRegion(context.Context, string) error
	CreateRate(context.Context, models.RateInput, string, *string) (models.TaxRate, error)
	UpdateRate(context.Context, string, models.RateUpdate, *string) (models.TaxRate, error)
	DeleteRate(context.Context, string) error
	CreateRateRules(context.Context, []models.RuleCreateInput) error
	DeleteRateRules(context.Context, []string) error
}

func CreateTaxRegion(c context.Context, s Service, in models.RegionInput) (models.TaxRegion, error) {
	return s.CreateRegion(c, in)
}
func UpdateTaxRegion(c context.Context, s Service, id string, in models.RegionUpdate) (models.TaxRegion, error) {
	return s.UpdateRegion(c, id, in)
}
func DeleteTaxRegion(c context.Context, s Service, id string) error { return s.DeleteRegion(c, id) }
func CreateTaxRate(c context.Context, s Service, in models.RateInput, region string, actor *string) (models.TaxRate, error) {
	return s.CreateRate(c, in, region, actor)
}
func UpdateTaxRate(c context.Context, s Service, id string, in models.RateUpdate, actor *string) (models.TaxRate, error) {
	return s.UpdateRate(c, id, in, actor)
}
func DeleteTaxRate(c context.Context, s Service, id string) error { return s.DeleteRate(c, id) }
func CreateTaxRateRules(c context.Context, s Service, rules []models.RuleCreateInput) error {
	return s.CreateRateRules(c, rules)
}
func DeleteTaxRateRules(c context.Context, s Service, ids []string) error {
	return s.DeleteRateRules(c, ids)
}
