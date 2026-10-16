package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	fulfillmentdomain "github.com/example/medusa-go-clone/internal/modules/fulfillment/domain"
	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
)

const shippingOptionColumns = `id,name,price_type,data,provider_id,metadata,service_zone_id,shipping_profile_id,shipping_option_type_id,created_at,updated_at`

func (r *Repository) ListShippingOptions(ctx context.Context, f models.ShippingOptionFilter, limit, offset int) ([]models.ShippingOption, int, error) {
	where, args := []string{}, []any{}
	if len(f.DeletedAt) == 0 {
		where = append(where, "shipping_option.deleted_at IS NULL")
	}
	add := func(values []string, column string) {
		if len(values) > 0 {
			args = append(args, values)
			where = append(where, fmt.Sprintf("%s=ANY($%d::text[])", column, len(args)))
		}
	}
	add(f.IDs, "id")
	add(f.ServiceZoneIDs, "service_zone_id")
	add(f.ShippingProfileIDs, "shipping_profile_id")
	add(f.ProviderIDs, "provider_id")
	add(f.TypeIDs, "shipping_option_type_id")
	if len(f.StockLocationIDs) > 0 {
		args = append(args, f.StockLocationIDs)
		where = append(where, fmt.Sprintf(`EXISTS (SELECT 1 FROM service_zone sz JOIN location_fulfillment_set lfs ON lfs.fulfillment_set_id=sz.fulfillment_set_id AND lfs.deleted_at IS NULL WHERE sz.id=shipping_option.service_zone_id AND sz.deleted_at IS NULL AND lfs.stock_location_id=ANY($%d::text[]))`, len(args)))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where = append(where, fmt.Sprintf("name ILIKE $%d", len(args)))
	}
	for _, dateFilter := range []struct {
		column string
		values map[string]string
	}{{"shipping_option.created_at", f.CreatedAt}, {"shipping_option.updated_at", f.UpdatedAt}, {"shipping_option.deleted_at", f.DeletedAt}} {
		if err := appendShippingOptionDateConditions(&where, &args, dateFilter.column, dateFilter.values); err != nil {
			return nil, 0, err
		}
	}
	w := strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM shipping_option WHERE "+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf("SELECT %s FROM shipping_option WHERE %s ORDER BY created_at DESC,id LIMIT $%d OFFSET $%d", shippingOptionColumns, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var v models.ShippingOption
		if err = scanShippingOption(rows, &v); err != nil {
			return nil, 0, err
		}
		ids = append(ids, v.ID)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	out := make([]models.ShippingOption, 0, len(ids))
	for _, id := range ids {
		v, e := r.GetShippingOption(ctx, id)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, nil
}

func appendShippingOptionDateConditions(conditions *[]string, args *[]any, column string, ops map[string]string) error {
	operators := map[string]string{"$eq": "=", "$ne": "<>", "$gt": ">", "$gte": ">=", "$lt": "<", "$lte": "<="}
	for key, value := range ops {
		if column == "shipping_option.deleted_at" && (value == "null" || value == "") {
			if key == "$eq" {
				*conditions = append(*conditions, column+" IS NULL")
				continue
			}
			if key == "$ne" {
				*conditions = append(*conditions, column+" IS NOT NULL")
				continue
			}
		}
		op, ok := operators[key]
		if !ok {
			return fmt.Errorf("unsupported shipping option date operator %q", key)
		}
		*args = append(*args, value)
		*conditions = append(*conditions, fmt.Sprintf("%s %s $%d", column, op, len(*args)))
	}
	return nil
}

func scanShippingOption(row pgx.Row, v *models.ShippingOption) error {
	return row.Scan(&v.ID, &v.Name, &v.PriceType, &v.Data, &v.ProviderID, &v.Metadata, &v.ServiceZoneID, &v.ShippingProfileID, &v.ShippingOptionTypeID, &v.CreatedAt, &v.UpdatedAt)
}

func (r *Repository) GetShippingOption(ctx context.Context, id string) (models.ShippingOption, error) {
	var v models.ShippingOption
	if err := scanShippingOption(r.db.QueryRow(ctx, "SELECT "+shippingOptionColumns+" FROM shipping_option WHERE id=$1 AND deleted_at IS NULL", id), &v); err != nil {
		return v, err
	}
	v.Rules = []models.ShippingOptionRule{}
	rows, err := r.db.Query(ctx, "SELECT id,attribute,operator,value FROM shipping_option_rule WHERE shipping_option_id=$1 AND deleted_at IS NULL ORDER BY created_at,id", id)
	if err != nil {
		return v, err
	}
	for rows.Next() {
		var rule models.ShippingOptionRule
		if err = rows.Scan(&rule.ID, &rule.Attribute, &rule.Operator, &rule.Value); err != nil {
			rows.Close()
			return v, err
		}
		v.Rules = append(v.Rules, rule)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return v, err
	}
	typ, err := r.GetShippingOptionType(ctx, v.ShippingOptionTypeID)
	if err != nil {
		return v, err
	}
	v.Type = &typ
	zone, err := r.GetServiceZone(ctx, v.ServiceZoneID)
	if err != nil {
		return v, err
	}
	v.ServiceZone = &zone
	if v.ShippingProfileID != nil {
		profile, e := r.GetShippingProfile(ctx, *v.ShippingProfileID)
		if e == nil {
			v.ShippingProfile = &profile
		} else if e != pgx.ErrNoRows {
			return v, e
		}
	}
	if v.ProviderID != nil {
		var enabled bool
		e := r.db.QueryRow(ctx, "SELECT is_enabled FROM fulfillment_provider WHERE id=$1 AND deleted_at IS NULL", *v.ProviderID).Scan(&enabled)
		if e == nil {
			v.Provider = map[string]any{"id": *v.ProviderID, "is_enabled": enabled}
		} else if e != pgx.ErrNoRows {
			return v, e
		}
	}
	v.Prices, err = r.listShippingOptionPrices(ctx, id)
	if err != nil {
		return v, err
	}
	return v, nil
}

func (r *Repository) listShippingOptionPrices(ctx context.Context, id string) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, `SELECT p.id,p.title,p.currency_code,p.amount,p.min_quantity,p.max_quantity,p.price_set_id FROM shipping_option_price_set l JOIN price p ON p.price_set_id=l.price_set_id AND p.deleted_at IS NULL WHERE l.shipping_option_id=$1 AND l.deleted_at IS NULL ORDER BY p.created_at,p.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var priceID, code, setID string
		var title *string
		var amount string
		var min, max *string
		if err = rows.Scan(&priceID, &title, &code, &amount, &min, &max, &setID); err != nil {
			return nil, err
		}
		item := map[string]any{"id": priceID, "currency_code": code, "amount": json.Number(amount), "price_set_id": setID, "price_rules": []map[string]any{}}
		if title != nil {
			item["title"] = *title
		}
		if min != nil {
			item["min_quantity"] = json.Number(*min)
		}
		if max != nil {
			item["max_quantity"] = json.Number(*max)
		}
		rules, e := r.db.Query(ctx, "SELECT id,attribute,value,operator,priority FROM price_rule WHERE price_id=$1 AND deleted_at IS NULL ORDER BY priority,id", priceID)
		if e != nil {
			return nil, e
		}
		pr := []map[string]any{}
		for rules.Next() {
			var rid, attribute, value, operator string
			var priority int
			if e = rules.Scan(&rid, &attribute, &value, &operator, &priority); e != nil {
				rules.Close()
				return nil, e
			}
			pr = append(pr, map[string]any{"id": rid, "attribute": attribute, "value": value, "operator": operator, "priority": priority})
		}
		e = rules.Err()
		rules.Close()
		if e != nil {
			return nil, e
		}
		item["price_rules"] = pr
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) CanProviderServeZone(ctx context.Context, zoneID, providerID string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (
	 SELECT 1 FROM service_zone sz
	 JOIN location_fulfillment_set lfs ON lfs.fulfillment_set_id=sz.fulfillment_set_id AND lfs.deleted_at IS NULL
	 JOIN location_fulfillment_provider lfp ON lfp.stock_location_id=lfs.stock_location_id AND lfp.deleted_at IS NULL
	 WHERE sz.id=$1 AND sz.deleted_at IS NULL AND lfp.fulfillment_provider_id=$2
	)`, zoneID, providerID).Scan(&allowed)
	return allowed, err
}

