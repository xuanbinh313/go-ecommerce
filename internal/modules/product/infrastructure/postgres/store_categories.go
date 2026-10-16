package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
)

func (r *Repository) ListStoreProductCategories(ctx context.Context, filter models.StoreProductCategoryFilter, limit, offset int) ([]models.ProductCategory, int, error) {
	where := []string{"deleted_at IS NULL", "is_active=TRUE", "is_internal=FALSE"}
	args := []any{}
	for _, condition := range []struct {
		column string
		values []string
	}{
		{"id", filter.IDs}, {"name", filter.Names}, {"description", filter.Descriptions}, {"handle", filter.Handles}, {"external_id", filter.ExternalIDs},
	} {
		if len(condition.values) > 0 {
			args = append(args, condition.values)
			where = append(where, fmt.Sprintf("%s=ANY($%d::text[])", condition.column, len(args)))
		}
	}
	if len(filter.ParentCategoryIDs) > 0 {
		var ids []string
		for _, id := range filter.ParentCategoryIDs {
			if id != "null" {
				ids = append(ids, id)
			}
		}
		conditions := []string{}
		if len(ids) > 0 {
			args = append(args, ids)
			conditions = append(conditions, fmt.Sprintf("parent_category_id=ANY($%d::text[])", len(args)))
		}
		for _, id := range filter.ParentCategoryIDs {
			if id == "null" {
				conditions = append(conditions, "parent_category_id IS NULL")
				break
			}
		}
		if len(conditions) > 0 {
			where = append(where, "("+strings.Join(conditions, " OR ")+")")
		}
	}
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d OR handle ILIKE $%d)", n, n, n))
	}
	for _, dateFilter := range []struct {
		column string
		values map[string]string
	}{{"created_at", filter.CreatedAt}, {"updated_at", filter.UpdatedAt}} {
		for operator, value := range dateFilter.values {
			sqlOperator, ok := map[string]string{"$eq": "=", "$ne": "<>", "$gt": ">", "$gte": ">=", "$lt": "<", "$lte": "<="}[operator]
			if !ok {
				return nil, 0, fmt.Errorf("unsupported %s filter operator %q", dateFilter.column, operator)
			}
			args = append(args, value)
			where = append(where, fmt.Sprintf("%s %s $%d::timestamptz", dateFilter.column, sqlOperator, len(args)))
		}
	}
	clause := strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM product_category WHERE `+clause, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT %s FROM product_category WHERE %s ORDER BY rank,created_at,id LIMIT $%d OFFSET $%d`, categoryColumns, clause, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.ProductCategory{}
	for rows.Next() {
		item, err := scanProductCategory(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, count, rows.Err()
}

func (r *Repository) ListProductIDsForCategory(ctx context.Context, id string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT p.id FROM product_category_product cp JOIN product p ON p.id=cp.product_id WHERE cp.product_category_id=$1 AND p.status='published' AND p.deleted_at IS NULL ORDER BY p.created_at,p.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var productID string
		if err := rows.Scan(&productID); err != nil {
			return nil, err
		}
		ids = append(ids, productID)
	}
	return ids, rows.Err()
}
