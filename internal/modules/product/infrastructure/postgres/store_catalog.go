package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
)

func storeCatalogWhere(filter models.ProductCatalogFilter) (string, []any, error) {
	where := []string{}
	if len(filter.DeletedAt) == 0 {
		where = append(where, "deleted_at IS NULL")
	}
	args := []any{}
	for _, condition := range []struct {
		column string
		values []string
	}{{"id", filter.IDs}, {"value", filter.Values}, {"external_id", filter.ExternalIDs}} {
		if len(condition.values) > 0 {
			args = append(args, condition.values)
			where = append(where, fmt.Sprintf("%s=ANY($%d::text[])", condition.column, len(args)))
		}
	}
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(value ILIKE $%d OR COALESCE(external_id,'') ILIKE $%d)", n, n))
	}
	for _, dateFilter := range []struct {
		column string
		values map[string]string
	}{{"created_at", filter.CreatedAt}, {"updated_at", filter.UpdatedAt}, {"deleted_at", filter.DeletedAt}} {
		for operator, value := range dateFilter.values {
			sqlOperator, ok := map[string]string{"$eq": "=", "$ne": "<>", "$gt": ">", "$gte": ">=", "$lt": "<", "$lte": "<="}[operator]
			if !ok {
				return "", nil, fmt.Errorf("unsupported %s filter operator %q", dateFilter.column, operator)
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
				return "", nil, fmt.Errorf("unsupported deleted_at null comparison %q", operator)
			}
			args = append(args, value)
			where = append(where, fmt.Sprintf("%s %s $%d::timestamptz", dateFilter.column, sqlOperator, len(args)))
		}
	}
	if len(where) == 0 {
		where = append(where, "TRUE")
	}
	return strings.Join(where, " AND "), args, nil
}

func (r *Repository) ListStoreProductTypes(ctx context.Context, filter models.ProductCatalogFilter, limit, offset int) ([]models.ProductType, int, error) {
	where, args, err := storeCatalogWhere(filter)
	if err != nil {
		return nil, 0, err
	}
	var count int
	if err = r.db.QueryRow(ctx, `SELECT count(*) FROM product_type WHERE `+where, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT id,value,external_id,metadata,created_at,updated_at,deleted_at FROM product_type WHERE %s ORDER BY created_at,id LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
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

func (r *Repository) ListStoreProductTags(ctx context.Context, filter models.ProductCatalogFilter, limit, offset int) ([]models.ProductTag, int, error) {
	where, args, err := storeCatalogWhere(filter)
	if err != nil {
		return nil, 0, err
	}
	var count int
	if err = r.db.QueryRow(ctx, `SELECT count(*) FROM product_tag WHERE `+where, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT id,value,external_id,metadata,created_at,updated_at,deleted_at FROM product_tag WHERE %s ORDER BY created_at,id LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
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

func (r *Repository) ListProductIDsForType(ctx context.Context, id string) ([]string, error) {
	return r.listPublishedProductIDs(ctx, `SELECT id FROM product WHERE type_id=$1 AND status='published' AND deleted_at IS NULL ORDER BY created_at,id`, id)
}

func (r *Repository) ListProductIDsForTag(ctx context.Context, id string) ([]string, error) {
	return r.listPublishedProductIDs(ctx, `SELECT p.id FROM product p JOIN product_tags pt ON pt.product_id=p.id WHERE pt.product_tag_id=$1 AND p.status='published' AND p.deleted_at IS NULL ORDER BY p.created_at,p.id`, id)
}

func (r *Repository) listPublishedProductIDs(ctx context.Context, query, id string) ([]string, error) {
	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
