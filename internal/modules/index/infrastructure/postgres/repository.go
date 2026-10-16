package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/index/domain"
	"github.com/example/medusa-go-clone/internal/modules/index/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) EnsureEntityPartition(ctx context.Context, schema models.EntitySchema) error {
	if schema.Name == "" || schema.TableName == "" {
		return fmt.Errorf("index entity name and table name are required")
	}
	entityPartition := quoteIdentifier(schema.TableName)
	if _, err := r.db.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+entityPartition+` PARTITION OF index_data FOR VALUES IN (`+quoteLiteral(schema.Name)+`)`); err != nil {
		return err
	}
	if _, err := r.db.Exec(ctx, `CREATE INDEX IF NOT EXISTS `+quoteIdentifier("IDX_"+schema.TableName+"_data_gin")+` ON `+entityPartition+` USING GIN (data jsonb_path_ops)`); err != nil {
		return err
	}
	if _, err := r.db.Exec(ctx, `CREATE INDEX IF NOT EXISTS `+quoteIdentifier("IDX_"+schema.TableName+"_id")+` ON `+entityPartition+` (id)`); err != nil {
		return err
	}
	for _, relation := range schema.Relations {
		if relation.Pivot == "" || relation.TableName == "" {
			return fmt.Errorf("index relation pivot and table name are required")
		}
		table := quoteIdentifier(relation.TableName)
		if _, err := r.db.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+table+` PARTITION OF index_relation FOR VALUES IN (`+quoteLiteral(relation.Pivot)+`)`); err != nil {
			return err
		}
		if _, err := r.db.Exec(ctx, `CREATE INDEX IF NOT EXISTS `+quoteIdentifier("IDX_"+relation.TableName+"_child_id")+` ON `+table+` (child_id)`); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) UpsertData(ctx context.Context, rows []models.IndexData) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, row := range rows {
		if row.ID == "" || row.Name == "" {
			return fmt.Errorf("index data requires id and name")
		}
		data := row.Data
		if len(data) == 0 {
			data = json.RawMessage(`{}`)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO index_data(id,name,data,staled_at) VALUES($1,$2,$3,$4) ON CONFLICT(id,name) DO UPDATE SET data=EXCLUDED.data,staled_at=EXCLUDED.staled_at,updated_at=now()`, row.ID, row.Name, data, row.StaledAt); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) UpsertEntity(ctx context.Context, rows []models.IndexData, relations []models.IndexRelation) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, row := range rows {
		data := row.Data
		if len(data) == 0 {
			data = json.RawMessage(`{}`)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO index_data(id,name,data,staled_at) VALUES($1,$2,$3,$4) ON CONFLICT(id,name) DO UPDATE SET data=EXCLUDED.data,staled_at=EXCLUDED.staled_at,updated_at=now()`, row.ID, row.Name, data, row.StaledAt); err != nil {
			return err
		}
	}
	for _, rel := range relations {
		if _, err := tx.Exec(ctx, `INSERT INTO index_relation(pivot,parent_name,parent_id,child_name,child_id,staled_at) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(parent_id,child_id,child_name,parent_name,pivot) DO UPDATE SET staled_at=EXCLUDED.staled_at,updated_at=now()`, rel.Pivot, rel.ParentName, rel.ParentID, rel.ChildName, rel.ChildID, rel.StaledAt); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) DeleteData(ctx context.Context, name string, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM index_data WHERE name=$1 AND id=ANY($2::text[])`, name, ids)
	return err
}

func (r *Repository) DeleteRelationsForEntityIDs(ctx context.Context, name string, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM index_relation WHERE (parent_name=$1 AND parent_id=ANY($2::text[])) OR (child_name=$1 AND child_id=ANY($2::text[]))`, name, ids)
	return err
}

func (r *Repository) DeleteEntityRecords(ctx context.Context, name string, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `DELETE FROM index_data WHERE name=$1 AND id=ANY($2::text[])`, name, ids); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM index_relation WHERE (parent_name=$1 AND parent_id=ANY($2::text[])) OR (child_name=$1 AND child_id=ANY($2::text[]))`, name, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) DeleteEntity(ctx context.Context, name string, staleOnly bool) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	condition := `name=$1`
	if staleOnly {
		condition += ` AND staled_at IS NOT NULL`
	}
	rows, err := tx.Query(ctx, `SELECT id FROM index_data WHERE `+condition, name)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if _, err := tx.Exec(ctx, `DELETE FROM index_data WHERE `+condition, name); err != nil {
		return err
	}
	if len(ids) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM index_relation WHERE (parent_name=$1 AND parent_id=ANY($2::text[])) OR (child_name=$1 AND child_id=ANY($2::text[]))`, name, ids); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) MarkEntityStale(ctx context.Context, name string) error {
	_, err := r.db.Exec(ctx, `UPDATE index_data SET staled_at=now(),updated_at=now() WHERE name=$1`, name)
	return err
}

