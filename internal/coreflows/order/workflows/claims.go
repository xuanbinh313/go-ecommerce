package workflows

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
)

const BeginClaimOrderWorkflowID = "begin-claim-order"
const BeginClaimOrderValidationStepID = "begin-claim-order-validation"
const CreateOrderClaimsStepID = "create-order-claims"
const CreateOrderChangeStepID = "create-order-change"
const CancelBeginOrderClaimWorkflowID = "cancel-begin-order-claim"
const CancelBeginOrderClaimValidationStepID = "validate-cancel-begin-order-claim"
const DeleteClaimsStepID = "delete-claims"
const DeleteOrderChangesStepID = "delete-order-change"

type ClaimCreator interface {
	CreateClaim(context.Context, models.CreateClaimInput) (models.Claim, error)
}

func BeginClaimOrder(ctx context.Context, service ClaimCreator, input models.CreateClaimInput) (models.Claim, error) {
	return service.CreateClaim(ctx, input)
}

type BeginClaimCanceler interface {
	CancelBeginClaim(context.Context, string) error
}

func CancelBeginOrderClaim(ctx context.Context, service BeginClaimCanceler, id string) error {
	return service.CancelBeginClaim(ctx, id)
}
