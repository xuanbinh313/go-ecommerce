package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/notification/domain"
	"github.com/example/medusa-go-clone/internal/modules/notification/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, filter models.NotificationFilter, limit, offset int, order string) ([]models.Notification, int, error) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	addArrayFilter := func(column string, values []string) {
		if len(values) == 0 {
			return
		}
		args = append(args, values)
		where = append(where, fmt.Sprintf("%s=ANY($%d)", column, len(args)))
	}
	addArrayFilter("id", filter.IDs)
	addArrayFilter(`"to"`, filter.To)
	addArrayFilter("channel", filter.Channel)
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		where = append(where, fmt.Sprintf(`("to" ILIKE $%d OR channel ILIKE $%d OR COALESCE(template,'') ILIKE $%d)`, len(args), len(args), len(args)))
	}
	whereSQL := " WHERE " + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM notification`+whereSQL, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	direction := "DESC"
	if order == "created_at" {
		direction = "ASC"
	}
	args = append(args, limit, offset)
	query := fmt.Sprintf(`SELECT id,"to","from",channel,template,data,provider_data,trigger_type,resource_id,resource_type,receiver_id,original_notification_id,idempotency_key,external_id,status,provider_id,created_at,updated_at FROM notification%s ORDER BY created_at %s LIMIT $%d OFFSET $%d`, whereSQL, direction, len(args)-1, len(args))
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.Notification{}
	for rows.Next() {
		var item models.Notification
		if err := rows.Scan(&item.ID, &item.To, &item.From, &item.Channel, &item.Template, &item.Data, &item.ProviderData, &item.TriggerType, &item.ResourceID, &item.ResourceType, &item.ReceiverID, &item.OriginalNotificationID, &item.IdempotencyKey, &item.ExternalID, &item.Status, &item.ProviderID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, count, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id string) (models.Notification, error) {
	var item models.Notification
	err := r.db.QueryRow(ctx, `SELECT id,"to","from",channel,template,data,provider_data,trigger_type,resource_id,resource_type,receiver_id,original_notification_id,idempotency_key,external_id,status,provider_id,created_at,updated_at FROM notification WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&item.ID, &item.To, &item.From, &item.Channel, &item.Template, &item.Data, &item.ProviderData, &item.TriggerType, &item.ResourceID, &item.ResourceType, &item.ReceiverID, &item.OriginalNotificationID, &item.IdempotencyKey, &item.ExternalID, &item.Status, &item.ProviderID, &item.CreatedAt, &item.UpdatedAt)
	if err == pgx.ErrNoRows {
		return models.Notification{}, pgx.ErrNoRows
	}
	return item, err
}

var _ domain.Repository = (*Repository)(nil)
