package postgres

import (
	"context"
	"encoding/json"

	"github.com/example/medusa-go-clone/internal/modules/search/domain"
	"github.com/example/medusa-go-clone/internal/modules/search/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func scanIndex(row pgx.Row) (models.SearchIndex, error) {
	var v models.SearchIndex
	err := row.Scan(&v.ID, &v.Name, &v.ActiveVersion, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	return v, err
}

const indexCols = `id,name,active_version,created_at,updated_at,deleted_at`

func (r *Repository) GetIndex(ctx context.Context, name string) (models.SearchIndex, error) {
	v, err := scanIndex(r.db.QueryRow(ctx, `SELECT `+indexCols+` FROM search_index WHERE name=$1 AND deleted_at IS NULL`, name))
	if err == pgx.ErrNoRows {
		return models.SearchIndex{}, domain.ErrNotFound
	}
	return v, err
}
func (r *Repository) ListIndexes(ctx context.Context, ids []string) ([]models.SearchIndex, error) {
	rows, err := r.db.Query(ctx, `SELECT `+indexCols+` FROM search_index WHERE deleted_at IS NULL AND (cardinality($1::text[])=0 OR id=ANY($1)) ORDER BY name`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.SearchIndex{}
	for rows.Next() {
		v, e := scanIndex(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *Repository) CreateIndex(ctx context.Context, in models.SearchIndex) (models.SearchIndex, error) {
	if in.ID == "" {
		in.ID = entityid.New("srhidx")
	}
	return scanIndex(r.db.QueryRow(ctx, `INSERT INTO search_index(id,name,active_version) VALUES($1,$2,$3) RETURNING `+indexCols, in.ID, in.Name, in.ActiveVersion))
}
func (r *Repository) UpdateActiveVersion(ctx context.Context, id string, version *int) error {
	ct, err := r.db.Exec(ctx, `UPDATE search_index SET active_version=$2,updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, version)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
func scanVersion(row pgx.Row) (models.SearchIndexVersion, error) {
	var v models.SearchIndexVersion
	err := row.Scan(&v.ID, &v.Version, &v.Provider, &v.PhysicalName, &v.DefinitionHash, &v.Status, &v.SearchIndexID, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	return v, err
}

const versionCols = `id,version,provider,physical_name,definition_hash,status,search_index_id,created_at,updated_at,deleted_at`

func (r *Repository) CreateVersion(ctx context.Context, in models.SearchIndexVersion) (models.SearchIndexVersion, error) {
	if in.ID == "" {
		in.ID = entityid.New("srhver")
	}
	return scanVersion(r.db.QueryRow(ctx, `INSERT INTO search_index_version(id,version,provider,physical_name,definition_hash,status,search_index_id) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+versionCols, in.ID, in.Version, in.Provider, in.PhysicalName, in.DefinitionHash, in.Status, in.SearchIndexID))
}
func (r *Repository) ListVersions(ctx context.Context, ids []string) ([]models.SearchIndexVersion, error) {
	rows, err := r.db.Query(ctx, `SELECT `+versionCols+` FROM search_index_version WHERE deleted_at IS NULL AND search_index_id=ANY($1) ORDER BY version`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.SearchIndexVersion{}
	for rows.Next() {
		v, e := scanVersion(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func scanSync(row pgx.Row) (models.SearchIndexSync, error) {
	var v models.SearchIndexSync
	err := row.Scan(&v.ID, &v.JobID, &v.Status, &v.Filters, &v.LastKey, &v.Resumable, &v.DocumentsSynced, &v.StartedAt, &v.CompletedAt, &v.Error, &v.SearchIndexVersionID, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	return v, err
}

const syncCols = `id,job_id,status,filters,last_key,resumable,documents_synced,started_at,completed_at,error,search_index_version_id,created_at,updated_at,deleted_at`

func (r *Repository) CreateSync(ctx context.Context, in models.SearchIndexSync) (models.SearchIndexSync, error) {
	if in.ID == "" {
		in.ID = entityid.New("srhsync")
	}
	if in.Status == "" {
		in.Status = models.SyncPending
	}
	raw := any(in.Filters)
	if len(in.Filters) == 0 {
		raw = nil
	}
	return scanSync(r.db.QueryRow(ctx, `INSERT INTO search_index_sync(id,job_id,status,filters,last_key,resumable,documents_synced,started_at,completed_at,error,search_index_version_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING `+syncCols, in.ID, in.JobID, in.Status, raw, in.LastKey, in.Resumable, in.DocumentsSynced, in.StartedAt, in.CompletedAt, in.Error, in.SearchIndexVersionID))
}
func (r *Repository) UpdateSync(ctx context.Context, in models.SearchIndexSync) error {
	ct, err := r.db.Exec(ctx, `UPDATE search_index_sync SET job_id=$2,status=$3,filters=$4,last_key=$5,resumable=$6,documents_synced=$7,started_at=$8,completed_at=$9,error=$10,updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, in.ID, in.JobID, in.Status, jsonOrNull(in.Filters), in.LastKey, in.Resumable, in.DocumentsSynced, in.StartedAt, in.CompletedAt, in.Error)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
func (r *Repository) ListSyncs(ctx context.Context, ids []string) ([]models.SearchIndexSync, error) {
	rows, err := r.db.Query(ctx, `SELECT `+syncCols+` FROM search_index_sync WHERE deleted_at IS NULL AND search_index_version_id=ANY($1) ORDER BY created_at`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.SearchIndexSync{}
	for rows.Next() {
		v, e := scanSync(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func jsonOrNull(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	return v
}

// Soft-delete support is used by the Search delete-index operation, which drops
// physical provider indexes before removing bookkeeping rows.
func (r *Repository) DeleteIndexRecords(ctx context.Context, name string) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var id string
	if err = tx.QueryRow(ctx, `SELECT id FROM search_index WHERE name=$1 AND deleted_at IS NULL FOR UPDATE`, name).Scan(&id); err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	var deletedVersions int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM search_index_version WHERE search_index_id=$1`, id).Scan(&deletedVersions); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM search_index_sync WHERE search_index_version_id IN (SELECT id FROM search_index_version WHERE search_index_id=$1)`, id); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM search_index_version WHERE search_index_id=$1`, id); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM search_index WHERE id=$1`, id); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return deletedVersions, nil
}
