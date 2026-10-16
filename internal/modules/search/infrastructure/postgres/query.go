package postgres

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/search/domain/models"
)

func compileFilters(filters models.SearchFilters, plan indexPlan) (string, []any, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}
	args := []any{}
	var compile func(map[string]any) (string, error)
	compile = func(branch map[string]any) (string, error) {
		parts := []string{}
		for key, value := range branch {
			switch key {
			case "q":
				continue
			case "$and", "$or":
				list, ok := value.([]any)
				if !ok {
					if typed, yes := value.([]map[string]any); yes {
						list = make([]any, len(typed))
						for i, v := range typed {
							list[i] = v
						}
					} else {
						return "", fmt.Errorf("%s filter must be an array", key)
					}
				}
				nested := []string{}
				for _, raw := range list {
					item, ok := raw.(map[string]any)
					if !ok {
						return "", fmt.Errorf("%s filter entries must be objects", key)
					}
					sql, err := compile(item)
					if err != nil {
						return "", err
					}
					if sql != "" {
						nested = append(nested, "("+sql+")")
					}
				}
				if len(nested) > 0 {
					join := " AND "
					if key == "$or" {
						join = " OR "
					}
					parts = append(parts, "("+strings.Join(nested, join)+")")
				}
			case "$not":
				item, ok := value.(map[string]any)
				if !ok {
					return "", fmt.Errorf("$not filter must be an object")
				}
				sql, err := compile(item)
				if err != nil {
					return "", err
				}
				if sql != "" {
					parts = append(parts, "NOT ("+sql+")")
				}
			default:
				field, ok := plan.Fields[key]
				if !ok {
					return "", fmt.Errorf("unknown filter field %q on the postgres search provider", key)
				}
				sql, err := fieldFilter(field, value, &args)
				if err != nil {
					return "", err
				}
				if sql != "" {
					parts = append(parts, sql)
				}
			}
		}
		return strings.Join(parts, " AND "), nil
	}
	sql, err := compile(map[string]any(filters))
	return sql, args, err
}

