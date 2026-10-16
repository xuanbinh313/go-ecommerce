package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) UpdateShippingOption(ctx context.Context, id string, input models.ShippingOptionUpdateInput) (models.ShippingOption, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.ShippingOption{}, err
	}
	defer tx.Rollback(ctx)

	var current models.ShippingOption
	if err = scanShippingOption(tx.QueryRow(ctx, "SELECT "+shippingOptionColumns+" FROM shipping_option WHERE id=$1 AND deleted_at IS NULL FOR UPDATE", id), &current); err != nil {
		return models.ShippingOption{}, err
	}
	name, priceType := current.Name, current.PriceType
	providerID, profileID := current.ProviderID, current.ShippingProfileID
	data, metadata := current.Data, current.Metadata
	typeID := current.ShippingOptionTypeID
	if input.Name != nil {
		name = *input.Name
	}
	if input.PriceType != nil {
		priceType = *input.PriceType
	}
	if input.ProviderID != nil {
		providerID = input.ProviderID
	}
	if input.ShippingProfileID != nil {
		profileID = input.ShippingProfileID
	}
	if input.Data != nil {
		data, err = json.Marshal(*input.Data)
		if err != nil {
			return models.ShippingOption{}, err
		}
	}
	if input.Metadata != nil {
		metadata, err = json.Marshal(*input.Metadata)
		if err != nil {
			return models.ShippingOption{}, err
		}
	}
	if input.Type != nil {
		if _, err = tx.Exec(ctx, `UPDATE shipping_option_type SET label=$2,description=$3,code=$4,updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, typeID, input.Type.Label, input.Type.Description, input.Type.Code); err != nil {
			return models.ShippingOption{}, err
		}
	} else if input.TypeID != nil {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM shipping_option_type WHERE id=$1 AND deleted_at IS NULL)`, *input.TypeID).Scan(&exists); err != nil {
			return models.ShippingOption{}, err
		}
		if !exists {
			return models.ShippingOption{}, pgx.ErrNoRows
		}
		typeID = *input.TypeID
	}
	if _, err = tx.Exec(ctx, `UPDATE shipping_option SET name=$2,price_type=$3,data=$4,metadata=$5,provider_id=$6,shipping_profile_id=$7,shipping_option_type_id=$8,updated_at=now() WHERE id=$1`, id, name, priceType, data, metadata, providerID, profileID, typeID); err != nil {
		return models.ShippingOption{}, err
	}

	if input.Rules != nil {
		retainedRuleIDs := make([]string, 0, len(*input.Rules))
		for _, rule := range *input.Rules {
			if rule.ID == "" {
				if rule.Attribute == nil || rule.Operator == nil || rule.Value == nil {
					return models.ShippingOption{}, fmt.Errorf("shipping option rule requires attribute, operator, and value")
				}
				if _, err = tx.Exec(ctx, `INSERT INTO shipping_option_rule(id,attribute,operator,value,shipping_option_id) VALUES($1,$2,$3,$4,$5)`, entityid.New("sorul"), *rule.Attribute, *rule.Operator, *rule.Value, id); err != nil {
					return models.ShippingOption{}, err
				}
				continue
			}
			retainedRuleIDs = append(retainedRuleIDs, rule.ID)
			_, err = tx.Exec(ctx, `UPDATE shipping_option_rule SET attribute=COALESCE($3,attribute),operator=COALESCE($4,operator),value=COALESCE($5,value),deleted_at=NULL,updated_at=now() WHERE id=$1 AND shipping_option_id=$2`, rule.ID, id, rule.Attribute, rule.Operator, rule.Value)
			if err != nil {
				return models.ShippingOption{}, err
			}
		}
		if len(retainedRuleIDs) == 0 {
			if _, err = tx.Exec(ctx, `UPDATE shipping_option_rule SET deleted_at=now(),updated_at=now() WHERE shipping_option_id=$1 AND deleted_at IS NULL`, id); err != nil {
				return models.ShippingOption{}, err
			}
		} else if _, err = tx.Exec(ctx, `UPDATE shipping_option_rule SET deleted_at=now(),updated_at=now() WHERE shipping_option_id=$1 AND deleted_at IS NULL AND id<>ALL($2::text[])`, id, retainedRuleIDs); err != nil {
			return models.ShippingOption{}, err
		}
	}
	if input.Prices != nil {
		if priceType == "calculated" {
			if _, err = tx.Exec(ctx, `UPDATE price SET deleted_at=now(),updated_at=now() WHERE price_set_id IN (SELECT price_set_id FROM shipping_option_price_set WHERE shipping_option_id=$1 AND deleted_at IS NULL) AND deleted_at IS NULL`, id); err != nil {
				return models.ShippingOption{}, err
			}
		} else if err = updateShippingOptionPrices(ctx, tx, id, *input.Prices); err != nil {
			return models.ShippingOption{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.ShippingOption{}, err
	}
	return r.GetShippingOption(ctx, id)
}

func updateShippingOptionPrices(ctx context.Context, tx pgx.Tx, optionID string, prices []models.ShippingOptionUpdatePriceInput) error {
	var priceSetID string
	if err := tx.QueryRow(ctx, `SELECT price_set_id FROM shipping_option_price_set WHERE shipping_option_id=$1 AND deleted_at IS NULL`, optionID).Scan(&priceSetID); err != nil {
		return err
	}
	retained := make([]string, 0, len(prices))
	for _, price := range prices {
		priceID := ""
		if price.ID != nil {
			priceID = *price.ID
		}
		currencyCode, amountText := "", ""
		var currentRegionID *string
		if priceID != "" {
			var belongs bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM price WHERE id=$1 AND price_set_id=$2)`, priceID, priceSetID).Scan(&belongs); err != nil {
				return err
			}
			if !belongs {
				return fmt.Errorf("shipping option price %s not found", priceID)
			}
			if err := tx.QueryRow(ctx, `SELECT currency_code,amount::text FROM price WHERE id=$1`, priceID).Scan(&currencyCode, &amountText); err != nil {
				return err
			}
			var regionID string
			if err := tx.QueryRow(ctx, `SELECT value FROM price_rule WHERE price_id=$1 AND attribute='region_id' AND deleted_at IS NULL ORDER BY priority LIMIT 1`, priceID).Scan(&regionID); err == nil {
				currentRegionID = &regionID
			} else if err != pgx.ErrNoRows {
				return err
			}
		} else {
			if price.Amount == nil {
				return fmt.Errorf("shipping option price amount is required")
			}
			if price.CurrencyCode == nil && price.RegionID == nil {
				return fmt.Errorf("shipping option price requires currency_code or region_id")
			}
			priceID = entityid.New("price")
		}
		if price.Amount != nil {
			amountText = strings.TrimSpace(string(*price.Amount))
		}
		if _, ok := new(big.Rat).SetString(amountText); !ok {
			return fmt.Errorf("invalid shipping option price amount")
		}
		if price.RegionID != nil {
			var regionCurrency string
			if err := tx.QueryRow(ctx, `SELECT currency_code FROM region WHERE id=$1 AND deleted_at IS NULL`, *price.RegionID).Scan(&regionCurrency); err != nil {
				return err
			}
			currencyCode = regionCurrency
			currentRegionID = price.RegionID
		} else if price.CurrencyCode != nil {
			currencyCode = *price.CurrencyCode
			currentRegionID = nil
		}
		var currencyExists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM currency WHERE code=$1)`, currencyCode).Scan(&currencyExists); err != nil {
			return err
		}
		if !currencyExists {
			return fmt.Errorf("currency %q not found", currencyCode)
		}
		rawAmount, err := json.Marshal(map[string]any{"value": amountText, "precision": 20})
		if err != nil {
			return err
		}
		if price.ID != nil && *price.ID != "" {
			if _, err = tx.Exec(ctx, `UPDATE price SET currency_code=$2,amount=$3,raw_amount=$4,deleted_at=NULL,updated_at=now() WHERE id=$1`, priceID, currencyCode, amountText, rawAmount); err != nil {
				return err
			}
		} else if _, err = tx.Exec(ctx, `INSERT INTO price(id,currency_code,amount,raw_amount,price_set_id) VALUES($1,$2,$3,$4,$5)`, priceID, currencyCode, amountText, rawAmount, priceSetID); err != nil {
			return err
		}
		retained = append(retained, priceID)
		if price.Rules != nil || price.RegionID != nil || price.CurrencyCode != nil {
			if _, err = tx.Exec(ctx, `UPDATE price_rule SET deleted_at=now(),updated_at=now() WHERE price_id=$1 AND deleted_at IS NULL`, priceID); err != nil {
				return err
			}
			rules := []models.ShippingOptionPriceRuleInput{}
			if price.Rules != nil {
				rules = append(rules, (*price.Rules)...)
			}
			if currentRegionID != nil {
				value, _ := json.Marshal(*currentRegionID)
				rules = append(rules, models.ShippingOptionPriceRuleInput{Attribute: "region_id", Operator: "eq", Value: value})
			}
			for _, rule := range rules {
				var value string
				if err = json.Unmarshal(rule.Value, &value); err != nil {
					value = strings.Trim(string(rule.Value), `"`)
				}
				if _, err = tx.Exec(ctx, `INSERT INTO price_rule(id,attribute,value,operator,priority,price_id) VALUES($1,$2,$3,$4,0,$5)`, entityid.New("prule"), rule.Attribute, value, rule.Operator, priceID); err != nil {
					return err
				}
			}
		}
	}
	if len(retained) == 0 {
		_, err := tx.Exec(ctx, `UPDATE price SET deleted_at=now(),updated_at=now() WHERE price_set_id=$1 AND deleted_at IS NULL`, priceSetID)
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE price SET deleted_at=now(),updated_at=now() WHERE price_set_id=$1 AND deleted_at IS NULL AND id<>ALL($2::text[])`, priceSetID, retained)
	return err
}
