package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/search/domain"
	"github.com/example/medusa-go-clone/internal/modules/search/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ProviderID = "search-postgres"

type Provider struct {
	pool     *pgxpool.Pool
	language string
	tsConfig string
}

type plannedField struct {
	Path  string                 `json:"path"`
	Kind  string                 `json:"kind"`
	Array bool                   `json:"is_array"`
	Date  bool                   `json:"is_date"`
	Field models.FieldDefinition `json:"field"`
}
type indexPlan struct {
	PrimaryKey string                  `json:"primary_key"`
	Searchable []string                `json:"searchable"`
	Vectors    []string                `json:"vectors"`
	SchemaHash string                  `json:"schema_hash"`
	Fields     map[string]plannedField `json:"-"`
}
type catalogEntry struct {
	Name          string
	Table         string
	SchemaHash    string
	Plan          indexPlan
	DocumentCount int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
type weightedText struct{ text, weight string }
type projectedDocument struct {
	id         string
	document   models.Document
	indexed    map[string]any
	searchText string
	weighted   []weightedText
}

// NewProvider uses the source default language and native PostgreSQL engine.
func NewProvider(pool *pgxpool.Pool, language string) (*Provider, error) {
	if pool == nil {
		return nil, errors.New("the postgres search provider requires a database pool")
	}
	if language == "" {
		language = "english"
	}
	language = strings.ToLower(language)
	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(language) {
		return nil, fmt.Errorf("invalid postgres search language %q", language)
	}
	return &Provider{pool: pool, language: language, tsConfig: "medusa_search_" + language}, nil
}
func (p *Provider) Identifier() string { return ProviderID }

func (p *Provider) UpsertIndex(ctx context.Context, definition models.IndexDefinition) (models.SearchTask, error) {
	plan, err := makePlan(definition)
	if err != nil {
		return models.SearchTask{}, err
	}
	table, err := physicalTable(definition.PhysicalName)
	if err != nil {
		return models.SearchTask{}, err
	}
	old, err := p.catalog(ctx, definition.PhysicalName)
	if err != nil {
		return models.SearchTask{}, err
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return models.SearchTask{}, err
	}
	defer tx.Rollback(ctx)
	if old != nil && old.SchemaHash != plan.SchemaHash {
		if _, err = tx.Exec(ctx, `DROP TABLE IF EXISTS `+ident(old.Table)+` CASCADE`); err != nil {
			return models.SearchTask{}, err
		}
	}
	qtable := ident(table)
	for _, sql := range []string{
		`CREATE TABLE IF NOT EXISTS ` + qtable + ` (id text NOT NULL PRIMARY KEY,document jsonb NOT NULL,indexed jsonb NOT NULL DEFAULT '{}'::jsonb,search_vector tsvector NOT NULL DEFAULT ''::tsvector,search_text text NOT NULL DEFAULT '',created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS ` + ident(table+"_fts_idx") + ` ON ` + qtable + ` USING GIN(search_vector)`,
		`CREATE INDEX IF NOT EXISTS ` + ident(table+"_trgm_idx") + ` ON ` + qtable + ` USING GIN(search_text gin_trgm_ops)`,
		`CREATE INDEX IF NOT EXISTS ` + ident(table+"_indexed_idx") + ` ON ` + qtable + ` USING GIN(indexed jsonb_path_ops)`,
	} {
		if _, err = tx.Exec(ctx, sql); err != nil {
			return models.SearchTask{}, err
		}
	}
	planJSON, _ := marshalPlan(plan)
	if old == nil {
		_, err = tx.Exec(ctx, `INSERT INTO search_postgres_index(name,table_name,schema_hash,plan,document_count) VALUES($1,$2,$3,$4::jsonb,0)`, definition.PhysicalName, table, plan.SchemaHash, planJSON)
	} else {
		count := old.DocumentCount
		if old.SchemaHash != plan.SchemaHash {
			count = 0
		}
		_, err = tx.Exec(ctx, `UPDATE search_postgres_index SET table_name=$1,schema_hash=$2,plan=$3::jsonb,document_count=$4,updated_at=now() WHERE name=$5`, table, plan.SchemaHash, planJSON, count, definition.PhysicalName)
	}
	if err != nil {
		return models.SearchTask{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return models.SearchTask{}, err
	}
	return task(definition.PhysicalName), nil
}

func (p *Provider) DeleteIndex(ctx context.Context, name string) (models.SearchTask, error) {
	entry, err := p.catalog(ctx, name)
	if err != nil {
		return models.SearchTask{}, err
	}
	if entry != nil {
		tx, err := p.pool.Begin(ctx)
		if err != nil {
			return models.SearchTask{}, err
		}
		defer tx.Rollback(ctx)
		if _, err = tx.Exec(ctx, `DROP TABLE IF EXISTS `+ident(entry.Table)+` CASCADE`); err != nil {
			return models.SearchTask{}, err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM search_postgres_index WHERE name=$1`, name); err != nil {
			return models.SearchTask{}, err
		}
		if err = tx.Commit(ctx); err != nil {
			return models.SearchTask{}, err
		}
	}
	return task(name), nil
}

func (p *Provider) ListIndexes(ctx context.Context) ([]models.SearchIndexInfo, error) {
	rows, err := p.pool.Query(ctx, `SELECT name,document_count,created_at,updated_at FROM search_postgres_index ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.SearchIndexInfo{}
	for rows.Next() {
		var v models.SearchIndexInfo
		var created, updated time.Time
		if err = rows.Scan(&v.Name, &v.DocumentCount, &created, &updated); err != nil {
			return nil, err
		}
		v.Provider, v.CreatedAt, v.UpdatedAt = ProviderID, &created, &updated
		out = append(out, v)
	}
	return out, rows.Err()
}

func (p *Provider) UpsertDocuments(ctx context.Context, name string, _ models.IndexDefinition, documents []models.Document) (models.SearchTask, error) {
	entry, err := p.retrieve(ctx, name)
	if err != nil {
		return models.SearchTask{}, err
	}
	if len(documents) == 0 {
		return task(name), nil
	}
	unique := map[string]projectedDocument{}
	for _, doc := range documents {
		value, e := projectForIndex(doc, entry.Plan)
		if e != nil {
			return models.SearchTask{}, e
		}
		unique[value.id] = value
	}
	items := make([]projectedDocument, 0, len(unique))
	for _, item := range unique {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].id < items[j].id })
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return models.SearchTask{}, err
	}
	defer tx.Rollback(ctx)
	inserted := int64(0)
	for start := 0; start < len(items); start += 200 {
		end := min(start+200, len(items))
		chunk := items[start:end]
		values := make([]string, 0, len(chunk))
		args := []any{}
		for _, item := range chunk {
			base := len(args)
			args = append(args, item.id, jsonString(item.document), jsonString(item.indexed))
			weights := []string{}
			for _, part := range item.weighted {
				args = append(args, part.text)
				weights = append(weights, fmt.Sprintf(`setweight(to_tsvector('%s',coalesce($%d,'')),'%s')`, p.tsConfig, len(args), part.weight))
			}
			args = append(args, item.searchText)
			vector := `''::tsvector`
			if len(weights) > 0 {
				vector = strings.Join(weights, ` || `)
			}
			values = append(values, fmt.Sprintf(`($%d,$%d::jsonb,$%d::jsonb,%s,$%d,now())`, base+1, base+2, base+3, vector, len(args)))
		}
		rows, e := tx.Query(ctx, `INSERT INTO `+ident(entry.Table)+`(id,document,indexed,search_vector,search_text,updated_at) VALUES `+strings.Join(values, ",")+` ON CONFLICT(id) DO UPDATE SET document=EXCLUDED.document,indexed=EXCLUDED.indexed,search_vector=EXCLUDED.search_vector,search_text=EXCLUDED.search_text,updated_at=now() RETURNING (xmax=0)::int`, args...)
		if e != nil {
			return models.SearchTask{}, e
		}
		for rows.Next() {
			var n int
			if e = rows.Scan(&n); e != nil {
				rows.Close()
				return models.SearchTask{}, e
			}
			inserted += int64(n)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return models.SearchTask{}, e
		}
	}
	if inserted > 0 {
		if _, err = tx.Exec(ctx, `UPDATE search_postgres_index SET document_count=GREATEST(document_count+$1,0),updated_at=now() WHERE name=$2`, inserted, name); err != nil {
			return models.SearchTask{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.SearchTask{}, err
	}
	return task(name), nil
}

func (p *Provider) DeleteDocuments(ctx context.Context, name string, filters models.SearchFilters) (models.SearchTask, error) {
	entry, err := p.retrieve(ctx, name)
	if err != nil {
		return models.SearchTask{}, err
	}
	if len(filters) == 0 {
		return models.SearchTask{}, fmt.Errorf("deleting from search index %q requires filters", name)
	}
	where, args, err := compileFilters(filters, entry.Plan)
	if err != nil {
		return models.SearchTask{}, err
	}
	if where == "" {
		return task(name), nil
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return models.SearchTask{}, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `DELETE FROM `+ident(entry.Table)+` WHERE `+where+` RETURNING id`, args...)
	if err != nil {
		return models.SearchTask{}, err
	}
	var deleted int64
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return models.SearchTask{}, err
		}
		deleted++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return models.SearchTask{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE search_postgres_index SET document_count=GREATEST(document_count-$1,0),updated_at=now() WHERE name=$2`, deleted, name); err != nil {
		return models.SearchTask{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return models.SearchTask{}, err
	}
	return task(name), nil
}

func (p *Provider) ClearIndex(ctx context.Context, name string) (models.SearchTask, error) {
	entry, err := p.retrieve(ctx, name)
	if err != nil {
		return models.SearchTask{}, err
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return models.SearchTask{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `TRUNCATE TABLE `+ident(entry.Table)); err != nil {
		return models.SearchTask{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE search_postgres_index SET document_count=0,updated_at=now() WHERE name=$1`, name); err != nil {
		return models.SearchTask{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return models.SearchTask{}, err
	}
	return task(name), nil
}

func (p *Provider) Search(ctx context.Context, query models.ProviderSearchQuery) (models.SearchResult, error) {
	started := time.Now()
	name := query.Index.PhysicalName
	if name == "" {
		name = query.Index.Name
	}
	entry, err := p.retrieve(ctx, name)
	if err != nil {
		return models.SearchResult{}, err
	}
	if query.Pagination.Cursor != "" {
		return models.SearchResult{}, errors.New("the postgres search provider does not support cursor pagination")
	}
	options := query.SearchOptions
	if options == nil {
		options = map[string]any{}
	}
	if options["vector"] != nil {
		return models.SearchResult{}, errors.New(`vector search requires engine: "lakebase"; native engine does not support search_options.vector`)
	}
	if listLength(options["locales"]) > 0 {
		return models.SearchResult{}, errors.New("the postgres search provider does not support query-time locales; configure language on the provider")
	}
	if options["highlight"] != nil && strings.TrimSpace(query.Q) != "" {
		return models.SearchResult{}, errors.New("the postgres search provider does not support highlighting")
	}
	if listLength(options["facets"]) > 0 {
		return models.SearchResult{}, errors.New("search-postgres facet queries are not yet supported by this Go adapter")
	}
	if options["distinct"] != nil || query.Index.Settings["distinct_attribute"] != nil {
		return models.SearchResult{}, errors.New("search-postgres distinct results are not yet supported by this Go adapter")
	}
	searchable := entry.Plan.Searchable
	if rawAttrs := options["attributes_to_search_on"]; rawAttrs != nil {
		attrs, ok := stringList(rawAttrs)
		if !ok {
			return models.SearchResult{}, errors.New("attributes_to_search_on must be an array of field names")
		}
		searchable = []string{}
		for _, field := range attrs {
			if !contains(entry.Plan.Searchable, field) {
				return models.SearchResult{}, fmt.Errorf("field %q is not searchable on search index %q", field, query.Index.Name)
			}
			searchable = append(searchable, field)
		}
	}
	where, filterArgs, err := compileFilters(query.Filters, entry.Plan)
	if err != nil {
		return models.SearchResult{}, err
	}
	q := strings.TrimSpace(query.Q)
	strategy, _ := options["match_strategy"].(string)
	if strategy != "" && strategy != "all" && strategy != "any" && strategy != "last" {
		return models.SearchResult{}, fmt.Errorf("unsupported match strategy %q", strategy)
	}
	searchArgs := []any{}
	rank := `0::float4`
	match := ""
	if q != "" {
		vectorExpr, textExpr := searchExpressions(searchable, entry.Plan, p.tsConfig)
		searchArgs = append(searchArgs, q)
		tsq := tsQuery(p.tsConfig, strategy, len(searchArgs))
		rank = `ts_rank_cd(` + vectorExpr + `,` + tsq + `)`
		match = vectorExpr + ` @@ ` + tsq
		typo := typoRequested(options) && typoEnabled(query.Index.Settings)
		if typo {
			searchArgs = append(searchArgs, q)
			rank = `GREATEST(` + rank + `,word_similarity($` + strconv.Itoa(len(searchArgs)) + `,` + textExpr + `))`
			searchArgs = append(searchArgs, q)
			match = `(` + match + ` OR word_similarity($` + strconv.Itoa(len(searchArgs)) + `,` + textExpr + `)>0.3)`
		}
		if len(searchable) == 0 {
			return models.SearchResult{}, fmt.Errorf("search index %q has no searchable fields", query.Index.Name)
		}
	}
	conditions := []string{}
	if match != "" {
		conditions = append(conditions, match)
	}
	if where != "" {
		conditions = append(conditions, shiftPlaceholders(where, len(searchArgs)))
	}
	whereSQL := ""
	if len(conditions) > 0 {
		whereSQL = " WHERE " + strings.Join(conditions, " AND ")
	}
	args := append(searchArgs, filterArgs...)
	skip, take := intOr(query.Pagination.Skip, 0), intOr(query.Pagination.Take, 20)
	if skip < 0 || take < 0 {
		return models.SearchResult{}, errors.New("search pagination values cannot be negative")
	}
	order, err := searchOrder(query.Pagination.Order, entry.Plan, q != "")
	if err != nil {
		return models.SearchResult{}, err
	}
	minScore, hasMin := floatValue(options["min_score"])
	base := `SELECT id,document,` + rank + ` AS score FROM ` + ident(entry.Table) + whereSQL
	if hasMin {
		args = append(args, minScore)
		base = `SELECT * FROM (` + base + `) ranked WHERE score >= $` + strconv.Itoa(len(args))
	}
	args = append(args, take, skip)
	limitAt := len(args) - 1
	rows, err := p.pool.Query(ctx, base+` ORDER BY `+order+` LIMIT $`+strconv.Itoa(limitAt)+` OFFSET $`+strconv.Itoa(limitAt+1), args...)
	if err != nil {
		return models.SearchResult{}, err
	}
	hits := []models.SearchHit{}
	for rows.Next() {
		var id string
		var raw []byte
		var score float64
		if err = rows.Scan(&id, &raw, &score); err != nil {
			rows.Close()
			return models.SearchResult{}, err
		}
		doc := models.Document{}
		if err = json.Unmarshal(raw, &doc); err != nil {
			rows.Close()
			return models.SearchResult{}, err
		}
		hit := models.SearchHit{ID: id, Document: projectDocument(doc, query.AttributesToRetrieve)}
		if include, _ := options["include_score"].(bool); include {
			hit.Score = &score
		}
		hits = append(hits, hit)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return models.SearchResult{}, err
	}
	var count *int64
	if options["count"] != "none" {
		countArgs := append([]any{}, searchArgs...)
		countArgs = append(countArgs, filterArgs...)
		countSQL := `SELECT COUNT(*) FROM ` + ident(entry.Table) + whereSQL
		if hasMin {
			countArgs = append(countArgs, minScore)
			countSQL = `SELECT COUNT(*) FROM (SELECT ` + rank + ` AS score FROM ` + ident(entry.Table) + whereSQL + `) ranked WHERE score >= $` + strconv.Itoa(len(countArgs))
		}
		var n int64
		if err = p.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&n); err != nil {
			return models.SearchResult{}, err
		}
		count = &n
	}
	ms := float64(time.Since(started).Microseconds()) / 1000
	return models.SearchResult{Hits: hits, Metadata: models.ResultMetadata{Skip: skip, Take: take, Count: count, Query: query.Q, ProcessingTimeMS: &ms}}, nil
}
func (p *Provider) SearchMany(ctx context.Context, queries []models.ProviderSearchQuery) ([]models.SearchResult, error) {
	if len(queries) == 0 {
		return []models.SearchResult{}, nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	out := make([]models.SearchResult, len(queries))
	errs := make(chan error, len(queries))
	var wg sync.WaitGroup
	for i, query := range queries {
		wg.Add(1)
		go func(i int, query models.ProviderSearchQuery) {
			defer wg.Done()
			value, err := p.Search(ctx, query)
			if err != nil {
				errs <- err
				cancel()
				return
			}
			out[i] = value
		}(i, query)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (p *Provider) catalog(ctx context.Context, name string) (*catalogEntry, error) {
	var e catalogEntry
	var raw []byte
	err := p.pool.QueryRow(ctx, `SELECT name,table_name,schema_hash,plan,document_count,created_at,updated_at FROM search_postgres_index WHERE name=$1`, name).Scan(&e.Name, &e.Table, &e.SchemaHash, &raw, &e.DocumentCount, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.Plan, err = decodePlan(raw)
	return &e, err
}
func (p *Provider) retrieve(ctx context.Context, name string) (*catalogEntry, error) {
	e, err := p.catalog(ctx, name)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, fmt.Errorf("the postgres search provider has no index %q", name)
	}
	return e, nil
}

func makePlan(def models.IndexDefinition) (indexPlan, error) {
	if def.PrimaryKey == "" {
		return indexPlan{}, errors.New("search index primary_key is required")
	}
	if typo, ok := def.Settings["typo_tolerance"].(map[string]any); ok {
		if listLength(typo["disabled_on_attributes"]) > 0 {
			return indexPlan{}, errors.New("the postgres search provider matches typos across all searchable fields and does not support typo_tolerance.disabled_on_attributes")
		}
	}
	plan := indexPlan{PrimaryKey: def.PrimaryKey, Fields: map[string]plannedField{}}
	var walk func(map[string]models.FieldDefinition, string, bool) error
	walk = func(fields map[string]models.FieldDefinition, prefix string, array bool) error {
		for name, f := range fields {
			path := name
			if prefix != "" {
				path = prefix + "." + name
			}
			isArray := array || f.Array
			if f.Type == "object" {
				if err := walk(f.Fields, path, isArray); err != nil {
					return err
				}
				continue
			}
			kind := ""
			switch f.Type {
			case "text", "keyword":
				kind = f.Type
			case "integer", "float", "date":
				kind = "number"
			case "boolean":
				kind = "boolean"
			case "vector":
				continue
			case "geo":
				return fmt.Errorf("the postgres search provider does not support geo fields (%q)", path)
			default:
				return fmt.Errorf("the postgres search provider does not support fields of type %q", f.Type)
			}
			plan.Fields[path] = plannedField{Path: path, Kind: kind, Array: isArray, Date: f.Type == "date", Field: f}
			if searchable(f.Searchable) && kind != "boolean" {
				plan.Searchable = append(plan.Searchable, path)
			}
		}
		return nil
	}
	if err := walk(def.Fields, "", false); err != nil {
		return indexPlan{}, err
	}
	sort.Strings(plan.Searchable)
	paths := make([]string, 0, len(plan.Fields))
	for path := range plan.Fields {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	type fieldFingerprint struct {
		Path       string `json:"path"`
		Kind       string `json:"kind"`
		Array      bool   `json:"is_array"`
		Date       bool   `json:"is_date"`
		Dimensions int    `json:"dimensions,omitempty"`
		Embed      bool   `json:"embed,omitempty"`
		Searchable bool   `json:"searchable"`
		Filterable bool   `json:"filterable"`
		Sortable   bool   `json:"sortable"`
		Facetable  bool   `json:"facetable"`
	}
	list := []fieldFingerprint{}
	for _, path := range paths {
		field := plan.Fields[path]
		list = append(list, fieldFingerprint{Path: path, Kind: field.Kind, Array: field.Array, Date: field.Date, Dimensions: field.Field.Dimensions, Embed: field.Field.Embed, Searchable: searchable(field.Field.Searchable), Filterable: field.Field.Filterable, Sortable: field.Field.Sortable, Facetable: searchable(field.Field.Facetable)})
	}
	schema := struct {
		PrimaryKey string             `json:"primary_key"`
		Fields     []fieldFingerprint `json:"fields"`
		Searchable []string           `json:"searchable"`
		Vectors    []string           `json:"vectors"`
	}{PrimaryKey: plan.PrimaryKey, Fields: list, Searchable: plan.Searchable, Vectors: []string{}}
	encoded, _ := json.Marshal(schema)
	sum := sha256.Sum256(encoded)
	plan.SchemaHash = hex.EncodeToString(sum[:])
	return plan, nil
}
func marshalPlan(plan indexPlan) (string, error) {
	fields := make([]plannedField, 0, len(plan.Fields))
	for _, field := range plan.Fields {
		fields = append(fields, field)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Path < fields[j].Path })
	data := map[string]any{"primary_key": plan.PrimaryKey, "searchable": plan.Searchable, "vectors": []string{}, "schema_hash": plan.SchemaHash, "fields": fields}
	b, err := json.Marshal(data)
	return string(b), err
}
func decodePlan(raw []byte) (indexPlan, error) {
	var value struct {
		PrimaryKey string         `json:"primary_key"`
		Searchable []string       `json:"searchable"`
		Vectors    []string       `json:"vectors"`
		SchemaHash string         `json:"schema_hash"`
		Fields     []plannedField `json:"fields"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return indexPlan{}, err
	}
	plan := indexPlan{PrimaryKey: value.PrimaryKey, Searchable: value.Searchable, Vectors: value.Vectors, SchemaHash: value.SchemaHash, Fields: map[string]plannedField{}}
	for _, f := range value.Fields {
		plan.Fields[f.Path] = f
	}
	return plan, nil
}

var _ domain.SearchProvider = (*Provider)(nil)
var _ domain.OptionalSearchMany = (*Provider)(nil)

func physicalTable(name string) (string, error) {
	safe := regexp.MustCompile(`[^a-z0-9_]+`).ReplaceAllString(strings.ToLower(name), "_")
	safe = regexp.MustCompile(`_+`).ReplaceAllString(safe, "_")
	safe = strings.Trim(safe, "_")
	if safe == "" {
		return "", fmt.Errorf("cannot derive a table name from index %q", name)
	}
	return "search_pg_" + safe, nil
}
func ident(name string) string           { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }
func task(name string) models.SearchTask { return models.SearchTask{Index: name, Status: "succeeded"} }
func jsonString(value any) string        { b, _ := json.Marshal(value); return string(b) }
func searchable(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case map[string]any:
		return true
	}
	return false
}
func contains(values []string, s string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}
func stringList(value any) ([]string, bool) {
	switch list := value.(type) {
	case []string:
		return list, true
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, text)
		}
		return out, true
	default:
		return nil, false
	}
}
func listLength(value any) int {
	switch list := value.(type) {
	case []any:
		return len(list)
	case []string:
		return len(list)
	}
	return 0
}
func intOr(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}
func floatValue(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		x, e := n.Float64()
		return x, e == nil
	case string:
		x, e := strconv.ParseFloat(n, 64)
		return x, e == nil
	}
	return 0, false
}
