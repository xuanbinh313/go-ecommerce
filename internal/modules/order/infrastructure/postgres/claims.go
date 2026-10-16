package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/order/domain"
	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
)

const claimColumns = `id,order_version,display_id,type,no_notification,raw_refund_amount,created_by,canceled_at,metadata,order_id,return_id,created_at,updated_at,deleted_at`

func scanClaim(row pgx.Row) (models.Claim, error) {
	var c models.Claim
	err := row.Scan(&c.ID, &c.OrderVersion, &c.DisplayID, &c.Type, &c.NoNotification, &c.RefundAmount, &c.CreatedBy, &c.CanceledAt, &c.Metadata, &c.OrderID, &c.ReturnID, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	return c, err
}
func (r *Repository) GetClaim(ctx context.Context, id string) (models.Claim, error) {
	claim, err := scanClaim(r.db.QueryRow(ctx, `SELECT `+claimColumns+` FROM order_claim WHERE id=$1 AND deleted_at IS NULL`, id))
	if err != nil {
		return claim, err
	}
	claim.AdditionalItems = []models.ClaimItem{}
	claim.ClaimItems = []models.ClaimItem{}
	items, err := r.listClaimItems(ctx, id)
	if err != nil {
		return claim, err
	}
	for _, item := range items {
		if item.IsAdditionalItem {
			claim.AdditionalItems = append(claim.AdditionalItems, item)
		} else {
			claim.ClaimItems = append(claim.ClaimItems, item)
		}
	}
	return claim, nil
}

func (r *Repository) listClaimItems(ctx context.Context, claimID string) ([]models.ClaimItem, error) {
	rows, err := r.db.Query(ctx, `SELECT id,claim_id,item_id,is_additional_item,reason,raw_quantity,note,metadata FROM order_claim_item WHERE claim_id=$1 AND deleted_at IS NULL ORDER BY created_at`, claimID)
	if err != nil {
		return nil, err
	}
	items := []models.ClaimItem{}
	ids := []string{}
	for rows.Next() {
		var v models.ClaimItem
		if err = rows.Scan(&v.ID, &v.ClaimID, &v.ItemID, &v.IsAdditionalItem, &v.Reason, &v.Quantity, &v.Note, &v.Metadata); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, v)
		ids = append(ids, v.ID)
	}
	err = rows.Err()
	rows.Close()
	if err != nil || len(ids) == 0 {
		return items, err
	}
	images, err := r.db.Query(ctx, `SELECT id,claim_item_id,url,metadata FROM order_claim_item_image WHERE claim_item_id=ANY($1) AND deleted_at IS NULL ORDER BY created_at`, ids)
	if err != nil {
		return nil, err
	}
	imageMap := map[string][]models.ClaimItemImage{}
	for images.Next() {
		var v models.ClaimItemImage
		if err = images.Scan(&v.ID, &v.ClaimItemID, &v.URL, &v.Metadata); err != nil {
			images.Close()
			return nil, err
		}
		imageMap[v.ClaimItemID] = append(imageMap[v.ClaimItemID], v)
	}
	err = images.Err()
	images.Close()
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Images = imageMap[items[i].ID]
	}
	return items, nil
}

