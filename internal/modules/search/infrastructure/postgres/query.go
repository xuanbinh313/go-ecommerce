package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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

type facetRequest struct {
	field  string
	typeID string
	limit  int
	sort   string
	query  string
	ranges []facetRange
}

type facetRange struct {
	key      string
	from, to *float64
}

func (p *Provider) resolveFacets(ctx context.Context, table string, plan indexPlan, raw any, scope string, scopeArgs []any) (map[string]models.FacetResult, error) {
	requests, err := normalizeFacets(raw, plan)
	if err != nil || len(requests) == 0 {
		return nil, err
	}
	result := make(map[string]models.FacetResult, len(requests))
	for _, request := range requests {
		facet, err := p.runFacet(ctx, table, plan, request, scope, scopeArgs)
		if err != nil {
			return nil, err
		}
		result[request.field] = facet
	}
	return result, nil
}

func normalizeFacets(raw any, plan indexPlan) ([]facetRequest, error) {
	items, ok := facetList(raw)
	if !ok {
		if raw == nil {
			return nil, nil
		}
		return nil, fmt.Errorf("search facets must be an array")
	}
	out := make([]facetRequest, 0, len(items))
	for _, item := range items {
		request := facetRequest{typeID: "value", limit: 10}
		if field, ok := item.(string); ok {
			request.field = field
		} else if object, ok := item.(map[string]any); ok {
			request.field, _ = object["field"].(string)
			if value, ok := object["type"].(string); ok && value != "" {
				request.typeID = value
			}
			if value, ok := object["sort"].(string); ok {
				request.sort = value
			}
			if value, ok := object["query"].(string); ok {
				request.query = value
			}
			if value, ok := floatValue(object["limit"]); ok && value >= 0 {
				request.limit = int(value)
			}
			if ranges, ok := facetList(object["ranges"]); ok {
				for _, rawRange := range ranges {
					value, ok := rawRange.(map[string]any)
					if !ok {
						return nil, fmt.Errorf("facet ranges must be objects")
					}
					r := facetRange{}
					if key, ok := value["key"].(string); ok {
						r.key = key
					}
					if n, ok := floatValue(value["from"]); ok {
						r.from = &n
					}
					if n, ok := floatValue(value["to"]); ok {
						r.to = &n
					}
					if r.key == "" {
						left, right := "*", "*"
						if r.from != nil {
							left = fmt.Sprint(*r.from)
						}
						if r.to != nil {
							right = fmt.Sprint(*r.to)
						}
						r.key = left + "-" + right
					}
					request.ranges = append(request.ranges, r)
				}
			}
		} else {
			return nil, fmt.Errorf("search facet must be a field name or request object")
		}
		field, exists := plan.Fields[request.field]
		if !exists || !facetable(field.Field.Facetable) {
			return nil, fmt.Errorf("field %q is not facetable on the postgres search provider", request.field)
		}
		if request.typeID != "value" && request.typeID != "range" && request.typeID != "stats" {
			return nil, fmt.Errorf("unsupported facet type %q", request.typeID)
		}
		if (request.typeID == "range" || request.typeID == "stats") && field.Kind != "number" {
			return nil, fmt.Errorf("%s facets require a numeric field (%q)", request.typeID, request.field)
		}
		if request.typeID == "range" && len(request.ranges) == 0 {
			return nil, fmt.Errorf("range facets require ranges for %q", request.field)
		}
		out = append(out, request)
	}
	return out, nil
}

