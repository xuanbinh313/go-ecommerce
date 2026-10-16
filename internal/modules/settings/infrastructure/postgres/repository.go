package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/settings/domain"
	"github.com/example/medusa-go-clone/internal/modules/settings/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) ListLayoutConfigurations(ctx context.Context, userID string, filter models.LayoutConfigurationFilter, limit, offset int) ([]models.LayoutConfiguration, int, error) {
	where := []string{"deleted_at IS NULL", "(user_id=$1 OR is_system_default=true)"}
	args := []any{userID}
	add := func(column string, values []string) {
		if len(values) > 0 {
			args = append(args, values)
			where = append(where, fmt.Sprintf("%s=ANY($%d)", column, len(args)))
		}
	}
	add("id", filter.IDs)
	add("zone", filter.Zones)
	if len(filter.UserIDs) > 0 {
		add("user_id", filter.UserIDs)
	}
	if filter.IsSystemDefault != nil {
		args = append(args, *filter.IsSystemDefault)
		where = append(where, fmt.Sprintf("is_system_default=$%d", len(args)))
	}
	whereSQL := " WHERE " + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM layout_configuration`+whereSQL, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT id,zone,user_id,is_system_default,configuration,created_at,updated_at FROM layout_configuration%s ORDER BY created_at LIMIT $%d OFFSET $%d`, whereSQL, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.LayoutConfiguration{}
	for rows.Next() {
		var item models.LayoutConfiguration
		if err := rows.Scan(&item.ID, &item.Zone, &item.UserID, &item.IsSystemDefault, &item.Configuration, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, count, rows.Err()
}

func (r *Repository) GetLayoutConfiguration(ctx context.Context, zone, userID string) (models.LayoutConfigurations, error) {
	personal, personalErr := r.getLayout(ctx, zone, userID, false)
	if personalErr != nil && !errors.Is(personalErr, pgx.ErrNoRows) {
		return models.LayoutConfigurations{}, personalErr
	}
	defaults, defaultErr := r.getLayout(ctx, zone, "", true)
	if defaultErr != nil && !errors.Is(defaultErr, pgx.ErrNoRows) {
		return models.LayoutConfigurations{}, defaultErr
	}
	var storedScope string
	var value json.RawMessage
	err := r.db.QueryRow(ctx, `SELECT value FROM user_preference WHERE user_id=$1 AND key=$2 AND deleted_at IS NULL`, userID, "active_layout."+zone).Scan(&value)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return models.LayoutConfigurations{}, err
	}
	if err == nil {
		var pref struct {
			Scope string `json:"scope"`
		}
		if json.Unmarshal(value, &pref) == nil {
			storedScope = pref.Scope
		}
	}
	scope := "default"
	if storedScope == "personal" && personalErr == nil {
		scope = "personal"
	} else if storedScope == "default" {
		scope = "default"
	} else if personalErr == nil {
		scope = "personal"
	}
	result := models.LayoutConfigurations{ActiveScope: scope}
	if personalErr == nil {
		result.PersonalConfiguration = &personal
	}
	if defaultErr == nil {
		result.DefaultConfiguration = &defaults
	}
	return result, nil
}

func (r *Repository) getLayout(ctx context.Context, zone, userID string, systemDefault bool) (models.LayoutConfiguration, error) {
	var item models.LayoutConfiguration
	var user any
	if userID != "" {
		user = userID
	}
	err := r.db.QueryRow(ctx, `SELECT id,zone,user_id,is_system_default,configuration,created_at,updated_at FROM layout_configuration WHERE zone=$1 AND user_id IS NOT DISTINCT FROM $2 AND is_system_default=$3 AND deleted_at IS NULL LIMIT 1`, zone, user, systemDefault).Scan(&item.ID, &item.Zone, &item.UserID, &item.IsSystemDefault, &item.Configuration, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

// SetLayoutConfiguration implements the set-layout-configuration workflow's persistence step.
func (r *Repository) SetLayoutConfiguration(ctx context.Context, zone, userID string, isDefault bool, configuration []byte) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var user any
	if !isDefault {
		user = userID
	}
	if isDefault {
		_, err = tx.Exec(ctx, `INSERT INTO layout_configuration(id,zone,user_id,is_system_default,configuration) VALUES($1,$2,NULL,true,$3) ON CONFLICT (zone) WHERE is_system_default=true AND deleted_at IS NULL DO UPDATE SET configuration=EXCLUDED.configuration,updated_at=now()`, entityid.New("layconf"), zone, configuration)
	} else {
		_, err = tx.Exec(ctx, `INSERT INTO layout_configuration(id,zone,user_id,is_system_default,configuration) VALUES($1,$2,$3,false,$4) ON CONFLICT (zone,user_id) WHERE deleted_at IS NULL DO UPDATE SET configuration=EXCLUDED.configuration,updated_at=now()`, entityid.New("layconf"), zone, user, configuration)
	}
	if err != nil {
		return err
	}
	{
		scope := "personal"
		if isDefault {
			scope = "default"
		}
		pref, _ := json.Marshal(map[string]string{"scope": scope})
		_, err = tx.Exec(ctx, `INSERT INTO user_preference(id,user_id,key,value) VALUES($1,$2,$3,$4) ON CONFLICT (user_id,key) WHERE deleted_at IS NULL DO UPDATE SET value=EXCLUDED.value,updated_at=now()`, entityid.New("userpref"), userID, "active_layout."+zone, pref)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ClearLayoutConfiguration implements clear-layout-configuration; only the actor's personal layout is removed.
func (r *Repository) ClearLayoutConfiguration(ctx context.Context, zone, userID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE layout_configuration SET deleted_at=now(),updated_at=now() WHERE zone=$1 AND user_id=$2 AND is_system_default=false AND deleted_at IS NULL`, zone, userID); err != nil {
		return err
	}
	pref, _ := json.Marshal(map[string]any{"scope": nil})
	if _, err = tx.Exec(ctx, `INSERT INTO user_preference(id,user_id,key,value) VALUES($1,$2,$3,$4) ON CONFLICT (user_id,key) WHERE deleted_at IS NULL DO UPDATE SET value=EXCLUDED.value,updated_at=now()`, entityid.New("userpref"), userID, "active_layout."+zone, pref); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ domain.Repository = (*Repository)(nil)