func (r *Repository) CreateClaim(ctx context.Context, in models.CreateClaimInput) (models.Claim, error) {
	var version int
	var status string
	err := r.db.QueryRow(ctx, `SELECT version,status FROM "order" WHERE id=$1 AND deleted_at IS NULL`, in.OrderID).Scan(&version, &status)
	if err != nil {
		return models.Claim{}, err
	}
	if status == "canceled" {
		return models.Claim{}, domain.InvalidData("Order is canceled and cannot have a claim")
	}
	metadata := in.Metadata
	if len(metadata) == 0 {
		metadata = json.RawMessage(`null`)
	}
	id := entityid.New("claim")
	claim, err := scanClaim(r.db.QueryRow(ctx, `INSERT INTO order_claim(id,order_version,type,created_by,metadata,order_id) VALUES($1,$2,$3,$4,$5,$6) RETURNING `+claimColumns, id, version, in.Type, in.CreatedBy, metadata, in.OrderID))
	return claim, err
}
func (r *Repository) CreateOrderChangeForClaim(ctx context.Context, in models.CreateClaimInput, claimID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var version int
	var status string
	if err = tx.QueryRow(ctx, `SELECT version,status FROM "order" WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, in.OrderID).Scan(&version, &status); err != nil {
		return err
	}
	if status == "canceled" {
		return domain.InvalidData("Order is canceled and cannot have a claim")
	}
	var active bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM order_change WHERE order_id=$1 AND status IN ('pending','requested') AND deleted_at IS NULL)`, in.OrderID).Scan(&active); err != nil {
		return err
	}
	if active {
		return domain.InvalidData("Order already has an active order change")
	}
	_, err = tx.Exec(ctx, `INSERT INTO order_change(id,order_id,claim_id,version,change_type,status,description,internal_note,created_by) VALUES($1,$2,$3,$4,'claim','pending',$5,$6,$7)`, entityid.New("ordch"), in.OrderID, claimID, version+1, in.Description, in.InternalNote, in.CreatedBy)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) SoftDeleteClaims(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE order_claim SET deleted_at=now(),updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL`, ids)
	return err
}

func (r *Repository) CancelBeginClaim(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var orderID string
	var returnID *string
	var claimCanceled *time.Time
	var orderStatus string
	err = tx.QueryRow(ctx, `SELECT c.order_id,c.return_id,c.canceled_at,o.status FROM order_claim c JOIN "order" o ON o.id=c.order_id WHERE c.id=$1 AND c.deleted_at IS NULL AND o.deleted_at IS NULL FOR UPDATE OF c,o`, id).Scan(&orderID, &returnID, &claimCanceled, &orderStatus)
	if err != nil {
		return err
	}
	if orderStatus == "canceled" || claimCanceled != nil {
		return domain.InvalidData("Order or claim is canceled")
	}
	if returnID != nil {
		return domain.InvalidData("Claims with a return cannot be canceled by this implementation")
	}
	var changeID string
	err = tx.QueryRow(ctx, `SELECT id FROM order_change WHERE order_id=$1 AND claim_id=$2 AND status IN ('pending','requested') AND deleted_at IS NULL`, orderID, id).Scan(&changeID)
	if err != nil {
		return err
	}
	var hasShipping bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM order_change_action WHERE order_change_id=$1 AND action='SHIPPING_ADD' AND deleted_at IS NULL)`, changeID).Scan(&hasShipping); err != nil {
		return err
	}
	if hasShipping {
		return domain.InvalidData("Claim shipping-method cleanup is not available")
	}
	if _, err = tx.Exec(ctx, `UPDATE order_change_action SET deleted_at=now(),updated_at=now() WHERE order_change_id=$1 AND deleted_at IS NULL`, changeID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE order_change SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, changeID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE order_claim_item_image SET deleted_at=now(),updated_at=now() WHERE claim_item_id IN (SELECT id FROM order_claim_item WHERE claim_id=$1) AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE order_claim_item SET deleted_at=now(),updated_at=now() WHERE claim_id=$1 AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE order_claim SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) ListClaims(ctx context.Context, f models.ClaimFilter) ([]models.Claim, int, error) {
	where, args := []string{}, []any{}
	if f.DeletedAt == nil {
		where = append(where, "deleted_at IS NULL")
	}
	for _, filter := range []struct {
		column string
		values []string
	}{{"id", f.IDs}, {"order_id", f.OrderIDs}} {
		if len(filter.values) > 0 {
			args = append(args, filter.values)
			where = append(where, fmt.Sprintf("%s=ANY($%d)", filter.column, len(args)))
		}
	}
	for _, filter := range []struct {
		column string
		values map[string]string
	}{{"created_at", f.CreatedAt}, {"updated_at", f.UpdatedAt}, {"deleted_at", f.DeletedAt}} {
		for op, value := range filter.values {
			operator := map[string]string{"$gt": ">", "$gte": ">=", "$lt": "<", "$lte": "<=", "$eq": "=", "$ne": "!="}[op]
			if operator == "" {
				continue
			}
			args = append(args, value)
			where = append(where, fmt.Sprintf("%s %s $%d", filter.column, operator, len(args)))
		}
	}
	w := ""
	if len(where) > 0 {
		w = " WHERE " + strings.Join(where, " AND ")
	}
	var count int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM order_claim"+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 15
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf("SELECT %s FROM order_claim%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", claimColumns, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	out := []models.Claim{}
	for rows.Next() {
		v, e := scanClaim(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, 0, err
	}
	rows.Close()
	if f.IncludeItems {
		for i := range out {
			out[i].AdditionalItems = []models.ClaimItem{}
			out[i].ClaimItems = []models.ClaimItem{}
			items, loadErr := r.listClaimItems(ctx, out[i].ID)
			if loadErr != nil {
				return nil, 0, loadErr
			}
			for _, item := range items {
				if item.IsAdditionalItem {
					out[i].AdditionalItems = append(out[i].AdditionalItems, item)
				} else {
					out[i].ClaimItems = append(out[i].ClaimItems, item)
				}
			}
		}
	}
	return out, count, nil
}