func (r *Repository) UpsertRelations(ctx context.Context, relations []models.IndexRelation) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, rel := range relations {
		_, err := tx.Exec(ctx, `INSERT INTO index_relation(pivot,parent_name,parent_id,child_name,child_id,staled_at) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(parent_id,child_id,child_name,parent_name,pivot) DO UPDATE SET staled_at=EXCLUDED.staled_at,updated_at=now()`, rel.Pivot, rel.ParentName, rel.ParentID, rel.ChildName, rel.ChildID, rel.StaledAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) DeleteRelations(ctx context.Context, relations []models.IndexRelation) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, rel := range relations {
		if _, err := tx.Exec(ctx, `DELETE FROM index_relation WHERE pivot=$1 AND parent_name=$2 AND parent_id=$3 AND child_name=$4 AND child_id=$5`, rel.Pivot, rel.ParentName, rel.ParentID, rel.ChildName, rel.ChildID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func scanSync(row pgx.Row) (models.IndexSync, error) {
	var item models.IndexSync
	err := row.Scan(&item.ID, &item.Entity, &item.LastKey, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt)
	return item, err
}
func (r *Repository) GetSync(ctx context.Context, entity string) (models.IndexSync, error) {
	item, err := scanSync(r.db.QueryRow(ctx, `SELECT id,entity,last_key,created_at,updated_at,deleted_at FROM index_sync WHERE entity=$1 AND deleted_at IS NULL`, entity))
	if err == pgx.ErrNoRows {
		return models.IndexSync{}, domain.ErrNotFound
	}
	return item, err
}
func (r *Repository) SaveSync(ctx context.Context, item models.IndexSync) error {
	if item.ID == "" {
		item.ID = entityid.New("idxsync")
	}
	_, err := r.db.Exec(ctx, `INSERT INTO index_sync(id,entity,last_key) VALUES($1,$2,$3) ON CONFLICT(entity) WHERE deleted_at IS NULL DO UPDATE SET last_key=EXCLUDED.last_key,updated_at=now()`, item.ID, item.Entity, item.LastKey)
	return err
}
func scanMetadata(row pgx.Row) (models.IndexMetadata, error) {
	var item models.IndexMetadata
	err := row.Scan(&item.ID, &item.Entity, &item.Fields, &item.FieldsHash, &item.Status, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt)
	return item, err
}
func (r *Repository) GetMetadata(ctx context.Context, entity string) (models.IndexMetadata, error) {
	item, err := scanMetadata(r.db.QueryRow(ctx, `SELECT id,entity,fields,fields_hash,status,created_at,updated_at,deleted_at FROM index_metadata WHERE entity=$1 AND deleted_at IS NULL`, entity))
	if err == pgx.ErrNoRows {
		return models.IndexMetadata{}, domain.ErrNotFound
	}
	return item, err
}

func (r *Repository) ListMetadata(ctx context.Context) ([]models.IndexMetadata, error) {
	rows, err := r.db.Query(ctx, `SELECT id,entity,fields,fields_hash,status,created_at,updated_at,deleted_at FROM index_metadata WHERE deleted_at IS NULL ORDER BY entity`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.IndexMetadata{}
	for rows.Next() {
		item, e := scanMetadata(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) DeleteMetadata(ctx context.Context, entities []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM index_metadata WHERE entity=ANY($1::text[])`, entities)
	return err
}
func (r *Repository) SaveMetadata(ctx context.Context, item models.IndexMetadata) error {
	if item.ID == "" {
		item.ID = entityid.New("idxmeta")
	}
	if item.Status == "" {
		item.Status = models.MetadataPending
	}
	_, err := r.db.Exec(ctx, `INSERT INTO index_metadata(id,entity,fields,fields_hash,status) VALUES($1,$2,$3,$4,$5) ON CONFLICT(entity) WHERE deleted_at IS NULL DO UPDATE SET fields=EXCLUDED.fields,fields_hash=EXCLUDED.fields_hash,status=EXCLUDED.status,updated_at=now()`, item.ID, item.Entity, item.Fields, item.FieldsHash, item.Status)
	return err
}

func quoteIdentifier(identifier string) string { return pgx.Identifier{identifier}.Sanitize() }
func quoteLiteral(value string) string         { return `'` + strings.ReplaceAll(value, `'`, `''`) + `'` }
