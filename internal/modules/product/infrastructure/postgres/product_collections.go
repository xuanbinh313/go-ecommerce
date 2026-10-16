package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/product/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
)

const productCollectionColumns = `id,title,handle,external_id,metadata,created_at,updated_at,deleted_at`

func scanProductCollection(row pgx.Row) (models.ProductCollection, error) {
	var v models.ProductCollection
	err := row.Scan(&v.ID, &v.Title, &v.Handle, &v.ExternalID, &v.Metadata, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	return v, err
}
func (r *Repository) ListProductCollections(ctx context.Context, f models.ProductCollectionFilter, limit, offset int) ([]models.ProductCollection, int, error) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	for _, x := range []struct {
		col    string
		values []string
	}{{"id", f.IDs}, {"title", f.Titles}, {"handle", f.Handles}, {"external_id", f.ExternalIDs}} {
		if len(x.values) > 0 {
			args = append(args, x.values)
			where = append(where, fmt.Sprintf("%s=ANY($%d::text[])", x.col, len(args)))
		}
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(title ILIKE $%d OR handle ILIKE $%d OR COALESCE(external_id,'') ILIKE $%d)", n, n, n))
	}
	for _, dateFilter := range []struct {
		column string
		values map[string]string
	}{{"created_at", f.CreatedAt}, {"updated_at", f.UpdatedAt}} {
		for operator, value := range dateFilter.values {
			sqlOperator, ok := map[string]string{"$eq": "=", "$ne": "<>", "$gt": ">", "$gte": ">=", "$lt": "<", "$lte": "<="}[operator]
			if !ok {
				return nil, 0, fmt.Errorf("unsupported %s filter operator %q", dateFilter.column, operator)
			}
			args = append(args, value)
			where = append(where, fmt.Sprintf("%s %s $%d::timestamptz", dateFilter.column, sqlOperator, len(args)))
		}
	}
	w := strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM product_collection WHERE `+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	orderColumn, orderDirection := "created_at", "DESC"
	if f.Order != "" {
		order := strings.TrimSpace(f.Order)
		if strings.HasPrefix(order, "-") {
			orderColumn, orderDirection = strings.TrimPrefix(order, "-"), "DESC"
		} else {
			orderColumn, orderDirection = order, "ASC"
		}
		if orderColumn != "created_at" && orderColumn != "updated_at" && orderColumn != "title" && orderColumn != "handle" && orderColumn != "external_id" && orderColumn != "id" {
			return nil, 0, fmt.Errorf("unsupported product collection order field %q", orderColumn)
		}
	}
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT %s FROM product_collection WHERE %s ORDER BY %s %s,id LIMIT $%d OFFSET $%d`, productCollectionColumns, w, orderColumn, orderDirection, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.ProductCollection{}
	for rows.Next() {
		v, e := scanProductCollection(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) GetProductCollection(ctx context.Context, id string) (models.ProductCollection, error) {
	return scanProductCollection(r.db.QueryRow(ctx, `SELECT `+productCollectionColumns+` FROM product_collection WHERE id=$1 AND deleted_at IS NULL`, id))
}
func (r *Repository) CreateProductCollections(ctx context.Context, in []models.ProductCollectionInput) ([]models.ProductCollection, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := make([]models.ProductCollection, 0, len(in))
	for _, v := range in {
		handle := v.Title
		if v.Handle != nil {
			handle = *v.Handle
		}
		created, e := scanProductCollection(tx.QueryRow(ctx, `INSERT INTO product_collection(id,title,handle,external_id,metadata) VALUES($1,$2,$3,$4,$5) RETURNING `+productCollectionColumns, entityid.New("pcol"), v.Title, handle, v.ExternalID, raw(v.Metadata)))
		if e != nil {
			return nil, e
		}
		out = append(out, created)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
func (r *Repository) UpdateProductCollection(ctx context.Context, id string, fields map[string]any) (models.ProductCollection, error) {
	columns := map[string]string{"title": "title", "handle": "handle", "external_id": "external_id", "metadata": "metadata"}
	sets := []string{}
	args := []any{id}
	for k, v := range fields {
		if col, ok := columns[k]; ok {
			args = append(args, v)
			sets = append(sets, fmt.Sprintf("%s=$%d", col, len(args)))
		}
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		tag, err := r.db.Exec(ctx, `UPDATE product_collection SET `+strings.Join(sets, ",")+` WHERE id=$1 AND deleted_at IS NULL`, args...)
		if err != nil {
			return models.ProductCollection{}, err
		}
		if tag.RowsAffected() == 0 {
			return models.ProductCollection{}, pgx.ErrNoRows
		}
	}
	return r.GetProductCollection(ctx, id)
}
func (r *Repository) SoftDeleteProductCollections(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE product_collection SET deleted_at=now(),updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NULL`, ids)
	return err
}
func (r *Repository) RestoreProductCollections(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE product_collection SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NOT NULL`, ids)
	return err
}
func (r *Repository) HardDeleteProductCollections(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM product_collection WHERE id=ANY($1::text[])`, ids)
	return err
}
func (r *Repository) ListProductIDsForCollection(ctx context.Context, id string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM product WHERE collection_id=$1 AND deleted_at IS NULL ORDER BY created_at,id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var v string
		if err = rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *Repository) SetProductsOnCollection(ctx context.Context, id string, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE product SET collection_id=NULL,updated_at=now() WHERE collection_id=$1 AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	if len(ids) > 0 {
		if _, err = tx.Exec(ctx, `UPDATE product SET collection_id=$1,updated_at=now() WHERE id=ANY($2::text[]) AND deleted_at IS NULL`, id, ids); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (r *Repository) LinkProductsToCollection(ctx context.Context, id string, add, remove []string) error {
	existing, err := r.ListProductIDsForCollection(ctx, id)
	if err != nil {
		return err
	}
	removed := map[string]bool{}
	for _, v := range remove {
		removed[v] = true
	}
	next := []string{}
	seen := map[string]bool{}
	for _, v := range existing {
		if !removed[v] && !seen[v] {
			next = append(next, v)
			seen[v] = true
		}
	}
	for _, v := range add {
		if !seen[v] {
			next = append(next, v)
			seen[v] = true
		}
	}
	return r.SetProductsOnCollection(ctx, id, next)
}
