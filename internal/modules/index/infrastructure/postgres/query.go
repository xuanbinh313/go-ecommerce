package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/index/domain/models"
)

// Query implements the Index storage query contract for indexed entity rows.
// Relation joins require Medusa's runtime entity schema and are rejected when
// requested rather than silently returning incomplete results.
func (r *Repository) Query(ctx context.Context, config models.QueryConfig) (models.QueryResult, error) {
	if config.Entity == "" {
		return models.QueryResult{}, fmt.Errorf("index query entity is required")
	}
	if len(config.JoinFilters) > 0 {
		return models.QueryResult{}, fmt.Errorf("index join filters require entity relation schema")
	}
	where := []string{"name = $1", "deleted_at IS NULL", "staled_at IS NULL"}
	args := []any{config.Entity}
	if err := appendJSONFilters(&where, &args, config.Filters, ""); err != nil {
		return models.QueryResult{}, err
	}
	whereSQL := strings.Join(where, " AND ")
	countNeeded := config.Pagination.Skip != nil
	var count *int64
	if countNeeded {
		var value int64
		if err := r.db.QueryRow(ctx, "SELECT count(*) FROM index_data WHERE "+whereSQL, args...).Scan(&value); err != nil {
			return models.QueryResult{}, err
		}
		count = &value
	}
	query := "SELECT id, data FROM index_data WHERE " + whereSQL
	if len(config.Pagination.Order) > 0 {
		orders := make([]string, 0, len(config.Pagination.Order))
		for field, rawDirection := range config.Pagination.Order {
			direction := strings.ToUpper(fmt.Sprint(rawDirection))
			if direction != "ASC" && direction != "DESC" {
				return models.QueryResult{}, fmt.Errorf("unsupported index order %q", rawDirection)
			}
			if field == "id" {
				orders = append(orders, "id "+direction)
			} else {
				orders = append(orders, jsonPathExpr(field)+" "+direction)
			}
		}
		query += " ORDER BY " + strings.Join(orders, ", ")
	} else {
		query += " ORDER BY id ASC"
	}
	if config.Pagination.Take != nil {
		args = append(args, *config.Pagination.Take)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if config.Pagination.Skip != nil {
		args = append(args, *config.Pagination.Skip)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return models.QueryResult{}, err
	}
	defer rows.Close()
	data := make([]map[string]any, 0)
	for rows.Next() {
		var id string
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return models.QueryResult{}, err
		}
		item := map[string]any{}
		if err := json.Unmarshal(raw, &item); err != nil {
			return models.QueryResult{}, err
		}
		item["id"] = id
		if config.IDsOnly {
			item = map[string]any{"id": id}
		} else if len(config.Fields) > 0 {
			projected := map[string]any{}
			for _, field := range config.Fields {
				if value, ok := getNested(item, strings.Split(field, ".")); ok {
					setNested(projected, strings.Split(field, "."), value)
				}
			}
			if _, ok := projected["id"]; !ok {
				projected["id"] = id
			}
			item = projected
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return models.QueryResult{}, err
	}
	metadata := models.QueryMetadata{Count: count, Skip: config.Pagination.Skip, Take: config.Pagination.Take}
	return models.QueryResult{Data: data, Metadata: metadata}, nil
}

func appendJSONFilters(where *[]string, args *[]any, filters map[string]any, prefix string) error {
	for key, value := range filters {
		if key == "$and" || key == "$or" {
			items, ok := value.([]any)
			if !ok {
				return fmt.Errorf("index %s filter must be an array", key)
			}
			parts := make([]string, 0, len(items))
			for _, item := range items {
				child, ok := item.(map[string]any)
				if !ok {
					return fmt.Errorf("index %s filter entries must be objects", key)
				}
				conditions := []string{}
				if err := appendJSONFilters(&conditions, args, child, prefix); err != nil {
					return err
				}
				parts = append(parts, "("+strings.Join(conditions, " AND ")+")")
			}
			joiner := " AND "
			if key == "$or" {
				joiner = " OR "
			}
			if len(parts) > 0 {
				*where = append(*where, "("+strings.Join(parts, joiner)+")")
			}
			continue
		}
		if key == "$not" {
			child, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf("index $not filter must be an object")
			}
			conditions := []string{}
			if err := appendJSONFilters(&conditions, args, child, prefix); err != nil {
				return err
			}
			*where = append(*where, "NOT ("+strings.Join(conditions, " AND ")+")")
			continue
		}
		field := key
		if prefix != "" {
			field = prefix + "." + key
		}
		if nested, ok := value.(map[string]any); ok {
			if isOperatorMap(nested) {
				for operator, operand := range nested {
					if err := appendJSONOperator(where, args, field, operator, operand); err != nil {
						return err
					}
				}
			} else if err := appendJSONFilters(where, args, nested, field); err != nil {
				return err
			}
			continue
		}
		if list, ok := value.([]any); ok {
			if len(list) == 0 {
				*where = append(*where, "FALSE")
				continue
			}
			*where = append(*where, inExpression(field, list, false, args))
			continue
		}
		if err := appendJSONOperator(where, args, field, "$eq", value); err != nil {
			return err
		}
	}
	return nil
}

