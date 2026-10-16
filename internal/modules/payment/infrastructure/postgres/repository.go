package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/payment/domain"
	"github.com/example/medusa-go-clone/internal/modules/payment/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func jsonValue(value any) any {
	if value == nil {
		return json.RawMessage("null")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("null")
	}
	return raw
}

func scanCollection(row pgx.Row) (*models.PaymentCollection, error) {
	var v models.PaymentCollection
	var amount string
	var authorized, captured, refunded *string
	var metadata json.RawMessage
	err := row.Scan(&v.ID, &v.CurrencyCode, &amount, &v.RawAmount, &authorized, &v.RawAuthorizedAmount, &captured, &v.RawCapturedAmount, &refunded, &v.RawRefundedAmount, &v.CompletedAt, &v.Status, &metadata, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	if err != nil {
		return nil, err
	}
	v.Amount = models.Amount(amount)
	if authorized != nil {
		x := models.Amount(*authorized)
		v.AuthorizedAmount = &x
	}
	if captured != nil {
		x := models.Amount(*captured)
		v.CapturedAmount = &x
	}
	if refunded != nil {
		x := models.Amount(*refunded)
		v.RefundedAmount = &x
	}
	if len(metadata) > 0 && string(metadata) != "null" {
		_ = json.Unmarshal(metadata, &v.Metadata)
	}
	return &v, nil
}

const collectionColumns = `id,currency_code,amount::text,raw_amount,authorized_amount::text,raw_authorized_amount,captured_amount::text,raw_captured_amount,refunded_amount::text,raw_refunded_amount,completed_at,status,metadata,created_at,updated_at,deleted_at`

func (r *Repository) CreatePaymentCollection(ctx context.Context, in models.CreatePaymentCollectionInput) (*models.PaymentCollection, error) {
	if len(in.RawAmount) == 0 {
		return nil, fmt.Errorf("raw_amount is required to preserve Medusa BigNumber precision")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	id := entityid.New("pay_col")
	row := tx.QueryRow(ctx, `INSERT INTO payment_collection(id,currency_code,amount,raw_amount,metadata) VALUES($1,$2,$3,$4,$5) RETURNING `+collectionColumns, id, in.CurrencyCode, string(in.Amount), in.RawAmount, jsonValue(in.Metadata))
	created, err := scanCollection(row)
	if err != nil {
		return nil, err
	}
	for _, providerID := range in.PaymentProviderIDs {
		if _, err = tx.Exec(ctx, `INSERT INTO payment_collection_payment_providers(payment_collection_id,payment_provider_id) VALUES($1,$2)`, id, providerID); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	created.PaymentProviders = make([]models.PaymentProvider, 0, len(in.PaymentProviderIDs))
	for _, providerID := range in.PaymentProviderIDs {
		created.PaymentProviders = append(created.PaymentProviders, models.PaymentProvider{ID: providerID, IsEnabled: true})
	}
	return created, nil
}

func (r *Repository) GetPaymentCollection(ctx context.Context, id string) (*models.PaymentCollection, error) {
	return scanCollection(r.db.QueryRow(ctx, `SELECT `+collectionColumns+` FROM payment_collection WHERE id=$1 AND deleted_at IS NULL`, id))
}

func (r *Repository) DeletePaymentCollections(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE payment_session SET deleted_at=now(),updated_at=now() WHERE payment_collection_id=ANY($1) AND deleted_at IS NULL`, ids); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE payment SET deleted_at=now(),updated_at=now() WHERE payment_collection_id=ANY($1) AND deleted_at IS NULL`, ids); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE payment_collection SET deleted_at=now(),updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL`, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) ListPaymentCollections(ctx context.Context, filters map[string]any, limit, offset int) ([]models.PaymentCollection, int64, error) {
	conditions := []string{"deleted_at IS NULL"}
	args := []any{}
	for _, key := range []string{"id", "currency_code", "status"} {
		value, ok := filters[key]
		if !ok {
			continue
		}
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf("%s=$%d", key, len(args)))
	}
	where := strings.Join(conditions, " AND ")
	var count int64
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM payment_collection WHERE `+where, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, `SELECT `+collectionColumns+` FROM payment_collection WHERE `+where+fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := []models.PaymentCollection{}
	for rows.Next() {
		v, e := scanCollection(rows)
		if e != nil {
			return nil, 0, e
		}
		result = append(result, *v)
	}
	return result, count, rows.Err()
}

func (r *Repository) GetPaymentSession(ctx context.Context, id string) (*models.PaymentSession, error) {
	var v models.PaymentSession
	var amount string
	var data, contextJSON, metadata json.RawMessage
	err := r.db.QueryRow(ctx, `SELECT id,currency_code,amount::text,raw_amount,provider_id,data,context,status,authorized_at,payment_collection_id,metadata,created_at,updated_at,deleted_at FROM payment_session WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&v.ID, &v.CurrencyCode, &amount, &v.RawAmount, &v.ProviderID, &data, &contextJSON, &v.Status, &v.AuthorizedAt, &v.PaymentCollectionID, &metadata, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	if err != nil {
		return nil, err
	}
	v.Amount = models.Amount(amount)
	_ = json.Unmarshal(data, &v.Data)
	if len(contextJSON) > 0 && string(contextJSON) != "null" {
		_ = json.Unmarshal(contextJSON, &v.Context)
	}
	if len(metadata) > 0 && string(metadata) != "null" {
		_ = json.Unmarshal(metadata, &v.Metadata)
	}
	return &v, nil
}

func (r *Repository) GetPayment(ctx context.Context, id string) (*models.Payment, error) {
	var v models.Payment
	var amount string
	var data, metadata json.RawMessage
	err := r.db.QueryRow(ctx, `SELECT id,amount::text,raw_amount,currency_code,provider_id,data,metadata,captured_at,canceled_at,payment_collection_id,payment_session_id,created_at,updated_at,deleted_at FROM payment WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&v.ID, &amount, &v.RawAmount, &v.CurrencyCode, &v.ProviderID, &data, &metadata, &v.CapturedAt, &v.CanceledAt, &v.PaymentCollectionID, &v.PaymentSessionID, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	if err != nil {
		return nil, err
	}
	v.Amount = models.Amount(amount)
	if len(data) > 0 && string(data) != "null" {
		_ = json.Unmarshal(data, &v.Data)
	}
	if len(metadata) > 0 && string(metadata) != "null" {
		_ = json.Unmarshal(metadata, &v.Metadata)
	}
	if err = r.loadPaymentRelations(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

const paymentColumns = `id,amount::text,raw_amount,currency_code,provider_id,data,metadata,captured_at,canceled_at,payment_collection_id,payment_session_id,created_at,updated_at,deleted_at`

func (r *Repository) ListPayments(ctx context.Context, f models.PaymentFilter) ([]models.Payment, int64, error) {
	conditions := []string{}
	args := []any{}
	if len(f.DeletedAt) == 0 {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if len(f.IDs) > 0 {
		args = append(args, f.IDs)
		conditions = append(conditions, fmt.Sprintf("id=ANY($%d)", len(args)))
	}
	if len(f.PaymentSessionIDs) > 0 {
		args = append(args, f.PaymentSessionIDs)
		conditions = append(conditions, fmt.Sprintf("payment_session_id=ANY($%d)", len(args)))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		conditions = append(conditions, fmt.Sprintf("(id ILIKE $%d OR payment_session_id ILIKE $%d)", len(args), len(args)))
	}
	for _, group := range []struct {
		name string
		ops  map[string]string
	}{{"created_at", f.CreatedAt}, {"updated_at", f.UpdatedAt}, {"deleted_at", f.DeletedAt}} {
		if err := appendDateConditions(&conditions, &args, group.name, group.ops); err != nil {
			return nil, 0, err
		}
	}
	where := strings.Join(conditions, " AND ")
	var count int64
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM payment WHERE `+where, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	limit := f.Limit
	if limit < 1 {
		limit = 20
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, `SELECT `+paymentColumns+` FROM payment WHERE `+where+fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	payments := []models.Payment{}
	for rows.Next() {
		var p models.Payment
		var amount string
		var data, metadata json.RawMessage
		if err = rows.Scan(&p.ID, &amount, &p.RawAmount, &p.CurrencyCode, &p.ProviderID, &data, &metadata, &p.CapturedAt, &p.CanceledAt, &p.PaymentCollectionID, &p.PaymentSessionID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt); err != nil {
			return nil, 0, err
		}
		p.Amount = models.Amount(amount)
		if len(data) > 0 && string(data) != "null" {
			_ = json.Unmarshal(data, &p.Data)
		}
		if len(metadata) > 0 && string(metadata) != "null" {
			_ = json.Unmarshal(metadata, &p.Metadata)
		}
		payments = append(payments, p)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	for i := range payments {
		if err = r.loadPaymentRelations(ctx, &payments[i]); err != nil {
			return nil, 0, err
		}
	}
	return payments, count, nil
}

func appendDateConditions(conditions *[]string, args *[]any, column string, ops map[string]string) error {
	if len(ops) == 0 {
		return nil
	}
	operators := map[string]string{"$eq": "=", "$ne": "<>", "$gt": ">", "$gte": ">=", "$lt": "<", "$lte": "<="}
	for key, value := range ops {
		if column == "deleted_at" && (value == "null" || value == "") {
			if key == "$eq" {
				*conditions = append(*conditions, "deleted_at IS NULL")
				continue
			}
			if key == "$ne" {
				*conditions = append(*conditions, "deleted_at IS NOT NULL")
				continue
			}
		}
		op, ok := operators[key]
		if !ok {
			return fmt.Errorf("unsupported %s operator %q", column, key)
		}
		*args = append(*args, value)
		*conditions = append(*conditions, fmt.Sprintf("%s %s $%d", column, op, len(*args)))
	}
	return nil
}

func (r *Repository) loadPaymentRelations(ctx context.Context, p *models.Payment) error {
	p.Captures = []models.Capture{}
	captureRows, err := r.db.Query(ctx, `SELECT id,amount::text,raw_amount,payment_id,metadata,created_by,created_at,updated_at,deleted_at FROM capture WHERE payment_id=$1 AND deleted_at IS NULL ORDER BY created_at`, p.ID)
	if err != nil {
		return err
	}
	for captureRows.Next() {
		var v models.Capture
		var amount string
		var meta json.RawMessage
		if err = captureRows.Scan(&v.ID, &amount, &v.RawAmount, &v.PaymentID, &meta, &v.CreatedBy, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt); err != nil {
			captureRows.Close()
			return err
		}
		v.Amount = models.Amount(amount)
		if len(meta) > 0 && string(meta) != "null" {
			_ = json.Unmarshal(meta, &v.Metadata)
		}
		p.Captures = append(p.Captures, v)
	}
	err = captureRows.Err()
	captureRows.Close()
	if err != nil {
		return err
	}
	p.Refunds = []models.Refund{}
	refundRows, err := r.db.Query(ctx, `SELECT r.id,r.amount::text,r.raw_amount,r.status,r.idempotency_key,r.payment_id,r.refund_reason_id,r.note,r.created_by,r.metadata,r.created_at,r.updated_at,r.deleted_at,rr.id,rr.label,rr.code,rr.description,rr.metadata,rr.created_at,rr.updated_at,rr.deleted_at FROM refund r LEFT JOIN refund_reason rr ON rr.id=r.refund_reason_id AND rr.deleted_at IS NULL WHERE r.payment_id=$1 AND r.deleted_at IS NULL ORDER BY r.created_at`, p.ID)
	if err != nil {
		return err
	}
	for refundRows.Next() {
		var v models.Refund
		var amount string
		var meta json.RawMessage
		var reasonID, reasonLabel, reasonCode *string
		var reasonDesc *string
		var reasonMeta json.RawMessage
		var reasonCreated, reasonUpdated *time.Time
		var reasonDeleted *time.Time
		if err = refundRows.Scan(&v.ID, &amount, &v.RawAmount, &v.Status, &v.IdempotencyKey, &v.PaymentID, &v.RefundReasonID, &v.Note, &v.CreatedBy, &meta, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt, &reasonID, &reasonLabel, &reasonCode, &reasonDesc, &reasonMeta, &reasonCreated, &reasonUpdated, &reasonDeleted); err != nil {
			refundRows.Close()
			return err
		}
		v.Amount = models.Amount(amount)
		if len(meta) > 0 && string(meta) != "null" {
			_ = json.Unmarshal(meta, &v.Metadata)
		}
		if reasonID != nil && reasonLabel != nil && reasonCode != nil {
			reason := models.RefundReason{ID: *reasonID, Label: *reasonLabel, Code: *reasonCode, Description: reasonDesc, CreatedAt: *reasonCreated, UpdatedAt: *reasonUpdated, DeletedAt: reasonDeleted}
			if len(reasonMeta) > 0 && string(reasonMeta) != "null" {
				_ = json.Unmarshal(reasonMeta, &reason.Metadata)
			}
			v.RefundReason = &reason
		}
		p.Refunds = append(p.Refunds, v)
	}
	err = refundRows.Err()
	refundRows.Close()
	return err
}

func (r *Repository) ListPaymentProviders(ctx context.Context, f models.PaymentProviderFilter) ([]models.PaymentProvider, int64, error) {
	conditions := []string{"deleted_at IS NULL"}
	args := []any{}
	if len(f.IDs) > 0 {
		args = append(args, f.IDs)
		conditions = append(conditions, fmt.Sprintf("id=ANY($%d)", len(args)))
	}
	if f.IsEnabled != nil {
		args = append(args, *f.IsEnabled)
		conditions = append(conditions, fmt.Sprintf("is_enabled=$%d", len(args)))
	}
	where := strings.Join(conditions, " AND ")
	var count int64
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM payment_provider WHERE `+where, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	limit := f.Limit
	if limit < 1 {
		limit = 20
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, `SELECT id,is_enabled FROM payment_provider WHERE `+where+fmt.Sprintf(" ORDER BY id LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := []models.PaymentProvider{}
	for rows.Next() {
		var v models.PaymentProvider
		if err = rows.Scan(&v.ID, &v.IsEnabled); err != nil {
			return nil, 0, err
		}
		result = append(result, v)
	}
	return result, count, rows.Err()
}

func (r *Repository) CreateCapture(ctx context.Context, v models.Capture) (*models.Capture, error) {
	if len(v.RawAmount) == 0 {
		return nil, fmt.Errorf("raw_amount is required")
	}
	if v.ID == "" {
		v.ID = entityid.New("capt")
	}
	err := r.db.QueryRow(ctx, `INSERT INTO capture(id,amount,raw_amount,payment_id,created_by,metadata) VALUES($1,$2,$3,$4,$5,$6) RETURNING created_at,updated_at,deleted_at`, v.ID, string(v.Amount), v.RawAmount, v.PaymentID, v.CreatedBy, jsonValue(v.Metadata)).Scan(&v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *Repository) CreateRefund(ctx context.Context, v models.Refund) (*models.Refund, error) {
	if len(v.RawAmount) == 0 {
		return nil, fmt.Errorf("raw_amount is required")
	}
	if v.ID == "" {
		v.ID = entityid.New("ref")
	}
	if v.Status == "" {
		v.Status = models.RefundPending
	}
	err := r.db.QueryRow(ctx, `INSERT INTO refund(id,amount,raw_amount,status,idempotency_key,payment_id,refund_reason_id,note,created_by,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING created_at,updated_at,deleted_at`, v.ID, string(v.Amount), v.RawAmount, v.Status, v.IdempotencyKey, v.PaymentID, v.RefundReasonID, v.Note, v.CreatedBy, jsonValue(v.Metadata)).Scan(&v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// SortFilterKeys is used by the API adapter when it needs deterministic filter handling.
func SortFilterKeys(filters map[string]any) []string {
	keys := make([]string, 0, len(filters))
	for key := range filters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

var _ domain.Repository = (*Repository)(nil)
