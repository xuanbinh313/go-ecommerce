package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/translation/domain"
	"github.com/example/medusa-go-clone/internal/modules/translation/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }
func scan(row pgx.Row) (models.Translation, error) {
	var v models.Translation
	var raw []byte
	err := row.Scan(&v.ID, &v.ReferenceID, &v.Reference, &v.LocaleCode, &raw, &v.TranslatedFieldCount, &v.CreatedAt, &v.UpdatedAt)
	if err == nil {
		err = json.Unmarshal(raw, &v.Translations)
	}
	return v, err
}

const cols = `id,reference_id,reference,locale_code,translations,translated_field_count,created_at,updated_at`

func (r *Repository) List(ctx context.Context, f models.Filter) ([]models.Translation, int, error) {
	where, args := []string{"deleted_at IS NULL"}, []any{}
	for _, x := range []struct {
		col    string
		values []string
	}{{"reference_id", f.ReferenceIDs}, {"reference", f.References}, {"locale_code", f.LocaleCodes}} {
		if len(x.values) > 0 {
			args = append(args, x.values)
			where = append(where, fmt.Sprintf("%s=ANY($%d)", x.col, len(args)))
		}
	}
	w := " WHERE "
	for i, x := range where {
		if i > 0 {
			w += " AND "
		}
		w += x
	}
	var count int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM translation"+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf("SELECT %s FROM translation%s ORDER BY updated_at DESC LIMIT $%d OFFSET $%d", cols, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.Translation{}
	for rows.Next() {
		v, e := scan(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) Batch(ctx context.Context, in models.BatchInput) ([]models.Translation, []models.Translation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)
	created, updated := []models.Translation{}, []models.Translation{}
	for _, v := range in.Create {
		raw, e := json.Marshal(v.Translations)
		if e != nil {
			return nil, nil, e
		}
		if v.Translations == nil {
			raw = []byte("{}")
		}
		row, e := scan(tx.QueryRow(ctx, `INSERT INTO translation(id,reference_id,reference,locale_code,translations,translated_field_count) VALUES($1,$2,$3,$4,$5,$6) RETURNING `+cols, entityid.New("trans"), v.ReferenceID, v.Reference, v.LocaleCode, raw, len(v.Translations)))
		if e != nil {
			return nil, nil, e
		}
		created = append(created, row)
	}
	for _, v := range in.Update {
		sets, args := []string{}, []any{v.ID}
		put := func(col string, value any) {
			args = append(args, value)
			sets = append(sets, fmt.Sprintf("%s=$%d", col, len(args)))
		}
		if v.ReferenceID != nil {
			put("reference_id", *v.ReferenceID)
		}
		if v.Reference != nil {
			put("reference", *v.Reference)
		}
		if v.LocaleCode != nil {
			put("locale_code", *v.LocaleCode)
		}
		if v.Translations != nil {
			var old map[string]string
			err = tx.QueryRow(ctx, "SELECT translations FROM translation WHERE id=$1 AND deleted_at IS NULL FOR UPDATE", v.ID).Scan(&old)
			if err != nil {
				return nil, nil, err
			}
			for k, x := range v.Translations {
				old[k] = x
			}
			raw, e := json.Marshal(old)
			if e != nil {
				return nil, nil, e
			}
			put("translations", raw)
			put("translated_field_count", len(old))
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			sql := "UPDATE translation SET "
			for i, x := range sets {
				if i > 0 {
					sql += ","
				}
				sql += x
			}
			sql += " WHERE id=$1 AND deleted_at IS NULL RETURNING " + cols
			row, e := scan(tx.QueryRow(ctx, sql, args...))
			if e != nil {
				return nil, nil, e
			}
			updated = append(updated, row)
		}
	}
	if len(in.Delete) > 0 {
		if _, err = tx.Exec(ctx, "UPDATE translation SET deleted_at=now(),updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL", in.Delete); err != nil {
			return nil, nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return created, updated, nil
}

const settingCols = `id,entity_type,fields,is_active,created_at,updated_at`

func scanSetting(row pgx.Row) (models.TranslationSetting, error) {
	var setting models.TranslationSetting
	var fields []byte
	err := row.Scan(&setting.ID, &setting.EntityType, &fields, &setting.IsActive, &setting.CreatedAt, &setting.UpdatedAt)
	if err == nil {
		err = json.Unmarshal(fields, &setting.Fields)
	}
	return setting, err
}

func (r *Repository) ListSettings(ctx context.Context, filter models.TranslationSettingsFilter) ([]models.TranslationSetting, error) {
	where, args := []string{"deleted_at IS NULL"}, []any{}
	if len(filter.IDs) > 0 {
		args = append(args, filter.IDs)
		where = append(where, fmt.Sprintf("id=ANY($%d)", len(args)))
	}
	if filter.EntityType != nil {
		args = append(args, *filter.EntityType)
		where = append(where, fmt.Sprintf("entity_type=$%d", len(args)))
	}
	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		where = append(where, fmt.Sprintf("is_active=$%d", len(args)))
	}
	rows, err := r.db.Query(ctx, `SELECT `+settingCols+` FROM translation_settings WHERE `+strings.Join(where, " AND ")+` ORDER BY entity_type`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.TranslationSetting{}
	for rows.Next() {
		row, err := scanSetting(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) CreateSettings(ctx context.Context, input []models.CreateTranslationSettingInput) ([]models.TranslationSetting, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := make([]models.TranslationSetting, 0, len(input))
	for _, setting := range input {
		active := true
		if setting.IsActive != nil {
			active = *setting.IsActive
		}
		fields, err := json.Marshal(setting.Fields)
		if err != nil {
			return nil, err
		}
		row, err := scanSetting(tx.QueryRow(ctx, `INSERT INTO translation_settings(id,entity_type,fields,is_active) VALUES($1,$2,$3,$4) RETURNING `+settingCols, entityid.New("trset"), setting.EntityType, fields, active))
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) UpdateSettings(ctx context.Context, input []models.UpdateTranslationSettingInput) ([]models.TranslationSetting, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := make([]models.TranslationSetting, 0, len(input))
	for _, setting := range input {
		sets, args := []string{}, []any{setting.ID}
		put := func(column string, value any) {
			args = append(args, value)
			sets = append(sets, fmt.Sprintf("%s=$%d", column, len(args)))
		}
		if setting.EntityType != nil {
			put("entity_type", *setting.EntityType)
		}
		if setting.Fields != nil {
			value, err := json.Marshal(*setting.Fields)
			if err != nil {
				return nil, err
			}
			put("fields", value)
		}
		if setting.IsActive != nil {
			put("is_active", *setting.IsActive)
		}
		if len(sets) == 0 {
			row, err := scanSetting(tx.QueryRow(ctx, `SELECT `+settingCols+` FROM translation_settings WHERE id=$1 AND deleted_at IS NULL`, setting.ID))
			if err != nil {
				return nil, err
			}
			out = append(out, row)
			continue
		}
		sets = append(sets, "updated_at=now()")
		row, err := scanSetting(tx.QueryRow(ctx, "UPDATE translation_settings SET "+strings.Join(sets, ",")+" WHERE id=$1 AND deleted_at IS NULL RETURNING "+settingCols, args...))
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) DeleteSettings(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `UPDATE translation_settings SET deleted_at=now(),updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL`, ids)
	return err
}

func (r *Repository) RestoreSettings(ctx context.Context, settings []models.TranslationSetting) error {
	if len(settings) == 0 {
		return nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, setting := range settings {
		fields, err := json.Marshal(setting.Fields)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO translation_settings(id,entity_type,fields,is_active) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO UPDATE SET entity_type=EXCLUDED.entity_type,fields=EXCLUDED.fields,is_active=EXCLUDED.is_active,deleted_at=NULL,updated_at=now()`, setting.ID, setting.EntityType, fields, setting.IsActive)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

var _ domain.Repository = (*Repository)(nil)
