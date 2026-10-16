package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/rbac/domain"
	"github.com/example/medusa-go-clone/internal/modules/rbac/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }
func (r *Repository) ListRoles(ctx context.Context, limit, offset int) ([]models.Role, int, error) {
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM rbac_role WHERE deleted_at IS NULL`).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT id,name,description,metadata,created_at,updated_at FROM rbac_role WHERE deleted_at IS NULL ORDER BY created_at LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.Role{}
	for rows.Next() {
		var v models.Role
		if err := rows.Scan(&v.ID, &v.Name, &v.Description, &v.Metadata, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, v)
	}
	return items, count, rows.Err()
}
func (r *Repository) ListPolicies(ctx context.Context, limit, offset int) ([]models.Policy, int, error) {
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM rbac_policy WHERE deleted_at IS NULL`).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT id,key,resource,operation,name,description,metadata,created_at,updated_at FROM rbac_policy WHERE deleted_at IS NULL ORDER BY created_at LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.Policy{}
	for rows.Next() {
		var v models.Policy
		if err := rows.Scan(&v.ID, &v.Key, &v.Resource, &v.Operation, &v.Name, &v.Description, &v.Metadata, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, v)
	}
	return items, count, rows.Err()
}
func (r *Repository) GetRole(ctx context.Context, id string) (models.Role, error) {
	var v models.Role
	err := r.db.QueryRow(ctx, `SELECT id,name,description,metadata,created_at,updated_at FROM rbac_role WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&v.ID, &v.Name, &v.Description, &v.Metadata, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
func (r *Repository) GetPolicy(ctx context.Context, id string) (models.Policy, error) {
	var v models.Policy
	err := r.db.QueryRow(ctx, `SELECT id,key,resource,operation,name,description,metadata,created_at,updated_at FROM rbac_policy WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&v.ID, &v.Key, &v.Resource, &v.Operation, &v.Name, &v.Description, &v.Metadata, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
func (r *Repository) UpdateRole(ctx context.Context, id string, in models.RoleUpdate) (models.Role, error) {
	var out models.Role
	err := r.db.QueryRow(ctx, `UPDATE rbac_role SET name=COALESCE($2,name),description=CASE WHEN $3 THEN $4 ELSE description END,metadata=CASE WHEN $5 THEN $6::jsonb ELSE metadata END,updated_at=now() WHERE id=$1 AND deleted_at IS NULL RETURNING id,name,description,metadata,created_at,updated_at`, id, in.Name, in.DescriptionSet, in.Description, in.MetadataSet, in.Metadata).Scan(&out.ID, &out.Name, &out.Description, &out.Metadata, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}
func (r *Repository) UpdatePolicy(ctx context.Context, id string, in models.PolicyUpdate) (models.Policy, error) {
	var out models.Policy
	err := r.db.QueryRow(ctx, `UPDATE rbac_policy SET key=COALESCE($2,key),resource=COALESCE($3,resource),operation=COALESCE($4,operation),name=CASE WHEN $5 THEN $6 ELSE name END,description=CASE WHEN $7 THEN $8 ELSE description END,metadata=CASE WHEN $9 THEN $10::jsonb ELSE metadata END,updated_at=now() WHERE id=$1 AND deleted_at IS NULL RETURNING id,key,resource,operation,name,description,metadata,created_at,updated_at`, id, in.Key, in.Resource, in.Operation, in.NameSet, in.Name, in.DescriptionSet, in.Description, in.MetadataSet, in.Metadata).Scan(&out.ID, &out.Key, &out.Resource, &out.Operation, &out.Name, &out.Description, &out.Metadata, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}
func metadata(value json.RawMessage) json.RawMessage {
	if value == nil {
		return json.RawMessage("null")
	}
	return value
}
func (r *Repository) CreateRoles(ctx context.Context, inputs []models.RoleInput) ([]models.Role, error) {
	out := []models.Role{}
	for _, v := range inputs {
		id := entityid.New("role")
		if v.ID != nil {
			id = *v.ID
		}
		var x models.Role
		err := r.db.QueryRow(ctx, `INSERT INTO rbac_role(id,name,description,metadata) VALUES($1,$2,$3,$4) RETURNING id,name,description,metadata,created_at,updated_at`, id, v.Name, v.Description, metadata(v.Metadata)).Scan(&x.ID, &x.Name, &x.Description, &x.Metadata, &x.CreatedAt, &x.UpdatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, nil
}
func (r *Repository) CreatePolicies(ctx context.Context, inputs []models.PolicyInput) ([]models.Policy, error) {
	out := []models.Policy{}
	for _, v := range inputs {
		id := entityid.New("rpol")
		if v.ID != nil {
			id = *v.ID
		}
		key := v.Key
		if key == "" {
			key = v.Resource + ":" + v.Operation
		}
		var x models.Policy
		err := r.db.QueryRow(ctx, `INSERT INTO rbac_policy(id,key,resource,operation,name,description,metadata) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,key,resource,operation,name,description,metadata,created_at,updated_at`, id, key, v.Resource, v.Operation, v.Name, v.Description, metadata(v.Metadata)).Scan(&x.ID, &x.Key, &x.Resource, &x.Operation, &x.Name, &x.Description, &x.Metadata, &x.CreatedAt, &x.UpdatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, nil
}
func (r *Repository) CreateRolePolicies(ctx context.Context, inputs []models.RolePolicyInput) ([]models.RolePolicy, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := []models.RolePolicy{}
	for _, v := range inputs {
		id := entityid.New("rlpl")
		if v.ID != nil {
			id = *v.ID
		}
		var x models.RolePolicy
		err := tx.QueryRow(ctx, `INSERT INTO rbac_role_policy(id,role_id,policy_id,metadata) VALUES($1,$2,$3,$4) RETURNING id,role_id,policy_id,metadata,created_at,updated_at`, id, v.RoleID, v.PolicyID, metadata(v.Metadata)).Scan(&x.ID, &x.RoleID, &x.PolicyID, &x.Metadata, &x.CreatedAt, &x.UpdatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
func (r *Repository) ListRolePolicies(ctx context.Context, roleID string, limit, offset int) ([]models.RolePolicy, int, error) {
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM rbac_role_policy rp JOIN rbac_policy p ON p.id=rp.policy_id WHERE rp.role_id=$1 AND rp.deleted_at IS NULL AND p.deleted_at IS NULL`, roleID).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT rp.id,rp.role_id,rp.policy_id,rp.metadata,rp.created_at,rp.updated_at,rp.deleted_at,p.id,p.key,p.resource,p.operation,p.name,p.description,p.metadata,p.created_at,p.updated_at,p.deleted_at FROM rbac_role_policy rp JOIN rbac_policy p ON p.id=rp.policy_id WHERE rp.role_id=$1 AND rp.deleted_at IS NULL AND p.deleted_at IS NULL ORDER BY rp.created_at LIMIT $2 OFFSET $3`, roleID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.RolePolicy{}
	for rows.Next() {
		var item models.RolePolicy
		var policy models.Policy
		if err := rows.Scan(&item.ID, &item.RoleID, &item.PolicyID, &item.Metadata, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt, &policy.ID, &policy.Key, &policy.Resource, &policy.Operation, &policy.Name, &policy.Description, &policy.Metadata, &policy.CreatedAt, &policy.UpdatedAt, &policy.DeletedAt); err != nil {
			return nil, 0, err
		}
		item.Policy = &policy
		items = append(items, item)
	}
	return items, count, rows.Err()
}
func (r *Repository) CreateRoleParents(ctx context.Context, inputs []models.RoleParentInput) ([]models.RoleParent, error) {
	out := []models.RoleParent{}
	for _, v := range inputs {
		id := entityid.New("rlin")
		if v.ID != nil {
			id = *v.ID
		}
		var x models.RoleParent
		err := r.db.QueryRow(ctx, `INSERT INTO rbac_role_parent(id,role_id,parent_id,metadata) VALUES($1,$2,$3,$4) RETURNING id,role_id,parent_id,metadata,created_at,updated_at`, id, v.RoleID, v.ParentID, metadata(v.Metadata)).Scan(&x.ID, &x.RoleID, &x.ParentID, &x.Metadata, &x.CreatedAt, &x.UpdatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, nil
}
func (r *Repository) PolicyIDsExist(ctx context.Context, ids []string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT count(DISTINCT id) FROM rbac_policy WHERE id=ANY($1) AND deleted_at IS NULL`, ids).Scan(&count)
	return count == len(unique(ids)), err
}
func (r *Repository) RoleIDsExist(ctx context.Context, ids []string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT count(DISTINCT id) FROM rbac_role WHERE id=ANY($1) AND deleted_at IS NULL`, ids).Scan(&count)
	return count == len(unique(ids)), err
}
func (r *Repository) ExistingRoleIDs(ctx context.Context, ids []string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM rbac_role WHERE id=ANY($1) AND deleted_at IS NULL`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
func (r *Repository) WouldCreateCycle(ctx context.Context, roleID, parentID string) (bool, error) {
	var cycle bool
	err := r.db.QueryRow(ctx, `WITH RECURSIVE hierarchy(id,path) AS (SELECT id,ARRAY[id]::text[] FROM rbac_role WHERE id=$1 AND deleted_at IS NULL UNION ALL SELECT p.parent_id,h.path||p.parent_id FROM hierarchy h JOIN rbac_role_parent p ON p.role_id=h.id JOIN rbac_role r ON r.id=p.parent_id WHERE p.deleted_at IS NULL AND r.deleted_at IS NULL AND NOT p.parent_id=ANY(h.path)) SELECT EXISTS(SELECT 1 FROM hierarchy WHERE id=$2)`, parentID, roleID).Scan(&cycle)
	return cycle, err
}
func (r *Repository) ListPoliciesForRoles(ctx context.Context, ids []string) (map[string][]models.Policy, error) {
	out := map[string][]models.Policy{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `WITH RECURSIVE role_hierarchy AS (SELECT id,id AS original_role_id,ARRAY[id]::text[] AS path FROM rbac_role WHERE id=ANY($1) AND deleted_at IS NULL UNION ALL SELECT p.parent_id,h.original_role_id,h.path||p.parent_id FROM role_hierarchy h JOIN rbac_role_parent p ON p.role_id=h.id JOIN rbac_role parent ON parent.id=p.parent_id WHERE p.deleted_at IS NULL AND parent.deleted_at IS NULL AND NOT p.parent_id=ANY(h.path)) SELECT DISTINCT h.original_role_id,p.id,p.key,p.resource,p.operation,p.name,p.description,p.metadata,p.created_at,p.updated_at,CASE WHEN rp.role_id=h.original_role_id THEN NULL ELSE rp.role_id END FROM role_hierarchy h JOIN rbac_role_policy rp ON rp.role_id=h.id JOIN rbac_policy p ON p.id=rp.policy_id WHERE rp.deleted_at IS NULL AND p.deleted_at IS NULL ORDER BY h.original_role_id,p.resource,p.operation,p.key`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var roleID string
		var p models.Policy
		if err := rows.Scan(&roleID, &p.ID, &p.Key, &p.Resource, &p.Operation, &p.Name, &p.Description, &p.Metadata, &p.CreatedAt, &p.UpdatedAt, &p.InheritedFromRoleID); err != nil {
			return nil, err
		}
		out[roleID] = append(out[roleID], p)
	}
	return out, rows.Err()
}
func (r *Repository) DeleteRoles(ctx context.Context, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, table := range []string{"rbac_role_policy", "rbac_role_parent"} {
		if _, err = tx.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE role_id=ANY($1) OR %s=ANY($1)`, table, map[string]string{"rbac_role_policy": "policy_id", "rbac_role_parent": "parent_id"}[table]), ids); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM rbac_role WHERE id=ANY($1)`, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) DeletePolicies(ctx context.Context, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `DELETE FROM rbac_role_policy WHERE policy_id=ANY($1)`, ids); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM rbac_policy WHERE id=ANY($1)`, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) SoftDeleteRoles(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE rbac_role SET deleted_at=now(),updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL`, ids)
	return err
}
func (r *Repository) RestoreRoles(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE rbac_role SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1) AND deleted_at IS NOT NULL`, ids)
	return err
}
func (r *Repository) SoftDeletePolicies(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE rbac_policy SET deleted_at=now(),updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL`, ids)
	return err
}
func (r *Repository) RestorePolicies(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE rbac_policy SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1) AND deleted_at IS NOT NULL`, ids)
	return err
}
func (r *Repository) DeleteRolePolicies(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM rbac_role_policy WHERE id=ANY($1)`, ids)
	return err
}
func (r *Repository) DeleteRoleParents(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM rbac_role_parent WHERE id=ANY($1)`, ids)
	return err
}
func unique(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

var _ domain.Repository = (*Repository)(nil)