func fieldFilter(field plannedField, raw any, args *[]any) (string, error) {
	if raw == nil || !isObject(raw) || isArray(raw) {
		values := []any{raw}
		if list, ok := raw.([]any); ok {
			values = list
		}
		parts := []string{}
		for _, value := range values {
			part, err := equalFilter(field, value, args)
			if err != nil {
				return "", err
			}
			parts = append(parts, part)
		}
		if len(parts) == 0 {
			return "FALSE", nil
		}
		return "(" + strings.Join(parts, " OR ") + ")", nil
	}
	operators, ok := raw.(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid filter for %q", field.Path)
	}
	parts := []string{}
	for op, value := range operators {
		if !strings.HasPrefix(op, "$") {
			continue
		}
		switch op {
		case "$eq":
			part, err := equalFilter(field, value, args)
			if err != nil {
				return "", err
			}
			parts = append(parts, part)
		case "$ne":
			if value == nil {
				parts = append(parts, existsFilter(field, true))
			} else {
				part, err := equalFilter(field, value, args)
				if err != nil {
					return "", err
				}
				parts = append(parts, "NOT ("+part+")")
			}
		case "$in", "$nin":
			values, ok := value.([]any)
			if !ok {
				values = []any{value}
			}
			matches := []string{}
			for _, v := range values {
				part, err := equalFilter(field, v, args)
				if err != nil {
					return "", err
				}
				matches = append(matches, part)
			}
			sql := "FALSE"
			if len(matches) > 0 {
				sql = "(" + strings.Join(matches, " OR ") + ")"
			}
			if op == "$nin" {
				sql = "NOT " + sql
			}
			parts = append(parts, sql)
		case "$exists":
			exists := value == true || value == "true"
			parts = append(parts, existsFilter(field, exists))
		case "$gt", "$gte", "$lt", "$lte":
			if field.Array {
				return "", fmt.Errorf("operator %q is not supported on array field %q", op, field.Path)
			}
			cast, err := castFilter(field, value)
			if err != nil {
				return "", err
			}
			cmp := map[string]string{"$gt": ">", "$gte": ">=", "$lt": "<", "$lte": "<="}[op]
			*args = append(*args, cast)
			parts = append(parts, castExpr(field)+" "+cmp+" $"+strconv.Itoa(len(*args)))
		case "$prefix", "$like":
			if field.Array {
				return "", fmt.Errorf("operator %q is not supported on array field %q", op, field.Path)
			}
			pattern := fmt.Sprint(value)
			if op == "$prefix" {
				pattern += "%"
			}
			*args = append(*args, pattern)
			parts = append(parts, jsonPath(field.Path, true)+" LIKE $"+strconv.Itoa(len(*args)))
		case "$contains", "$overlaps":
			if !field.Array {
				return "", fmt.Errorf("operator %q needs an array field %q", op, field.Path)
			}
			values, ok := value.([]any)
			if !ok {
				values = []any{value}
			}
			if op == "$contains" {
				cast := make([]any, 0, len(values))
				for _, v := range values {
					x, e := castFilter(field, v)
					if e != nil {
						return "", e
					}
					cast = append(cast, x)
				}
				doc := nestPath(field.Path, cast)
				*args = append(*args, jsonValue(doc))
				parts = append(parts, "indexed @> $"+strconv.Itoa(len(*args))+"::jsonb")
			} else {
				matches := []string{}
				for _, v := range values {
					part, e := equalFilter(field, v, args)
					if e != nil {
						return "", e
					}
					matches = append(matches, part)
				}
				parts = append(parts, "("+strings.Join(matches, " OR ")+")")
			}
		default:
			return "", fmt.Errorf("the postgres search provider does not support operator %q on %q", op, field.Path)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("empty operator map on filter field %q", field.Path)
	}
	return "(" + strings.Join(parts, " AND ") + ")", nil
}

func equalFilter(field plannedField, value any, args *[]any) (string, error) {
	if value == nil {
		return existsFilter(field, false), nil
	}
	cast, err := castFilter(field, value)
	if err != nil {
		return "", err
	}
	if field.Array {
		cast = []any{cast}
	}
	*args = append(*args, jsonValue(nestPath(field.Path, cast)))
	return "indexed @> $" + strconv.Itoa(len(*args)) + "::jsonb", nil
}
func existsFilter(field plannedField, exists bool) string {
	expr := jsonPath(field.Path, false)
	if exists {
		return "(" + expr + " IS NOT NULL AND " + expr + " <> 'null'::jsonb)"
	}
	return "(" + expr + " IS NULL OR " + expr + " = 'null'::jsonb)"
}
func castExpr(field plannedField) string {
	expr := jsonPath(field.Path, true)
	if field.Kind == "number" {
		return "(" + expr + ")::float8"
	}
	if field.Kind == "boolean" {
		return "(" + expr + ")::boolean"
	}
	return expr
}
func castFilter(field plannedField, value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch field.Kind {
	case "number":
		if field.Date {
			if tm, ok := value.(time.Time); ok {
				return float64(tm.UnixMilli()), nil
			}
			if s, ok := value.(string); ok {
				if n, e := strconv.ParseFloat(s, 64); e == nil {
					return n, nil
				}
				tm, e := time.Parse(time.RFC3339Nano, s)
				if e == nil {
					return float64(tm.UnixMilli()), nil
				}
			}
		}
		if n, ok := floatValue(value); ok {
			return n, nil
		}
		return nil, fmt.Errorf("cannot cast filter value for %q to a number", field.Path)
	case "boolean":
		if b, ok := value.(bool); ok {
			return b, nil
		}
		return value == "true", nil
	default:
		return fmt.Sprint(value), nil
	}
}
func jsonPath(path string, asText bool) string {
	segments := strings.Split(path, ".")
	expr := "indexed"
	for i, s := range segments {
		s = strings.ReplaceAll(s, "'", "''")
		if i == len(segments)-1 && asText {
			expr += "->>'" + s + "'"
		} else {
			expr += "->'" + s + "'"
		}
	}
	return expr
}
func nestPath(path string, value any) any {
	segments := strings.Split(path, ".")
	for i := len(segments) - 1; i >= 0; i-- {
		value = map[string]any{segments[i]: value}
	}
	return value
}
func jsonValue(value any) string { b, _ := json.Marshal(value); return string(b) }
func isObject(v any) bool {
	switch v.(type) {
	case map[string]any:
		return true
	}
	return false
}
func isArray(v any) bool {
	switch v.(type) {
	case []any, []string:
		return true
	}
	return false
}

func projectForIndex(doc models.Document, plan indexPlan) (projectedDocument, error) {
	primary := doc[plan.PrimaryKey]
	if primary == nil || fmt.Sprint(primary) == "" {
		return projectedDocument{}, fmt.Errorf("a document written to the postgres search provider is missing its primary key %q", plan.PrimaryKey)
	}
	out := projectedDocument{id: fmt.Sprint(primary), document: doc, indexed: map[string]any{}}
	textParts := []string{}
	for _, field := range plan.Fields {
		values := readPath(doc, strings.Split(field.Path, "."))
		coerced := []any{}
		for _, v := range values {
			if v == nil {
				continue
			}
			value, err := coerceField(v, field)
			if err != nil {
				return projectedDocument{}, err
			}
			coerced = append(coerced, value)
		}
		if len(coerced) == 0 {
			continue
		}
		value := coerced[0]
		if field.Array {
			value = coerced
		}
		putPath(out.indexed, strings.Split(field.Path, "."), value)
		if searchable(field.Field.Searchable) && (field.Kind == "text" || field.Kind == "keyword" || (field.Kind == "number" && !field.Date)) {
			vals := []any{value}
			if list, ok := value.([]any); ok {
				vals = list
			}
			words := []string{}
			for _, v := range vals {
				words = append(words, fmt.Sprint(v))
			}
			part := strings.Join(words, " ")
			textParts = append(textParts, part)
			out.weighted = append(out.weighted, weightedText{part, weightLabel(field.Field.Searchable)})
		}
	}
	out.searchText = strings.Join(textParts, " ")
	return out, nil
}
func coerceField(v any, field plannedField) (any, error) {
	switch field.Kind {
	case "number":
		if field.Date {
			if tm, ok := v.(time.Time); ok {
				return float64(tm.UnixMilli()), nil
			}
			if text, ok := v.(string); ok {
				if n, e := strconv.ParseFloat(text, 64); e == nil {
					return n, nil
				}
				tm, e := time.Parse(time.RFC3339Nano, text)
				if e == nil {
					return float64(tm.UnixMilli()), nil
				}
			}
		}
		if n, ok := floatValue(v); ok {
			return n, nil
		}
		return nil, fmt.Errorf("cannot cast field %q to a number", field.Path)
	case "boolean":
		if b, ok := v.(bool); ok {
			return b, nil
		}
		if text, ok := v.(string); ok {
			return text == "true", nil
		}
		return v != nil, nil
	case "text", "keyword":
		return fmt.Sprint(v), nil
	}
	return v, nil
}
func readPath(value any, path []string) []any {
	if len(path) == 0 {
		if list, ok := value.([]any); ok {
			return list
		}
		return []any{value}
	}
	if list, ok := value.([]any); ok {
		out := []any{}
		for _, item := range list {
			out = append(out, readPath(item, path)...)
		}
		return out
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return readPath(object[path[0]], path[1:])
}
func putPath(object map[string]any, path []string, value any) {
	if len(path) == 1 {
		object[path[0]] = value
		return
	}
	child, ok := object[path[0]].(map[string]any)
	if !ok {
		child = map[string]any{}
		object[path[0]] = child
	}
	putPath(child, path[1:], value)
}
func projectDocument(doc models.Document, fields []string) models.Document {
	if fields == nil {
		return doc
	}
	out := models.Document{}
	for _, name := range fields {
		path := strings.Split(name, ".")
		if value, ok := getPath(doc, path); ok {
			putPath(out, path, value)
		}
	}
	return out
}
func getPath(value map[string]any, path []string) (any, bool) {
	if len(path) == 0 {
		return value, true
	}
	v, ok := value[path[0]]
	if !ok {
		return nil, false
	}
	if len(path) == 1 {
		return v, true
	}
	child, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	return getPath(child, path[1:])
}

func searchExpressions(paths []string, plan indexPlan, tsConfig string) (string, string) {
	vectors := []string{}
	texts := []string{}
	for _, path := range paths {
		jsonPath := "'{" + strings.ReplaceAll(path, ".", ",") + "}'"
		text := "coalesce(indexed#>>" + jsonPath + ",'')"
		texts = append(texts, text)
		field := plan.Fields[path]
		vectors = append(vectors, "setweight(to_tsvector('"+tsConfig+"',"+text+"),'"+weightLabel(field.Field.Searchable)+"')")
	}
	if len(paths) == 0 {
		return `''::tsvector`, `''`
	}
	return strings.Join(vectors, " || "), strings.Join(texts, " || ' ' || ")
}
func tsQuery(config, strategy string, n int) string {
	base := fmt.Sprintf("plainto_tsquery('%s',$%d)", config, n)
	switch strategy {
	case "any":
		return "replace(" + base + "::text,' & ',' | ')::tsquery"
	case "last":
		return "regexp_replace(" + base + "::text, '''([^'']+)''$', '''\\1'':*')::tsquery"
	default:
		return base
	}
}
func typoRequested(options map[string]any) bool {
	switch v := options["typo_tolerance"].(type) {
	case bool:
		return v
	case map[string]any:
		b, _ := v["enabled"].(bool)
		return b
	default:
		return false
	}
}
func typoEnabled(settings map[string]any) bool {
	raw, ok := settings["typo_tolerance"].(map[string]any)
	if !ok {
		return true
	}
	b, ok := raw["enabled"].(bool)
	return !ok || b
}
func weightLabel(value any) string {
	m, ok := value.(map[string]any)
	if !ok {
		return "D"
	}
	n, _ := floatValue(m["weight"])
	if n >= 3 {
		return "A"
	}
	if n >= 2 {
		return "B"
	}
	if n > 1 {
		return "C"
	}
	return "D"
}
func shiftPlaceholders(sql string, offset int) string {
	re := regexp.MustCompile(`\$(\d+)`)
	return re.ReplaceAllStringFunc(sql, func(s string) string {
		n, _ := strconv.Atoi(strings.TrimPrefix(s, "$"))
		return "$" + strconv.Itoa(n+offset)
	})
}
func searchOrder(order map[string]string, plan indexPlan, hasScore bool) (string, error) {
	if len(order) == 0 {
		if hasScore {
			return `score DESC NULLS LAST,id ASC`, nil
		}
		return `id ASC`, nil
	}
	parts := []string{}
	for field, direction := range order {
		dir := "DESC"
		if direction == "ASC" {
			dir = "ASC"
		}
		if field == "_score" {
			parts = append(parts, "score "+dir+" NULLS LAST")
			continue
		}
		planned, ok := plan.Fields[field]
		if !ok {
			return "", fmt.Errorf("cannot sort search index by unknown field %q", field)
		}
		if planned.Array {
			return "", fmt.Errorf("the postgres search provider cannot sort by %q", field)
		}
		expr := jsonPath(field, true)
		if planned.Kind == "number" {
			expr = "(" + expr + ")::float8"
		} else if planned.Kind == "boolean" {
			expr = "(" + expr + ")::boolean"
		}
		parts = append(parts, expr+" "+dir+" NULLS LAST")
	}
	return strings.Join(parts, ","), nil
}

var _ = fmt.Sprintf