func (p *Provider) runFacet(ctx context.Context, table string, plan indexPlan, request facetRequest, scope string, scopeArgs []any) (models.FacetResult, error) {
	field := plan.Fields[request.field]
	expr := jsonPath(request.field, true)
	where := ""
	if scope != "" {
		where = " WHERE (" + scope + ")"
	}
	if request.typeID == "stats" {
		var min, max, avg, sum *float64
		var count int64
		err := p.pool.QueryRow(ctx, `SELECT MIN((`+expr+`)::float8),MAX((`+expr+`)::float8),AVG((`+expr+`)::float8),SUM((`+expr+`)::float8),COUNT(*) FROM `+ident(table)+where, scopeArgs...).Scan(&min, &max, &avg, &sum, &count)
		if err != nil {
			return nil, err
		}
		result := models.FacetResult{"type": "stats", "min": numberOrZero(min), "max": numberOrZero(max), "count": count}
		if avg != nil {
			result["avg"] = *avg
		}
		if sum != nil {
			result["sum"] = *sum
		}
		return result, nil
	}
	if request.typeID == "range" {
		selects, args := []string{}, []any{}
		for i, r := range request.ranges {
			conditions := []string{}
			if r.from != nil {
				args = append(args, *r.from)
				conditions = append(conditions, `(`+expr+`)::float8 >= $`+strconv.Itoa(len(args)))
			}
			if r.to != nil {
				args = append(args, *r.to)
				conditions = append(conditions, `(`+expr+`)::float8 < $`+strconv.Itoa(len(args)))
			}
			condition := "TRUE"
			if len(conditions) > 0 {
				condition = strings.Join(conditions, " AND ")
			}
			selects = append(selects, fmt.Sprintf(`COUNT(*) FILTER (WHERE %s)::int AS r%d`, condition, i))
		}
		args = append(args, scopeArgs...)
		query := `SELECT ` + strings.Join(selects, ",") + ` FROM ` + ident(table)
		if scope != "" {
			query += ` WHERE (` + shiftPlaceholders(scope, len(args)-len(scopeArgs)) + `)`
		}
		rows, err := p.pool.Query(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		if !rows.Next() {
			return models.FacetResult{}, rows.Err()
		}
		values := make([]any, len(request.ranges))
		dest := make([]any, len(values))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		rangeResults := make([]map[string]any, len(request.ranges))
		for i, r := range request.ranges {
			var count int64
			switch n := values[i].(type) {
			case int64:
				count = n
			case int32:
				count = int64(n)
			case int:
				count = int64(n)
			case []byte:
				count, _ = strconv.ParseInt(string(n), 10, 64)
			}
			entry := map[string]any{"key": r.key, "count": count}
			if r.from != nil {
				entry["from"] = *r.from
			}
			if r.to != nil {
				entry["to"] = *r.to
			}
			rangeResults[i] = entry
		}
		return models.FacetResult{"type": "range", "ranges": rangeResults}, nil
	}
	valueExpr := expr
	if field.Array {
		valueExpr = `jsonb_array_elements_text(` + jsonPath(request.field, false) + `)`
	}
	args := append([]any(nil), scopeArgs...)
	valueFilter := ""
	if request.query != "" {
		args = append(args, request.query+"%")
		valueFilter = ` AND value LIKE $` + strconv.Itoa(len(args))
	}
	args = append(args, request.limit)
	limit := `$` + strconv.Itoa(len(args))
	order := `count DESC,value ASC`
	if request.sort == "alpha" {
		order = `value ASC`
	}
	var query string
	if field.Array {
		query = `SELECT value,COUNT(*)::int AS count FROM (SELECT ` + valueExpr + ` AS value FROM ` + ident(table) + where + `) exploded WHERE value IS NOT NULL` + valueFilter + ` GROUP BY value ORDER BY ` + order + ` LIMIT ` + limit
	} else {
		guard := valueExpr + ` IS NOT NULL`
		if scope != "" {
			where = ` WHERE (` + scope + `) AND ` + guard
		} else {
			where = ` WHERE ` + guard
		}
		query = `SELECT ` + valueExpr + ` AS value,COUNT(*)::int AS count FROM ` + ident(table) + where + valueFilter + ` GROUP BY 1 ORDER BY ` + order + ` LIMIT ` + limit
	}
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []map[string]any{}
	for rows.Next() {
		var value any
		var count int64
		if err := rows.Scan(&value, &count); err != nil {
			return nil, err
		}
		values = append(values, map[string]any{"value": fmt.Sprint(value), "count": count})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return models.FacetResult{"type": "value", "values": values}, nil
}

func facetList(value any) ([]any, bool) {
	switch list := value.(type) {
	case []any:
		return list, true
	case []string:
		out := make([]any, len(list))
		for i, v := range list {
			out[i] = v
		}
		return out, true
	}
	return nil, false
}
func facetable(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case map[string]any:
		return true
	}
	return false
}
func numberOrZero(value *float64) float64 {
	if value == nil || math.IsNaN(*value) {
		return 0
	}
	return *value
}

var _ = fmt.Sprintf
