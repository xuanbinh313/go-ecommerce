package workflows

import (
	"context"
	"encoding/json"

	customerdomain "github.com/example/medusa-go-clone/internal/modules/customer/domain"
	customermodels "github.com/example/medusa-go-clone/internal/modules/customer/domain/models"
)

const FindOrCreateCustomerStepID = "find-or-create-customer"

func FindOrCreateCustomerByEmail(ctx context.Context, customers customerdomain.OrderCustomerResolver, email string) (customermodels.Customer, bool, error) {
	rows, err := customers.FindCustomersByEmail(ctx, email)
	if err != nil {
		return customermodels.Customer{}, false, err
	}
	if len(rows) > 0 {
		return rows[0], false, nil
	}
	created, err := customers.Create(ctx, customermodels.Input{Email: &email, HasAccount: false, Metadata: json.RawMessage("null")})
	if err != nil {
		return customermodels.Customer{}, false, err
	}
	return created, true, nil
}
