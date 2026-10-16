package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/links/domain"
	"github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, d models.Definition, links []map[string]any) ([]models.Link, error) {
	out := make([]models.Link, 0, len(links))
	for _, fields := range links {
		columns := []string{"id"}
		values := []any{entityid.New(d.IDPrefix)}
		for _, key := range d.ForeignKeys {
			columns = append(columns, key)
			values = append(values, fields[key])
		}
		for _, key := range d.ExtraFields {
			if value, ok := fields[key]; ok {
				columns = append(columns, key)
				values = append(values, value)
			}
		}
		marks := make([]string, len(values))
		for i := range marks {
			marks[i] = fmt.Sprintf("$%d", i+1)
		}
		query := fmt.Sprintf("INSERT INTO %s(%s) VALUES(%s) RETURNING id,created_at,updated_at,deleted_at", d.TableName, strings.Join(columns, ","), strings.Join(marks, ","))
		item := models.Link{ID: values[0].(string), Definition: d.Name, Fields: map[string]any{}}
		for i, key := range d.ForeignKeys {
			item.Fields[key] = values[i+1]
		}
		for _, key := range d.ExtraFields {
			if value, ok := fields[key]; ok {
				item.Fields[key] = value
			}
		}
		if err := r.db.QueryRow(ctx, query, values...).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *Repository) Update(ctx context.Context, d models.Definition, filters, data map[string]any) ([]models.Link, error) {
	sets, args := []string{}, []any{}
	for _, key := range d.ExtraFields {
		if value, ok := data[key]; ok {
			args = append(args, value)
			sets = append(sets, fmt.Sprintf("%s=$%d", key, len(args)))
		}
	}
	if len(sets) > 0 {
		where, whereArgs := predicate(d, filters, false)
		if where == "" {
			return nil, fmt.Errorf("link update requires filters")
		}
		for _, value := range whereArgs {
			args = append(args, value)
		}
		// predicate placeholders start at one; shift them after update values.
		shifted := where
		for i := len(whereArgs); i >= 1; i-- {
			shifted = strings.ReplaceAll(shifted, fmt.Sprintf("$%d", i), fmt.Sprintf("$%d", len(sets)+i))
		}
		sets = append(sets, "updated_at=now()")
		if _, err := r.db.Exec(ctx, fmt.Sprintf("UPDATE %s SET %s%s", d.TableName, strings.Join(sets, ","), shifted), args...); err != nil {
			return nil, err
		}
	}
	return r.List(ctx, d, filters)
}

func (r *Repository) List(ctx context.Context, d models.Definition, filters map[string]any) ([]models.Link, error) {
	where, args := predicate(d, filters, false)
	columns := append(append([]string{}, d.ForeignKeys...), d.ExtraFields...)
	query := fmt.Sprintf("SELECT id,%s,created_at,updated_at,deleted_at FROM %s%s ORDER BY created_at", strings.Join(columns, ","), d.TableName, where)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Link{}
	for rows.Next() {
		item := models.Link{Definition: d.Name, Fields: map[string]any{}}
		dest := []any{&item.ID}
		vals := make([]any, len(columns))
		for i := range vals {
			if i < len(d.ForeignKeys) {
				vals[i] = new(string)
			} else {
				vals[i] = new(float64)
			}
			dest = append(dest, vals[i])
		}
		dest = append(dest, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt)
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		for i, k := range d.ForeignKeys {
			item.Fields[k] = *(vals[i].(*string))
		}
		for i, k := range d.ExtraFields {
			item.Fields[k] = *(vals[len(d.ForeignKeys)+i].(*float64))
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) Dismiss(ctx context.Context, d models.Definition, filters []map[string]any) ([]models.Link, error) {
	var dismissed []models.Link
	for _, f := range filters {
		rows, err := r.List(ctx, d, f)
		if err != nil {
			return nil, err
		}
		dismissed = append(dismissed, rows...)
		where, args := predicate(d, f, false)
		if _, err := r.db.Exec(ctx, fmt.Sprintf("UPDATE %s SET deleted_at=now(),updated_at=now()%s", d.TableName, where), args...); err != nil {
			return nil, err
		}
	}
	return dismissed, nil
}
func (r *Repository) Delete(ctx context.Context, d models.Definition, filters []map[string]any) error {
	for _, f := range filters {
		where, args := predicate(d, f, true)
		if _, err := r.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s%s", d.TableName, where), args...); err != nil {
			return err
		}
	}
	return nil
}
func (r *Repository) Restore(ctx context.Context, d models.Definition, filters []map[string]any) ([]models.Link, error) {
	for _, f := range filters {
		where, args := predicate(d, f, true)
		if _, err := r.db.Exec(ctx, fmt.Sprintf("UPDATE %s SET deleted_at=NULL,updated_at=now()%s", d.TableName, where), args...); err != nil {
			return nil, err
		}
	}
	merged := map[string]any{}
	if len(filters) == 1 {
		merged = filters[0]
	}
	return r.List(ctx, d, merged)
}

func predicate(d models.Definition, filters map[string]any, includeDeleted bool) (string, []any) {
	clauses := []string{}
	args := []any{}
	for _, key := range append([]string{"id"}, d.ForeignKeys...) {
		if val, ok := filters[key]; ok {
			args = append(args, val)
			clauses = append(clauses, fmt.Sprintf("%s=$%d", key, len(args)))
		}
	}
	if !includeDeleted {
		clauses = append(clauses, "deleted_at IS NULL")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

var _ domain.Repository = (*Repository)(nil)