func appendJSONOperator(where *[]string, args *[]any, field, operator string, value any) error {
	ops := map[string]string{"$eq": "=", "$ne": "<>", "$lt": "<", "$lte": "<=", "$gt": ">", "$gte": ">=", "$is": "IS NOT DISTINCT FROM", "$like": "LIKE", "$ilike": "ILIKE"}
	if operator == "$in" || operator == "$nin" {
		list, ok := value.([]any)
		if !ok {
			return fmt.Errorf("index %s expects an array", operator)
		}
		if len(list) == 0 {
			if operator == "$in" {
				*where = append(*where, "FALSE")
			}
			return nil
		}
		*where = append(*where, inExpression(field, list, operator == "$nin", args))
		return nil
	}
	sqlOperator, ok := ops[operator]
	if !ok {
		return fmt.Errorf("unsupported index filter operator %q", operator)
	}
	if field == "id" {
		*where = append(*where, "id "+sqlOperator+" "+bind(args, value))
		return nil
	}
	expression := jsonPathExpr(field)
	switch value.(type) {
	case float64, int, int32, int64:
		expression = "(" + expression + ")::numeric"
	case bool:
		expression = "(" + expression + ")::boolean"
	}
	*where = append(*where, expression+" "+sqlOperator+" "+bind(args, value))
	return nil
}

func jsonPathExpr(field string) string {
	parts := strings.Split(field, ".")
	expr := "data"
	for _, part := range parts {
		expr += "->" + quoteLiteral(part)
	}
	return "(" + expr + ") #>> '{}'"
}

func bind(args *[]any, value any) string {
	*args = append(*args, value)
	return "$" + strconv.Itoa(len(*args))
}

func isOperatorMap(value map[string]any) bool {
	for key := range value {
		if strings.HasPrefix(key, "$") {
			return true
		}
	}
	return false
}

func inExpression(field string, values []any, negate bool, args *[]any) string {
	column := "id"
	if field != "id" {
		column = jsonPathExpr(field)
	}
	placeholders := make([]string, 0, len(values))
	for _, value := range values {
		placeholders = append(placeholders, bind(args, value))
	}
	operator := " IN "
	if negate {
		operator = " NOT IN "
	}
	return column + operator + "(" + strings.Join(placeholders, ",") + ")"
}

func getNested(value map[string]any, path []string) (any, bool) {
	if len(path) == 0 {
		return value, true
	}
	child, ok := value[path[0]]
	if !ok {
		return nil, false
	}
	if len(path) == 1 {
		return child, true
	}
	nested, ok := child.(map[string]any)
	if !ok {
		return nil, false
	}
	return getNested(nested, path[1:])
}

func setNested(target map[string]any, path []string, value any) {
	if len(path) == 1 {
		target[path[0]] = value
		return
	}
	child, ok := target[path[0]].(map[string]any)
	if !ok {
		child = map[string]any{}
		target[path[0]] = child
	}
	setNested(child, path[1:], value)
}

// Compile-time interface checks keep the storage adapter aligned with the
// module's provider boundary.
var _ interface {
	Query(context.Context, models.QueryConfig) (models.QueryResult, error)
} = (*Repository)(nil)