func (r *Repository) CreateShippingOption(ctx context.Context, in models.ShippingOptionInput) (models.ShippingOption, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.ShippingOption{}, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM service_zone WHERE id=$1 AND deleted_at IS NULL)", in.ServiceZoneID).Scan(&exists); err != nil {
		return models.ShippingOption{}, err
	}
	if !exists {
		return models.ShippingOption{}, pgx.ErrNoRows
	}
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM shipping_profile WHERE id=$1 AND deleted_at IS NULL)", in.ShippingProfileID).Scan(&exists); err != nil {
		return models.ShippingOption{}, err
	}
	if !exists {
		return models.ShippingOption{}, pgx.ErrNoRows
	}
	providerOK := false
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM service_zone sz JOIN location_fulfillment_set lfs ON lfs.fulfillment_set_id=sz.fulfillment_set_id AND lfs.deleted_at IS NULL JOIN location_fulfillment_provider lfp ON lfp.stock_location_id=lfs.stock_location_id AND lfp.deleted_at IS NULL WHERE sz.id=$1 AND sz.deleted_at IS NULL AND lfp.fulfillment_provider_id=$2)`, in.ServiceZoneID, in.ProviderID).Scan(&providerOK)
	if err != nil {
		return models.ShippingOption{}, err
	}
	if !providerOK {
		return models.ShippingOption{}, fmt.Errorf("fulfillment provider %q is not enabled for the service zone", in.ProviderID)
	}
	// Provider rows are the Fulfillment module's local registration records.
	if _, err = tx.Exec(ctx, `INSERT INTO fulfillment_provider(id,is_enabled) VALUES($1,true) ON CONFLICT(id) DO UPDATE SET is_enabled=true,deleted_at=NULL,updated_at=now()`, in.ProviderID); err != nil {
		return models.ShippingOption{}, err
	}
	typeID := in.TypeID
	if in.Type != nil {
		typeID = entityid.New("sotype")
		if _, err = tx.Exec(ctx, `INSERT INTO shipping_option_type(id,label,description,code) VALUES($1,$2,$3,$4)`, typeID, in.Type.Label, in.Type.Description, in.Type.Code); err != nil {
			return models.ShippingOption{}, err
		}
	} else {
		if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM shipping_option_type WHERE id=$1 AND deleted_at IS NULL)", typeID).Scan(&exists); err != nil {
			return models.ShippingOption{}, err
		}
		if !exists {
			return models.ShippingOption{}, pgx.ErrNoRows
		}
	}
	id := entityid.New("so")
	data, err := json.Marshal(in.Data)
	if err != nil {
		return models.ShippingOption{}, err
	}
	meta, err := json.Marshal(in.Metadata)
	if err != nil {
		return models.ShippingOption{}, err
	}
	if in.Data == nil {
		data = []byte("null")
	}
	if in.Metadata == nil {
		meta = []byte("null")
	}
	if _, err = tx.Exec(ctx, `INSERT INTO shipping_option(id,name,price_type,data,metadata,service_zone_id,shipping_profile_id,provider_id,shipping_option_type_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, in.Name, in.PriceType, data, meta, in.ServiceZoneID, in.ShippingProfileID, in.ProviderID, typeID); err != nil {
		return models.ShippingOption{}, err
	}
	for _, rule := range in.Rules {
		value := rule.Value
		if len(value) == 0 {
			value = []byte("null")
		}
		if _, err = tx.Exec(ctx, `INSERT INTO shipping_option_rule(id,attribute,operator,value,shipping_option_id) VALUES($1,$2,$3,$4,$5)`, entityid.New("sorul"), rule.Attribute, rule.Operator, value, id); err != nil {
			return models.ShippingOption{}, err
		}
	}
	priceSetID := entityid.New("pset")
	if _, err = tx.Exec(ctx, `INSERT INTO price_set(id) VALUES($1)`, priceSetID); err != nil {
		return models.ShippingOption{}, err
	}
	for _, price := range in.Prices {
		currencyCode := ""
		if price.RegionID != nil && *price.RegionID != "" {
			if err = tx.QueryRow(ctx, "SELECT currency_code FROM region WHERE id=$1 AND deleted_at IS NULL", *price.RegionID).Scan(&currencyCode); err != nil {
				return models.ShippingOption{}, err
			}
		} else if price.CurrencyCode != nil {
			currencyCode = *price.CurrencyCode
		}
		if currencyCode == "" {
			return models.ShippingOption{}, fmt.Errorf("%w: shipping option price requires currency_code or region_id", fulfillmentdomain.ErrInvalidData)
		}
		if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM currency WHERE code=$1)", currencyCode).Scan(&exists); err != nil {
			return models.ShippingOption{}, err
		}
		if !exists {
			return models.ShippingOption{}, fmt.Errorf("%w: currency %q not found", fulfillmentdomain.ErrInvalidData, currencyCode)
		}
		amountText := strings.TrimSpace(string(price.Amount))
		_, ok := new(big.Rat).SetString(amountText)
		if !ok {
			return models.ShippingOption{}, fmt.Errorf("%w: invalid shipping option price amount", fulfillmentdomain.ErrInvalidData)
		}
		priceID := entityid.New("price")
		rawAmount, _ := json.Marshal(map[string]any{"value": amountText, "precision": 20})
		if _, err = tx.Exec(ctx, `INSERT INTO price(id,currency_code,amount,raw_amount,price_set_id) VALUES($1,$2,$3,$4,$5)`, priceID, currencyCode, amountText, rawAmount, priceSetID); err != nil {
			return models.ShippingOption{}, err
		}
		prules := append([]models.ShippingOptionPriceRuleInput{}, price.Rules...)
		if price.RegionID != nil && *price.RegionID != "" {
			value, _ := json.Marshal(*price.RegionID)
			prules = append(prules, models.ShippingOptionPriceRuleInput{Attribute: "region_id", Operator: "eq", Value: value})
		}
		for _, rule := range prules {
			if rule.Attribute == "" || !validPriceOperator(rule.Operator) {
				return models.ShippingOption{}, fmt.Errorf("%w: invalid price rule", fulfillmentdomain.ErrInvalidData)
			}
			var value string
			if err = json.Unmarshal(rule.Value, &value); err != nil {
				value = strings.Trim(string(rule.Value), "\"")
			}
			if _, err = tx.Exec(ctx, `INSERT INTO price_rule(id,attribute,value,operator,priority,price_id) VALUES($1,$2,$3,$4,0,$5)`, entityid.New("prule"), rule.Attribute, value, rule.Operator, priceID); err != nil {
				return models.ShippingOption{}, err
			}
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO shipping_option_price_set(id,shipping_option_id,price_set_id) VALUES($1,$2,$3)`, entityid.New("sops"), id, priceSetID); err != nil {
		return models.ShippingOption{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return models.ShippingOption{}, err
	}
	return r.GetShippingOption(ctx, id)
}

func validPriceOperator(v string) bool {
	switch v {
	case "eq", "ne", "gt", "gte", "lt", "lte":
		return true
	}
	return false
}

func (r *Repository) HardDeleteShippingOption(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var typeID string
	if err = tx.QueryRow(ctx, "SELECT shipping_option_type_id FROM shipping_option WHERE id=$1", id).Scan(&typeID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, "SELECT price_set_id FROM shipping_option_price_set WHERE shipping_option_id=$1", id)
	if err != nil {
		return err
	}
	sets := []string{}
	for rows.Next() {
		var set string
		if err = rows.Scan(&set); err != nil {
			rows.Close()
			return err
		}
		sets = append(sets, set)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM shipping_option_price_set WHERE shipping_option_id=$1", id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM shipping_option_rule WHERE shipping_option_id=$1", id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM shipping_option WHERE id=$1", id); err != nil {
		return err
	}
	for _, set := range sets {
		var linked bool
		if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM shipping_option_price_set WHERE price_set_id=$1 AND deleted_at IS NULL)", set).Scan(&linked); err != nil {
			return err
		}
		if !linked {
			if _, err = tx.Exec(ctx, "DELETE FROM price_rule WHERE price_id IN (SELECT id FROM price WHERE price_set_id=$1)", set); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, "DELETE FROM price WHERE price_set_id=$1", set); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, "DELETE FROM price_set WHERE id=$1", set); err != nil {
				return err
			}
		}
	}
	if _, err = tx.Exec(ctx, "DELETE FROM shipping_option_type WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM shipping_option WHERE shipping_option_type_id=$1)", typeID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) SoftDeleteShippingOptions(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE shipping_option SET deleted_at=now(),updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NULL`, ids)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != int64(len(ids)) {
		return pgx.ErrNoRows
	}
	if _, err = tx.Exec(ctx, `UPDATE shipping_option_price_set SET deleted_at=now(),updated_at=now() WHERE shipping_option_id=ANY($1::text[]) AND deleted_at IS NULL`, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) RestoreShippingOptions(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE shipping_option SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1::text[])`, ids); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE shipping_option_price_set SET deleted_at=NULL,updated_at=now() WHERE shipping_option_id=ANY($1::text[])`, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
