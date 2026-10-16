package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/promotion/domain"
	"github.com/example/medusa-go-clone/internal/modules/promotion/domain/models"
)

type Service struct{ repo domain.Repository }

func New(r domain.Repository) *Service { return &Service{repo: r} }
func (s *Service) ListCampaigns(c context.Context, f models.CampaignFilter) ([]models.Campaign, int, error) {
	return s.repo.ListCampaigns(c, f)
}
func (s *Service) GetCampaign(c context.Context, id string) (models.Campaign, error) {
	return s.repo.GetCampaign(c, id)
}
func (s *Service) CreateCampaign(c context.Context, in models.CampaignInput) (models.Campaign, error) {
	if in.Name == "" || in.CampaignIdentifier == "" {
		return models.Campaign{}, fmt.Errorf("name and campaign_identifier are required")
	}
	if in.Budget != nil {
		b := in.Budget
		switch b.Type {
		case "usage", "spend", "use_by_attribute", "spend_by_attribute":
		default:
			return models.Campaign{}, fmt.Errorf("invalid campaign budget type")
		}
		if (b.Type == "spend" && (b.CurrencyCode == nil || *b.CurrencyCode == "")) || ((b.Type == "usage" || b.Type == "use_by_attribute") && b.CurrencyCode != nil) {
			return models.Campaign{}, fmt.Errorf("invalid currency_code for campaign budget type")
		}
		if (b.Type == "use_by_attribute" || b.Type == "spend_by_attribute") && (b.Attribute == nil || *b.Attribute == "") {
			return models.Campaign{}, fmt.Errorf("campaign budget attribute is required")
		}
		if _, err := campaignBudgetLimit(b.Limit); err != nil {
			return models.Campaign{}, err
		}
	}
	return s.repo.CreateCampaign(c, in)
}
func (s *Service) UpdateCampaign(c context.Context, id string, in models.CampaignUpdate) (models.Campaign, error) {
	if in.Budget != nil {
		if _, err := campaignBudgetLimit(in.Budget.Limit); err != nil {
			return models.Campaign{}, err
		}
	}
	return s.repo.UpdateCampaign(c, id, in)
}
func (s *Service) DeleteCampaign(c context.Context, id string) error {
	return s.repo.DeleteCampaign(c, id)
}
func (s *Service) LinkCampaignPromotions(c context.Context, id string, add, remove []string) error {
	return s.repo.LinkCampaignPromotions(c, id, add, remove)
}
func (s *Service) List(c context.Context, f models.Filter) ([]models.Promotion, int, error) {
	return s.repo.List(c, f)
}
func (s *Service) Get(c context.Context, id string) (models.Promotion, error) {
	return s.repo.Get(c, id)
}
func (s *Service) Create(c context.Context, in models.CreateInput) (models.Promotion, error) {
	if in.Code == "" {
		return models.Promotion{}, fmt.Errorf("code is required")
	}
	if in.Type != "standard" && in.Type != "buyget" {
		return models.Promotion{}, fmt.Errorf("invalid promotion type")
	}
	if in.Status == "" {
		in.Status = "draft"
	}
	if in.Status != "draft" && in.Status != "active" && in.Status != "inactive" {
		return models.Promotion{}, fmt.Errorf("invalid promotion status")
	}
	if in.IsAutomatic != nil && *in.IsAutomatic && in.Limit != nil {
		return models.Promotion{}, fmt.Errorf("automatic promotions cannot have a usage limit")
	}
	if in.ApplicationMethod.Type != "fixed" && in.ApplicationMethod.Type != "percentage" {
		return models.Promotion{}, fmt.Errorf("invalid application method type")
	}
	if in.ApplicationMethod.Value == nil {
		return models.Promotion{}, fmt.Errorf("application_method.value is required")
	}
	if in.ApplicationMethod.TargetType != "order" && in.ApplicationMethod.TargetType != "shipping_methods" && in.ApplicationMethod.TargetType != "items" {
		return models.Promotion{}, fmt.Errorf("invalid application method target type")
	}
	if in.Type == "buyget" && (len(in.ApplicationMethod.BuyRules) == 0 || in.ApplicationMethod.ApplyToQuantity == nil || in.ApplicationMethod.BuyRulesMinQuantity == nil) {
		return models.Promotion{}, fmt.Errorf("buyget promotions require buy rules and quantities to be defined")
	}
	for _, rule := range append(append([]models.RuleInput{}, in.Rules...), append(in.ApplicationMethod.TargetRules, in.ApplicationMethod.BuyRules...)...) {
		if rule.Attribute == "" || !validOperator(rule.Operator) || len(rule.Values) == 0 {
			return models.Promotion{}, fmt.Errorf("promotion rules require an attribute, valid operator, and values")
		}
	}
	return s.repo.Create(c, in)
}
func (s *Service) Update(c context.Context, id string, in models.UpdateInput) (models.Promotion, error) {
	if in.IsAutomatic != nil && *in.IsAutomatic && in.Limit != nil {
		return models.Promotion{}, fmt.Errorf("automatic promotions cannot have a usage limit")
	}
	return s.repo.Update(c, id, in)
}
func (s *Service) Delete(c context.Context, id string) error { return s.repo.Delete(c, id) }

func validOperator(v string) bool {
	switch v {
	case "gte", "lte", "gt", "lt", "eq", "ne", "in":
		return true
	}
	return false
}

func campaignBudgetLimit(raw json.RawMessage) (*string, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var value any
	if err := dec.Decode(&value); err != nil {
		return nil, fmt.Errorf("budget.limit must be a number or null")
	}
	n, ok := value.(json.Number)
	if !ok {
		return nil, fmt.Errorf("budget.limit must be a number or null")
	}
	text := n.String()
	if text == "" {
		return nil, fmt.Errorf("budget.limit must be a number or null")
	}
	return &text, nil
}
