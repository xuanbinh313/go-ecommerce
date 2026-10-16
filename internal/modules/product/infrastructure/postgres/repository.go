package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/product/domain"
	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }
func raw(v json.RawMessage) any {
	if v == nil {
		return json.RawMessage("null")
	}
	return v
}

func (r *Repository) CreateProducts(ctx context.Context, inputs []models.ProductInput) ([]models.Product, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	ids := make([]string, 0, len(inputs))
	for _, in := range inputs {
		id := entityid.New("prod")
		if in.ID != nil {
			id = *in.ID
		}
		handle := in.Title
		if in.Handle != nil {
			handle = *in.Handle
		}
		status := "draft"
		if in.Status != nil {
			status = *in.Status
		}
		gift, discount := false, true
		if in.IsGiftcard != nil {
			gift = *in.IsGiftcard
		}
		if in.Discountable != nil {
			discount = *in.Discountable
		}
		if _, err = tx.Exec(ctx, `INSERT INTO product(id,title,handle,subtitle,description,is_giftcard,status,thumbnail,weight,length,height,width,origin_country,hs_code,mid_code,material,type_id,collection_id,discountable,external_id,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`, id, in.Title, handle, in.Subtitle, in.Description, gift, status, in.Thumbnail, in.Weight, in.Length, in.Height, in.Width, in.OriginCountry, in.HSCode, in.MIDCode, in.Material, in.TypeID, in.CollectionID, discount, in.ExternalID, raw(in.Metadata)); err != nil {
			return nil, err
		}
		ids = append(ids, id)
		for _, option := range in.Options {
			optionID := ""
			if option.ID != nil {
				optionID = *option.ID
			} else {
				optionID = entityid.New("opt")
				exclusive := true
				if option.IsExclusive != nil {
					exclusive = *option.IsExclusive
				}
				if _, err = tx.Exec(ctx, `INSERT INTO product_option(id,title,is_exclusive,metadata) VALUES($1,$2,$3,$4)`, optionID, option.Title, exclusive, raw(option.Metadata)); err != nil {
					return nil, err
				}
				for rank, value := range option.Values {
					if _, err = tx.Exec(ctx, `INSERT INTO product_option_value(id,value,option_id,rank,metadata) VALUES($1,$2,$3,$4,'null')`, entityid.New("optval"), value, optionID, rank); err != nil {
						return nil, err
					}
				}
			}
			joinID := entityid.New("prodopt")
			if _, err = tx.Exec(ctx, `INSERT INTO product_product_option(id,product_id,product_option_id) VALUES($1,$2,$3)`, joinID, id, optionID); err != nil {
				return nil, err
			}
			if len(option.ValueIDs) > 0 {
				var found int
				if err = tx.QueryRow(ctx, `SELECT count(DISTINCT id) FROM product_option_value WHERE id=ANY($1) AND option_id=$2 AND deleted_at IS NULL`, option.ValueIDs, optionID).Scan(&found); err != nil {
					return nil, err
				}
				if found != len(option.ValueIDs) {
					return nil, fmt.Errorf("one or more option values do not belong to option %s", optionID)
				}
				for _, valueID := range option.ValueIDs {
					if _, err = tx.Exec(ctx, `INSERT INTO product_product_option_value(id,product_product_option_id,product_option_value_id) VALUES($1,$2,$3)`, entityid.New("prodoptval"), joinID, valueID); err != nil {
						return nil, err
					}
				}
			} else {
				valueRows, queryErr := tx.Query(ctx, `SELECT id FROM product_option_value WHERE option_id=$1 AND deleted_at IS NULL`, optionID)
				if queryErr != nil {
					return nil, queryErr
				}
				valueIDs := []string{}
				for valueRows.Next() {
					var valueID string
					if queryErr = valueRows.Scan(&valueID); queryErr != nil {
						valueRows.Close()
						return nil, queryErr
					}
					valueIDs = append(valueIDs, valueID)
				}
				queryErr = valueRows.Err()
				valueRows.Close()
				if queryErr != nil {
					return nil, queryErr
				}
				for _, valueID := range valueIDs {
					if _, err = tx.Exec(ctx, `INSERT INTO product_product_option_value(id,product_product_option_id,product_option_value_id) VALUES($1,$2,$3)`, entityid.New("prodoptval"), joinID, valueID); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	out := make([]models.Product, 0, len(ids))
	for _, id := range ids {
		x, e := r.GetProduct(ctx, id)
		if e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, nil
}
func (r *Repository) CreateVariants(ctx context.Context, inputs []models.ProductVariantInput) ([]models.ProductVariant, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := []models.ProductVariant{}
	for _, in := range inputs {
		id := entityid.New("variant")
		if in.ID != nil {
			id = *in.ID
		}
		backorder, manage := false, true
		if in.AllowBackorder != nil {
			backorder = *in.AllowBackorder
		}
		if in.ManageInventory != nil {
			manage = *in.ManageInventory
		}
		var v models.ProductVariant
		err = tx.QueryRow(ctx, `INSERT INTO product_variant(id,title,sku,barcode,ean,upc,allow_backorder,manage_inventory,hs_code,origin_country,mid_code,material,weight,length,height,width,metadata,variant_rank,thumbnail,product_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20) RETURNING id,title,sku,barcode,ean,upc,allow_backorder,manage_inventory,hs_code,origin_country,mid_code,material,weight,length,height,width,metadata,variant_rank,thumbnail,product_id,created_at,updated_at,deleted_at`, id, in.Title, in.SKU, in.Barcode, in.EAN, in.UPC, backorder, manage, in.HSCode, in.OriginCountry, in.MIDCode, in.Material, in.Weight, in.Length, in.Height, in.Width, raw(in.Metadata), in.VariantRank, in.Thumbnail, in.ProductID).Scan(&v.ID, &v.Title, &v.SKU, &v.Barcode, &v.EAN, &v.UPC, &v.AllowBackorder, &v.ManageInventory, &v.HSCode, &v.OriginCountry, &v.MIDCode, &v.Material, &v.Weight, &v.Length, &v.Height, &v.Width, &v.Metadata, &v.VariantRank, &v.Thumbnail, &v.ProductID, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
		if err != nil {
			return nil, err
		}
		for name, value := range in.Options {
			var optionValueID string
			err = tx.QueryRow(ctx, `SELECT pov.id FROM product_product_option ppo JOIN product_option po ON po.id=ppo.product_option_id JOIN product_product_option_value ppov ON ppov.product_product_option_id=ppo.id JOIN product_option_value pov ON pov.id=ppov.product_option_value_id WHERE ppo.product_id=$1 AND po.title=$2 AND pov.value=$3 AND ppo.deleted_at IS NULL AND ppov.deleted_at IS NULL AND po.deleted_at IS NULL AND pov.deleted_at IS NULL`, in.ProductID, name, value).Scan(&optionValueID)
			if err != nil {
				return nil, fmt.Errorf("product option %s=%s: %w", name, value, err)
			}
			if _, err = tx.Exec(ctx, `INSERT INTO product_variant_option(variant_id,option_value_id) VALUES($1,$2)`, id, optionValueID); err != nil {
				return nil, err
			}
		}
		out = append(out, v)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
func (r *Repository) GetProduct(ctx context.Context, id string) (models.Product, error) {
	var p models.Product
	err := r.db.QueryRow(ctx, `SELECT id,title,handle,subtitle,description,is_giftcard,status,thumbnail,weight,length,height,width,origin_country,hs_code,mid_code,material,type_id,collection_id,discountable,external_id,metadata,created_at,updated_at,deleted_at FROM product WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&p.ID, &p.Title, &p.Handle, &p.Subtitle, &p.Description, &p.IsGiftcard, &p.Status, &p.Thumbnail, &p.Weight, &p.Length, &p.Height, &p.Width, &p.OriginCountry, &p.HSCode, &p.MIDCode, &p.Material, &p.TypeID, &p.CollectionID, &p.Discountable, &p.ExternalID, &p.Metadata, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
	if err != nil {
		return p, err
	}
	p.Options, err = r.productOptions(ctx, id)
	if err != nil {
		return p, err
	}
	p.Variants, err = r.productVariants(ctx, id)
	return p, err
}

func (r *Repository) UpdateProduct(ctx context.Context, id string, fields map[string]any) (models.Product, error) {
	columns := map[string]string{"title": "title", "handle": "handle", "subtitle": "subtitle", "description": "description", "is_giftcard": "is_giftcard", "status": "status", "thumbnail": "thumbnail", "weight": "weight", "length": "length", "height": "height", "width": "width", "origin_country": "origin_country", "hs_code": "hs_code", "mid_code": "mid_code", "material": "material", "type_id": "type_id", "collection_id": "collection_id", "discountable": "discountable", "external_id": "external_id", "metadata": "metadata"}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		if _, ok := columns[key]; !ok {
			return models.Product{}, fmt.Errorf("unsupported product update field %q", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	assignments := make([]string, 0, len(keys)+1)
	args := []any{id}
	for _, key := range keys {
		value := fields[key]
		if key == "metadata" && value != nil {
			encoded, err := json.Marshal(value)
			if err != nil {
				return models.Product{}, fmt.Errorf("encode product metadata: %w", err)
			}
			value = encoded
		}
		args = append(args, value)
		assignments = append(assignments, columns[key]+fmt.Sprintf("=$%d", len(args)))
	}
	if len(assignments) > 0 {
		assignments = append(assignments, "updated_at=now()")
		result, err := r.db.Exec(ctx, `UPDATE product SET `+strings.Join(assignments, ",")+` WHERE id=$1 AND deleted_at IS NULL`, args...)
		if err != nil {
			return models.Product{}, err
		}
		if result.RowsAffected() == 0 {
			return models.Product{}, pgx.ErrNoRows
		}
	}
	return r.GetProduct(ctx, id)
}
func (r *Repository) productOptions(ctx context.Context, id string) ([]models.ProductOption, error) {
	rows, err := r.db.Query(ctx, `SELECT po.id,ppo.product_id,po.title,po.is_exclusive,po.metadata,po.created_at,po.updated_at FROM product_product_option ppo JOIN product_option po ON po.id=ppo.product_option_id WHERE ppo.product_id=$1 AND ppo.deleted_at IS NULL AND po.deleted_at IS NULL ORDER BY ppo.created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ProductOption{}
	for rows.Next() {
		var x models.ProductOption
		if err := rows.Scan(&x.ID, &x.ProductID, &x.Title, &x.IsExclusive, &x.Metadata, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		x.Values = []models.ProductOptionValue{}
		vals, e := r.db.Query(ctx, `SELECT pov.id,pov.value,pov.option_id,pov.rank,pov.metadata FROM product_product_option_value ppov JOIN product_product_option ppo ON ppo.id=ppov.product_product_option_id JOIN product_option_value pov ON pov.id=ppov.product_option_value_id WHERE ppo.product_id=$1 AND ppo.product_option_id=$2 AND ppov.deleted_at IS NULL AND pov.deleted_at IS NULL ORDER BY pov.rank,pov.created_at`, id, x.ID)
		if e != nil {
			return nil, e
		}
		for vals.Next() {
			var v models.ProductOptionValue
			if e = vals.Scan(&v.ID, &v.Value, &v.OptionID, &v.Rank, &v.Metadata); e != nil {
				vals.Close()
				return nil, e
			}
			x.Values = append(x.Values, v)
		}
		if e = vals.Err(); e != nil {
			vals.Close()
			return nil, e
		}
		vals.Close()
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *Repository) ListProductOptions(ctx context.Context, filter models.ProductOptionFilter, limit, offset int) ([]models.ProductOption, int, error) {
	where := []string{}
	if len(filter.DeletedAt) == 0 {
		where = append(where, "deleted_at IS NULL")
	}
	args := []any{}
	appendArray := func(column string, values []string) {
		if len(values) == 0 {
			return
		}
		args = append(args, values)
		where = append(where, fmt.Sprintf("%s=ANY($%d::text[])", column, len(args)))
	}
	appendArray("id", filter.IDs)
	appendArray("title", filter.Titles)
	if filter.IsExclusive != nil {
		args = append(args, *filter.IsExclusive)
		where = append(where, fmt.Sprintf("is_exclusive=$%d", len(args)))
	}
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		where = append(where, fmt.Sprintf("title ILIKE $%d", len(args)))
	}
	for _, dateFilter := range []struct {
		column string
		values map[string]string
	}{{"created_at", filter.CreatedAt}, {"updated_at", filter.UpdatedAt}, {"deleted_at", filter.DeletedAt}} {
		for operator, value := range dateFilter.values {
			sqlOperator, ok := map[string]string{"$eq": "=", "$ne": "<>", "$gt": ">", "$gte": ">=", "$lt": "<", "$lte": "<="}[operator]
			if !ok {
				return nil, 0, fmt.Errorf("unsupported %s filter operator %q", dateFilter.column, operator)
			}
			if dateFilter.column == "deleted_at" && value == "null" {
				if operator == "$eq" {
					where = append(where, "deleted_at IS NULL")
					continue
				}
				if operator == "$ne" {
					where = append(where, "deleted_at IS NOT NULL")
					continue
				}
				return nil, 0, fmt.Errorf("unsupported deleted_at null comparison %q", operator)
			}
			args = append(args, value)
			where = append(where, fmt.Sprintf("%s %s $%d::timestamptz", dateFilter.column, sqlOperator, len(args)))
		}
	}
	if len(where) == 0 {
		where = append(where, "TRUE")
	}
	whereSQL := strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM product_option WHERE `+whereSQL, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, `SELECT id,title,is_exclusive,metadata,created_at,updated_at,deleted_at FROM product_option WHERE `+whereSQL+fmt.Sprintf(" ORDER BY created_at,id LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.ProductOption{}
	for rows.Next() {
		var item models.ProductOption
		if err := rows.Scan(&item.ID, &item.Title, &item.IsExclusive, &item.Metadata, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt); err != nil {
			return nil, 0, err
		}
		item.Values, err = r.productOptionValues(ctx, item.ID)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, count, rows.Err()
}

func (r *Repository) GetProductOption(ctx context.Context, id string) (models.ProductOption, error) {
	var item models.ProductOption
	err := r.db.QueryRow(ctx, `SELECT id,title,is_exclusive,metadata,created_at,updated_at,deleted_at FROM product_option WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&item.ID, &item.Title, &item.IsExclusive, &item.Metadata, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt)
	if err != nil {
		return item, err
	}
	item.Values, err = r.productOptionValues(ctx, id)
	return item, err
}

func (r *Repository) CreateProductOptions(ctx context.Context, inputs []models.CreateProductOptionInput) ([]models.ProductOption, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	ids := make([]string, 0, len(inputs))
	for _, input := range inputs {
		id := entityid.New("opt")
		exclusive := false
		if input.IsExclusive != nil {
			exclusive = *input.IsExclusive
		}
		if _, err := tx.Exec(ctx, `INSERT INTO product_option(id,title,is_exclusive,metadata) VALUES($1,$2,$3,$4)`, id, input.Title, exclusive, raw(input.Metadata)); err != nil {
			return nil, err
		}
		ids = append(ids, id)
		for _, value := range input.Values {
			var rank *int
			if valueRank, ok := input.Ranks[value]; ok {
				value := valueRank
				rank = &value
			}
			if _, err := tx.Exec(ctx, `INSERT INTO product_option_value(id,value,option_id,rank,metadata) VALUES($1,$2,$3,$4,'null')`, entityid.New("optval"), value, id, rank); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	items := make([]models.ProductOption, 0, len(ids))
	for _, id := range ids {
		item, err := r.GetProductOption(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Repository) HardDeleteProductOptions(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM product_option WHERE id=ANY($1::text[])`, ids)
	return err
}

func (r *Repository) UpdateProductOption(ctx context.Context, id string, fields map[string]any) (models.ProductOption, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.ProductOption{}, err
	}
	defer tx.Rollback(ctx)
	var currentExclusive bool
	if err = tx.QueryRow(ctx, `SELECT is_exclusive FROM product_option WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&currentExclusive); err != nil {
		return models.ProductOption{}, err
	}
	if requested, ok := fields["is_exclusive"].(bool); ok && requested && !currentExclusive {
		return models.ProductOption{}, fmt.Errorf("%w: cannot change a global product option to exclusive", domain.ErrInvalidData)
	}
	columns := map[string]string{"title": "title", "is_exclusive": "is_exclusive", "metadata": "metadata"}
	keys := []string{}
	for key := range fields {
		if _, ok := columns[key]; ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	sets := []string{}
	args := []any{id}
	for _, key := range keys {
		value := fields[key]
		if key == "metadata" && value != nil {
			encoded, e := json.Marshal(value)
			if e != nil {
				return models.ProductOption{}, e
			}
			value = encoded
		}
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s=$%d", columns[key], len(args)))
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		if _, err = tx.Exec(ctx, `UPDATE product_option SET `+strings.Join(sets, ",")+` WHERE id=$1`, args...); err != nil {
			return models.ProductOption{}, err
		}
	}
	if rawValues, exists := fields["values"]; exists {
		values, ok := rawValues.([]any)
		if !ok {
			return models.ProductOption{}, fmt.Errorf("%w: product option values must be an array", domain.ErrInvalidData)
		}
		newValues := make([]string, 0, len(values))
		seen := map[string]bool{}
		for _, raw := range values {
			value, ok := raw.(string)
			if !ok {
				return models.ProductOption{}, fmt.Errorf("%w: product option values must be strings", domain.ErrInvalidData)
			}
			if seen[value] {
				return models.ProductOption{}, fmt.Errorf("%w: duplicate product option value %q", domain.ErrInvalidData, value)
			}
			seen[value] = true
			newValues = append(newValues, value)
		}
		var rows pgx.Rows
		rows, err = tx.Query(ctx, `SELECT id,value,rank FROM product_option_value WHERE option_id=$1 AND deleted_at IS NULL FOR UPDATE`, id)
		if err != nil {
			return models.ProductOption{}, err
		}
		type existingValue struct {
			id    string
			value string
			rank  *int
		}
		existing := map[string]existingValue{}
		removedIDs := []string{}
		for rows.Next() {
			var item existingValue
			if err = rows.Scan(&item.id, &item.value, &item.rank); err != nil {
				rows.Close()
				return models.ProductOption{}, err
			}
			existing[item.value] = item
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return models.ProductOption{}, err
		}
		rows.Close()
		for value, item := range existing {
			if !seen[value] {
				removedIDs = append(removedIDs, item.id)
			}
		}
		if len(removedIDs) > 0 {
			var associated bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM product_product_option_value ppov JOIN product_product_option ppo ON ppo.id=ppov.product_product_option_id WHERE ppov.product_option_value_id=ANY($1::text[]) AND ppov.deleted_at IS NULL AND ppo.deleted_at IS NULL)`, removedIDs).Scan(&associated); err != nil {
				return models.ProductOption{}, err
			}
			if associated {
				return models.ProductOption{}, fmt.Errorf("%w: cannot remove product option values associated with products", domain.ErrInvalidData)
			}
			if _, err = tx.Exec(ctx, `UPDATE product_option_value SET deleted_at=now(),updated_at=now() WHERE id=ANY($1::text[])`, removedIDs); err != nil {
				return models.ProductOption{}, err
			}
		}
		ranks, err := optionRanks(fields["ranks"])
		if err != nil {
			return models.ProductOption{}, err
		}
		for rankValue := range ranks {
			if !seen[rankValue] {
				return models.ProductOption{}, fmt.Errorf("%w: rank provided for missing product option value %q", domain.ErrInvalidData, rankValue)
			}
		}
		for _, value := range newValues {
			if old, ok := existing[value]; ok {
				if rank, supplied := ranks[value]; supplied {
					if _, err = tx.Exec(ctx, `UPDATE product_option_value SET rank=$2,updated_at=now() WHERE id=$1`, old.id, rank); err != nil {
						return models.ProductOption{}, err
					}
				}
				continue
			}
			var rank *int
			if n, ok := ranks[value]; ok {
				valueRank := n
				rank = &valueRank
			}
			if _, err = tx.Exec(ctx, `INSERT INTO product_option_value(id,value,option_id,rank,metadata) VALUES($1,$2,$3,$4,'null')`, entityid.New("optval"), value, id, rank); err != nil {
				return models.ProductOption{}, err
			}
		}
	} else if rawRanks, exists := fields["ranks"]; exists {
		ranks, rankErr := optionRanks(rawRanks)
		if rankErr != nil {
			return models.ProductOption{}, rankErr
		}
		for value, rank := range ranks {
			command, e := tx.Exec(ctx, `UPDATE product_option_value SET rank=$3,updated_at=now() WHERE option_id=$1 AND value=$2 AND deleted_at IS NULL`, id, value, rank)
			if e != nil {
				return models.ProductOption{}, e
			}
			if command.RowsAffected() == 0 {
				return models.ProductOption{}, fmt.Errorf("%w: rank provided for unknown product option value %q", domain.ErrInvalidData, value)
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.ProductOption{}, err
	}
	return r.GetProductOption(ctx, id)
}

func optionRanks(raw any) (map[string]int, error) {
	if raw == nil {
		return map[string]int{}, nil
	}
	values, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: product option ranks must be an object", domain.ErrInvalidData)
	}
	out := make(map[string]int, len(values))
	for key, value := range values {
		switch n := value.(type) {
		case float64:
			out[key] = int(n)
		case int:
			out[key] = n
		default:
			return nil, fmt.Errorf("%w: rank for %q must be an integer", domain.ErrInvalidData, key)
		}
	}
	return out, nil
}

func (r *Repository) RestoreProductOptionSnapshot(ctx context.Context, previous models.ProductOption) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE product_option SET title=$2,is_exclusive=$3,metadata=$4,updated_at=now() WHERE id=$1`, previous.ID, previous.Title, previous.IsExclusive, raw(previous.Metadata)); err != nil {
		return err
	}
	keep := make([]string, 0, len(previous.Values))
	for _, value := range previous.Values {
		keep = append(keep, value.ID)
	}
	if _, err = tx.Exec(ctx, `UPDATE product_option_value SET deleted_at=now(),updated_at=now() WHERE option_id=$1 AND deleted_at IS NULL AND NOT(id=ANY($2::text[]))`, previous.ID, keep); err != nil {
		return err
	}
	for _, value := range previous.Values {
		if _, err = tx.Exec(ctx, `UPDATE product_option_value SET value=$2,rank=$3,metadata=$4,deleted_at=NULL,updated_at=now() WHERE id=$1`, value.ID, value.Value, value.Rank, raw(value.Metadata)); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) SoftDeleteProductOptions(ctx context.Context, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var associated bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM product_product_option WHERE product_option_id=ANY($1::text[]) AND deleted_at IS NULL)`, ids).Scan(&associated); err != nil {
		return err
	}
	if associated {
		return fmt.Errorf("%w: cannot delete product options associated with products", domain.ErrInvalidData)
	}
	if _, err = tx.Exec(ctx, `UPDATE product_option_value SET deleted_at=now(),updated_at=now() WHERE option_id=ANY($1::text[]) AND deleted_at IS NULL`, ids); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE product_option SET deleted_at=now(),updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NULL`, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) RestoreProductOptions(ctx context.Context, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE product_option SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NOT NULL`, ids); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE product_option_value SET deleted_at=NULL,updated_at=now() WHERE option_id=ANY($1::text[]) AND deleted_at IS NOT NULL`, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) productOptionValues(ctx context.Context, optionID string) ([]models.ProductOptionValue, error) {
	rows, err := r.db.Query(ctx, `SELECT id,value,option_id,rank,metadata FROM product_option_value WHERE option_id=$1 AND deleted_at IS NULL ORDER BY rank,created_at,id`, optionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []models.ProductOptionValue{}
	for rows.Next() {
		var item models.ProductOptionValue
		if err := rows.Scan(&item.ID, &item.Value, &item.OptionID, &item.Rank, &item.Metadata); err != nil {
			return nil, err
		}
		values = append(values, item)
	}
	return values, rows.Err()
}
func (r *Repository) productVariants(ctx context.Context, id string) ([]models.ProductVariant, error) {
	rows, err := r.db.Query(ctx, `SELECT id,title,sku,barcode,ean,upc,allow_backorder,manage_inventory,hs_code,origin_country,mid_code,material,weight,length,height,width,metadata,variant_rank,thumbnail,product_id,created_at,updated_at,deleted_at FROM product_variant WHERE product_id=$1 AND deleted_at IS NULL ORDER BY variant_rank,created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ProductVariant{}
	for rows.Next() {
		var v models.ProductVariant
		if err := rows.Scan(&v.ID, &v.Title, &v.SKU, &v.Barcode, &v.EAN, &v.UPC, &v.AllowBackorder, &v.ManageInventory, &v.HSCode, &v.OriginCountry, &v.MIDCode, &v.Material, &v.Weight, &v.Length, &v.Height, &v.Width, &v.Metadata, &v.VariantRank, &v.Thumbnail, &v.ProductID, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt); err != nil {
			return nil, err
		}
		v.Options, err = r.productVariantOptions(ctx, v.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) ListProductVariants(ctx context.Context, productID string, filter models.ProductVariantFilter, limit, offset int) ([]models.ProductVariant, int, error) {
	where := []string{}
	if len(filter.DeletedAt) == 0 {
		where = append(where, "deleted_at IS NULL")
	}
	args := []any{}
	if productID != "" {
		args = append(args, productID)
		where = append(where, fmt.Sprintf("product_id=$%d", len(args)))
	}
	appendStringFilter := func(column string, values []string) {
		if len(values) == 0 {
			return
		}
		args = append(args, values)
		where = append(where, fmt.Sprintf("%s=ANY($%d::text[])", column, len(args)))
	}
	appendStringFilter("id", filter.IDs)
	appendStringFilter("product_id", filter.ProductIDs)
	appendStringFilter("sku", filter.SKUs)
	appendStringFilter("ean", filter.EANs)
	appendStringFilter("upc", filter.UPCs)
	appendStringFilter("barcode", filter.Barcodes)
	if filter.PublishedOnly {
		where = append(where, "EXISTS (SELECT 1 FROM product p WHERE p.id=product_variant.product_id AND p.status='published' AND p.deleted_at IS NULL)")
	}
	if len(filter.SalesChannelIDs) > 0 {
		args = append(args, filter.SalesChannelIDs)
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM product_sales_channel psc WHERE psc.product_id=product_variant.product_id AND psc.sales_channel_id=ANY($%d::text[]) AND psc.deleted_at IS NULL)", len(args)))
	}
	if filter.OptionValue != nil || filter.OptionID != nil {
		conditions := []string{"pvo.variant_id=product_variant.id", "pov.id=pvo.option_value_id", "pov.deleted_at IS NULL"}
		if filter.OptionValue != nil {
			args = append(args, *filter.OptionValue)
			conditions = append(conditions, fmt.Sprintf("pov.value=$%d", len(args)))
		}
		if filter.OptionID != nil {
			args = append(args, *filter.OptionID)
			conditions = append(conditions, fmt.Sprintf("pov.option_id=$%d", len(args)))
		}
		where = append(where, "EXISTS (SELECT 1 FROM product_variant_option pvo JOIN product_option_value pov ON pov.id=pvo.option_value_id WHERE "+strings.Join(conditions, " AND ")+")")
	}
	if filter.ManageInventory != nil {
		args = append(args, *filter.ManageInventory)
		where = append(where, fmt.Sprintf("manage_inventory=$%d", len(args)))
	}
	if filter.AllowBackorder != nil {
		args = append(args, *filter.AllowBackorder)
		where = append(where, fmt.Sprintf("allow_backorder=$%d", len(args)))
	}
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		parameter := fmt.Sprintf("$%d", len(args))
		where = append(where, "(title ILIKE "+parameter+" OR COALESCE(sku,'') ILIKE "+parameter+" OR COALESCE(ean,'') ILIKE "+parameter+" OR COALESCE(upc,'') ILIKE "+parameter+" OR COALESCE(barcode,'') ILIKE "+parameter+")")
	}
	for _, dateFilter := range []struct {
		column string
		values map[string]string
	}{{"created_at", filter.CreatedAt}, {"updated_at", filter.UpdatedAt}, {"deleted_at", filter.DeletedAt}} {
		for operator, value := range dateFilter.values {
			sqlOperator, ok := map[string]string{"$eq": "=", "$ne": "<>", "$gt": ">", "$gte": ">=", "$lt": "<", "$lte": "<="}[operator]
			if !ok {
				return nil, 0, fmt.Errorf("unsupported %s filter operator %q", dateFilter.column, operator)
			}
			if dateFilter.column == "deleted_at" && value == "null" {
				if operator == "$eq" {
					where = append(where, "deleted_at IS NULL")
					continue
				}
				if operator == "$ne" {
					where = append(where, "deleted_at IS NOT NULL")
					continue
				}
				return nil, 0, fmt.Errorf("unsupported deleted_at null comparison %q", operator)
			}
			args = append(args, value)
			where = append(where, fmt.Sprintf("%s %s $%d::timestamptz", dateFilter.column, sqlOperator, len(args)))
		}
	}
	if len(where) == 0 {
		where = append(where, "TRUE")
	}
	whereSQL := strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM product_variant WHERE `+whereSQL, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, `SELECT id,title,sku,barcode,ean,upc,allow_backorder,manage_inventory,hs_code,origin_country,mid_code,material,weight,length,height,width,metadata,variant_rank,thumbnail,product_id,created_at,updated_at,deleted_at FROM product_variant WHERE `+whereSQL+fmt.Sprintf(" ORDER BY variant_rank,created_at,id LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.ProductVariant{}
	for rows.Next() {
		var item models.ProductVariant
		if err := rows.Scan(&item.ID, &item.Title, &item.SKU, &item.Barcode, &item.EAN, &item.UPC, &item.AllowBackorder, &item.ManageInventory, &item.HSCode, &item.OriginCountry, &item.MIDCode, &item.Material, &item.Weight, &item.Length, &item.Height, &item.Width, &item.Metadata, &item.VariantRank, &item.Thumbnail, &item.ProductID, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt); err != nil {
			return nil, 0, err
		}
		item.Options, err = r.productVariantOptions(ctx, item.ID)
		if err != nil {
			return nil, 0, err
		}
		if item.ProductID != nil {
			var product models.ProductVariantProduct
			if err := r.db.QueryRow(ctx, `SELECT id,type_id FROM product WHERE id=$1 AND deleted_at IS NULL`, *item.ProductID).Scan(&product.ID, &product.TypeID); err == nil {
				item.Product = &product
			}
		}
		items = append(items, item)
	}
	return items, count, rows.Err()
}

func (r *Repository) productVariantOptions(ctx context.Context, variantID string) ([]models.ProductOptionValue, error) {
	rows, err := r.db.Query(ctx, `SELECT pov.id,pov.value,pov.option_id,pov.rank,pov.metadata FROM product_variant_option pvo JOIN product_option_value pov ON pov.id=pvo.option_value_id WHERE pvo.variant_id=$1 AND pov.deleted_at IS NULL ORDER BY pov.rank,pov.id`, variantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	options := []models.ProductOptionValue{}
	for rows.Next() {
		var option models.ProductOptionValue
		if err := rows.Scan(&option.ID, &option.Value, &option.OptionID, &option.Rank, &option.Metadata); err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	return options, rows.Err()
}
func (r *Repository) ListProducts(ctx context.Context, ids []string) ([]models.Product, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM product WHERE deleted_at IS NULL AND (cardinality($1::text[])=0 OR id=ANY($1)) ORDER BY created_at`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Product{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		p, err := r.GetProduct(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) ListAdminProducts(ctx context.Context, ids []string, isGiftcard *bool, limit, offset int) ([]models.Product, int, error) {
	var giftcard any
	if isGiftcard != nil {
		giftcard = *isGiftcard
	}
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM product WHERE deleted_at IS NULL AND (cardinality($1::text[])=0 OR id=ANY($1)) AND ($2::boolean IS NULL OR is_giftcard=$2)`, ids, giftcard).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT id FROM product WHERE deleted_at IS NULL AND (cardinality($1::text[])=0 OR id=ANY($1)) AND ($2::boolean IS NULL OR is_giftcard=$2) ORDER BY created_at DESC,id LIMIT $3 OFFSET $4`, ids, giftcard, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	products := []models.Product{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, 0, err
		}
		product, err := r.GetProduct(ctx, id)
		if err != nil {
			return nil, 0, err
		}
		products = append(products, product)
	}
	return products, count, rows.Err()
}

func (r *Repository) ListProductTypes(ctx context.Context, ids []string, limit, offset int) ([]models.ProductType, int, error) {
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM product_type WHERE deleted_at IS NULL AND (cardinality($1::text[])=0 OR id=ANY($1))`, ids).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT id,value,external_id,metadata,created_at,updated_at,deleted_at FROM product_type WHERE deleted_at IS NULL AND (cardinality($1::text[])=0 OR id=ANY($1)) ORDER BY created_at,id LIMIT $2 OFFSET $3`, ids, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.ProductType{}
	for rows.Next() {
		var item models.ProductType
		if err := rows.Scan(&item.ID, &item.Value, &item.ExternalID, &item.Metadata, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, count, rows.Err()
}

func (r *Repository) GetProductType(ctx context.Context, id string) (models.ProductType, error) {
	var item models.ProductType
	err := r.db.QueryRow(ctx, `SELECT id,value,external_id,metadata,created_at,updated_at,deleted_at FROM product_type WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&item.ID, &item.Value, &item.ExternalID, &item.Metadata, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt)
	return item, err
}

func (r *Repository) CreateProductTypes(ctx context.Context, inputs []models.ProductTypeInput) ([]models.ProductType, error) {
	items := make([]models.ProductType, 0, len(inputs))
	for _, input := range inputs {
		id := entityid.New("ptyp")
		var item models.ProductType
		err := r.db.QueryRow(ctx, `INSERT INTO product_type(id,value,external_id,metadata) VALUES($1,$2,$3,$4) RETURNING id,value,external_id,metadata,created_at,updated_at,deleted_at`, id, input.Value, input.ExternalID, raw(input.Metadata)).Scan(&item.ID, &item.Value, &item.ExternalID, &item.Metadata, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Repository) UpdateProductType(ctx context.Context, id string, fields map[string]any) (models.ProductType, error) {
	if err := updateProductCatalogEntity(ctx, r, "product_type", id, fields); err != nil {
		return models.ProductType{}, err
	}
	return r.GetProductType(ctx, id)
}

func (r *Repository) SoftDeleteProductTypes(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE product_type SET deleted_at=now(),updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NULL`, ids)
	return err
}
func (r *Repository) DetachProductTypeReferences(ctx context.Context, ids []string) ([]models.ProductTypeProductReference, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id,type_id FROM product WHERE type_id=ANY($1::text[]) AND deleted_at IS NULL FOR UPDATE`, ids)
	if err != nil {
		return nil, err
	}
	refs := []models.ProductTypeProductReference{}
	for rows.Next() {
		var ref models.ProductTypeProductReference
		if err := rows.Scan(&ref.ProductID, &ref.TypeID); err != nil {
			rows.Close()
			return nil, err
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if _, err = tx.Exec(ctx, `UPDATE product SET type_id=NULL,updated_at=now() WHERE type_id=ANY($1::text[]) AND deleted_at IS NULL`, ids); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return refs, nil
}
func (r *Repository) RestoreProductTypeReferences(ctx context.Context, refs []models.ProductTypeProductReference) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, ref := range refs {
		if _, err := tx.Exec(ctx, `UPDATE product SET type_id=$2,updated_at=now() WHERE id=$1 AND type_id IS NULL AND deleted_at IS NULL`, ref.ProductID, ref.TypeID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (r *Repository) RestoreProductTypes(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE product_type SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NOT NULL`, ids)
	return err
}
func (r *Repository) HardDeleteProductTypes(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM product_type WHERE id=ANY($1::text[])`, ids)
	return err
}

func (r *Repository) ListProductTags(ctx context.Context, ids []string, limit, offset int) ([]models.ProductTag, int, error) {
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM product_tag WHERE deleted_at IS NULL AND (cardinality($1::text[])=0 OR id=ANY($1))`, ids).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT id,value,external_id,metadata,created_at,updated_at,deleted_at FROM product_tag WHERE deleted_at IS NULL AND (cardinality($1::text[])=0 OR id=ANY($1)) ORDER BY created_at,id LIMIT $2 OFFSET $3`, ids, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.ProductTag{}
	for rows.Next() {
		var item models.ProductTag
		if err := rows.Scan(&item.ID, &item.Value, &item.ExternalID, &item.Metadata, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, count, rows.Err()
}

func (r *Repository) GetProductTag(ctx context.Context, id string) (models.ProductTag, error) {
	var item models.ProductTag
	err := r.db.QueryRow(ctx, `SELECT id,value,external_id,metadata,created_at,updated_at,deleted_at FROM product_tag WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&item.ID, &item.Value, &item.ExternalID, &item.Metadata, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt)
	return item, err
}

func (r *Repository) CreateProductTags(ctx context.Context, inputs []models.ProductTagInput) ([]models.ProductTag, error) {
	items := make([]models.ProductTag, 0, len(inputs))
	for _, input := range inputs {
		id := entityid.New("ptag")
		var item models.ProductTag
		err := r.db.QueryRow(ctx, `INSERT INTO product_tag(id,value,external_id,metadata) VALUES($1,$2,$3,$4) RETURNING id,value,external_id,metadata,created_at,updated_at,deleted_at`, id, input.Value, input.ExternalID, raw(input.Metadata)).Scan(&item.ID, &item.Value, &item.ExternalID, &item.Metadata, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Repository) UpdateProductTag(ctx context.Context, id string, fields map[string]any) (models.ProductTag, error) {
	if err := updateProductCatalogEntity(ctx, r, "product_tag", id, fields); err != nil {
		return models.ProductTag{}, err
	}
	return r.GetProductTag(ctx, id)
}

func (r *Repository) SoftDeleteProductTags(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE product_tag SET deleted_at=now(),updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NULL`, ids)
	return err
}
func (r *Repository) RestoreProductTags(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE product_tag SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NOT NULL`, ids)
	return err
}
func (r *Repository) HardDeleteProductTags(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM product_tag WHERE id=ANY($1::text[])`, ids)
	return err
}

func updateProductCatalogEntity(ctx context.Context, r *Repository, table, id string, fields map[string]any) error {
	columns := map[string]bool{"value": true, "external_id": true, "metadata": true}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		if !columns[key] {
			return fmt.Errorf("unsupported %s field %q", table, key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	sets := make([]string, 0, len(keys)+1)
	args := []any{id}
	for _, key := range keys {
		value := fields[key]
		if key == "metadata" && value != nil {
			encoded, err := json.Marshal(value)
			if err != nil {
				return err
			}
			value = encoded
		}
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s=$%d", key, len(args)))
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at=now()")
	command, err := r.db.Exec(ctx, `UPDATE `+table+` SET `+strings.Join(sets, ",")+` WHERE id=$1 AND deleted_at IS NULL`, args...)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteProducts(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE product SET deleted_at=now(),updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NULL`, ids)
	return err
}

func (r *Repository) RestoreProducts(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE product SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NOT NULL`, ids)
	return err
}
func (r *Repository) DeleteVariants(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM product_variant WHERE id=ANY($1)`, ids)
	return err
}

const categoryColumns = `id,name,description,handle,mpath,is_active,is_internal,rank,external_id,metadata,parent_category_id,created_at,updated_at,deleted_at`

func scanProductCategory(row pgx.Row) (models.ProductCategory, error) {
	var v models.ProductCategory
	err := row.Scan(&v.ID, &v.Name, &v.Description, &v.Handle, &v.MPath, &v.IsActive, &v.IsInternal, &v.Rank, &v.ExternalID, &v.Metadata, &v.ParentCategoryID, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	return v, err
}

func (r *Repository) ListProductCategories(ctx context.Context, ids []string, limit, offset int) ([]models.ProductCategory, int, error) {
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM product_category WHERE deleted_at IS NULL AND (cardinality($1::text[])=0 OR id=ANY($1))`, ids).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT `+categoryColumns+` FROM product_category WHERE deleted_at IS NULL AND (cardinality($1::text[])=0 OR id=ANY($1)) ORDER BY rank,created_at,id LIMIT $2 OFFSET $3`, ids, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.ProductCategory{}
	for rows.Next() {
		v, e := scanProductCategory(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) GetProductCategory(ctx context.Context, id string) (models.ProductCategory, error) {
	return scanProductCategory(r.db.QueryRow(ctx, `SELECT `+categoryColumns+` FROM product_category WHERE id=$1 AND deleted_at IS NULL`, id))
}
func (r *Repository) CreateProductCategories(ctx context.Context, inputs []models.ProductCategoryInput) ([]models.ProductCategory, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := make([]models.ProductCategory, 0, len(inputs))
	for _, in := range inputs {
		id := entityid.New("pcat")
		handle := in.Name
		if in.Handle != nil {
			handle = *in.Handle
		}
		description := ""
		if in.Description != nil {
			description = *in.Description
		}
		active, internal := false, false
		if in.IsActive != nil {
			active = *in.IsActive
		}
		if in.IsInternal != nil {
			internal = *in.IsInternal
		}
		rank := float64(0)
		if in.Rank != nil {
			rank = *in.Rank
		}
		path := id
		if in.ParentCategoryID != nil {
			var parentPath string
			if err = tx.QueryRow(ctx, `SELECT mpath FROM product_category WHERE id=$1 AND deleted_at IS NULL`, *in.ParentCategoryID).Scan(&parentPath); err != nil {
				return nil, err
			}
			path = parentPath + "." + id
		}
		v, e := scanProductCategory(tx.QueryRow(ctx, `INSERT INTO product_category(id,name,description,handle,mpath,is_active,is_internal,rank,external_id,metadata,parent_category_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING `+categoryColumns, id, in.Name, description, handle, path, active, internal, rank, in.ExternalID, raw(in.Metadata), in.ParentCategoryID))
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
func (r *Repository) UpdateProductCategory(ctx context.Context, id string, fields map[string]any) (models.ProductCategory, error) {
	allowed := map[string]string{"name": "name", "description": "description", "handle": "handle", "is_active": "is_active", "is_internal": "is_internal", "rank": "rank", "external_id": "external_id", "metadata": "metadata", "parent_category_id": "parent_category_id"}
	sets := []string{}
	args := []any{id}
	for k, v := range fields {
		col, ok := allowed[k]
		if !ok {
			continue
		}
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s=$%d", col, len(args)))
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		tag, err := r.db.Exec(ctx, `UPDATE product_category SET `+strings.Join(sets, ",")+` WHERE id=$1 AND deleted_at IS NULL`, args...)
		if err != nil {
			return models.ProductCategory{}, err
		}
		if tag.RowsAffected() == 0 {
			return models.ProductCategory{}, pgx.ErrNoRows
		}
	}
	return r.GetProductCategory(ctx, id)
}
func (r *Repository) SoftDeleteProductCategories(ctx context.Context, ids []string) error {
	var children int
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM product_category c WHERE c.deleted_at IS NULL AND c.parent_category_id=ANY($1::text[]) AND NOT(c.id=ANY($1::text[]))`, ids).Scan(&children)
	if err != nil {
		return err
	}
	if children > 0 {
		return fmt.Errorf("product category has child categories")
	}
	_, err = r.db.Exec(ctx, `UPDATE product_category SET deleted_at=now(),updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NULL`, ids)
	return err
}
func (r *Repository) RestoreProductCategories(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE product_category SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NOT NULL`, ids)
	return err
}
func (r *Repository) LinkProductsToCategory(ctx context.Context, id string, add, remove []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if len(remove) > 0 {
		if _, err = tx.Exec(ctx, `DELETE FROM product_category_product WHERE product_category_id=$1 AND product_id=ANY($2::text[])`, id, remove); err != nil {
			return err
		}
	}
	for _, pid := range add {
		if _, err = tx.Exec(ctx, `INSERT INTO product_category_product(product_id,product_category_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, pid, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

var _ domain.Repository = (*Repository)(nil)
