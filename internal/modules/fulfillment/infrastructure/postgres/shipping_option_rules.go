package postgres

import (
	"context"
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
)

func scanShippingOptionRule(row pgx.Row) (models.ShippingOptionRule, error) {
	var v models.ShippingOptionRule
	err := row.Scan(&v.ID, &v.Attribute, &v.Operator, &v.Value, &v.ShippingOptionID)
	return v, err
}

// BatchShippingOptionRules commits the create/update/delete groups together so
// an error cannot leave a partially applied batch.
func (r *Repository) BatchShippingOptionRules(ctx context.Context, optionID string, input models.ShippingOptionRuleBatchInput) (models.ShippingOptionRuleBatchResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.ShippingOptionRuleBatchResult{}, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM shipping_option WHERE id=$1 AND deleted_at IS NULL)", optionID).Scan(&exists); err != nil {
		return models.ShippingOptionRuleBatchResult{}, err
	}
	if !exists {
		return models.ShippingOptionRuleBatchResult{}, pgx.ErrNoRows
	}
	result := models.ShippingOptionRuleBatchResult{Created: []models.ShippingOptionRule{}, Updated: []models.ShippingOptionRule{}, Deleted: []string{}, PreviousUpdated: []models.ShippingOptionRule{}, DeletedRules: []models.ShippingOptionRule{}}
	for _, rule := range input.Create {
		value := rule.Value
		if len(value) == 0 {
			value = []byte("null")
		}
		created, e := scanShippingOptionRule(tx.QueryRow(ctx, `INSERT INTO shipping_option_rule(id,attribute,operator,value,shipping_option_id) VALUES($1,$2,$3,$4,$5) RETURNING id,attribute,operator,value,shipping_option_id`, entityid.New("sorul"), rule.Attribute, rule.Operator, value, optionID))
		if e != nil {
			return models.ShippingOptionRuleBatchResult{}, e
		}
		result.Created = append(result.Created, created)
	}
	for _, rule := range input.Update {
		previous, e := scanShippingOptionRule(tx.QueryRow(ctx, `SELECT id,attribute,operator,value,shipping_option_id FROM shipping_option_rule WHERE id=$1 AND deleted_at IS NULL`, rule.ID))
		if e != nil {
			return models.ShippingOptionRuleBatchResult{}, e
		}
		result.PreviousUpdated = append(result.PreviousUpdated, previous)
		updated, e := scanShippingOptionRule(tx.QueryRow(ctx, `UPDATE shipping_option_rule SET attribute=COALESCE($2,attribute),operator=COALESCE($3,operator),value=COALESCE($4,value),updated_at=now() WHERE id=$1 AND deleted_at IS NULL RETURNING id,attribute,operator,value,shipping_option_id`, rule.ID, rule.Attribute, rule.Operator, rule.Value))
		if e != nil {
			return models.ShippingOptionRuleBatchResult{}, e
		}
		result.Updated = append(result.Updated, updated)
	}
	for _, id := range input.Delete {
		previous, e := scanShippingOptionRule(tx.QueryRow(ctx, `SELECT id,attribute,operator,value,shipping_option_id FROM shipping_option_rule WHERE id=$1 AND deleted_at IS NULL`, id))
		if e != nil {
			return models.ShippingOptionRuleBatchResult{}, e
		}
		result.DeletedRules = append(result.DeletedRules, previous)
		tag, e := tx.Exec(ctx, `UPDATE shipping_option_rule SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
		if e != nil {
			return models.ShippingOptionRuleBatchResult{}, e
		}
		if tag.RowsAffected() == 0 {
			return models.ShippingOptionRuleBatchResult{}, fmt.Errorf("%w: shipping option rule %s not found", pgx.ErrNoRows, id)
		}
		result.Deleted = append(result.Deleted, id)
	}
	if err = tx.Commit(ctx); err != nil {
		return models.ShippingOptionRuleBatchResult{}, err
	}
	return result, nil
}

func (r *Repository) CompensateShippingOptionRuleBatch(ctx context.Context, result models.ShippingOptionRuleBatchResult) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, rule := range result.Created {
		if _, err = tx.Exec(ctx, `DELETE FROM shipping_option_rule WHERE id=$1`, rule.ID); err != nil {
			return err
		}
	}
	for _, rule := range result.PreviousUpdated {
		if _, err = tx.Exec(ctx, `UPDATE shipping_option_rule SET attribute=$2,operator=$3,value=$4,updated_at=now() WHERE id=$1`, rule.ID, rule.Attribute, rule.Operator, rule.Value); err != nil {
			return err
		}
	}
	for _, rule := range result.DeletedRules {
		if _, err = tx.Exec(ctx, `UPDATE shipping_option_rule SET deleted_at=NULL,updated_at=now() WHERE id=$1`, rule.ID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
