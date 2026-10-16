package domain

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
)

type Repository interface {
	ListClaims(context.Context, models.ClaimFilter) ([]models.Claim, int, error)
	GetClaim(context.Context, string) (models.Claim, error)
	CreateClaim(context.Context, models.CreateClaimInput) (models.Claim, error)
	CreateOrderChangeForClaim(context.Context, models.CreateClaimInput, string) error
	SoftDeleteClaims(context.Context, []string) error
	CancelBeginClaim(context.Context, string) error
	ListReturnReasons(context.Context, models.ReturnReasonFilter, int, int) ([]models.ReturnReason, int, error)
	GetReturnReason(context.Context, string) (models.ReturnReason, error)
	CreateReturnReasons(context.Context, []models.ReturnReasonInput) ([]models.ReturnReason, error)
	UpdateReturnReasons(context.Context, []string, map[string]any) ([]models.ReturnReason, error)
	SoftDeleteReturnReasons(context.Context, []string) error
	RestoreReturnReasons(context.Context, []string) error
	HardDeleteReturnReasons(context.Context, []string) error
	List(context.Context, models.Filter) (models.ListResult, error)
	Get(context.Context, string, *int) (models.Order, error)
	Archive(context.Context, []string) ([]models.Order, error)
	Complete(context.Context, []string) ([]models.Order, map[string]string, error)
	RestoreStatuses(context.Context, map[string]string) error
	Update(context.Context, models.UpdateInput) (models.Order, error)
	DeleteDraftOrders(context.Context, []string) error
}
