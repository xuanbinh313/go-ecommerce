package stripe

import (
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/payment/domain"
)

type Registry struct{ providers map[string]domain.Provider }

func NewRegistry(options Options) (*Registry, error) {
	variants := map[string]string{
		"pp_stripe_stripe":                       "stripe",
		"pp_stripe-oxxo_stripe-oxxo":             "stripe-oxxo",
		"pp_stripe-bancontact_stripe-bancontact": "stripe-bancontact",
		"pp_stripe-blik_stripe-blik":             "stripe-blik",
		"pp_stripe-giropay_stripe-giropay":       "stripe-giropay",
		"pp_stripe-ideal_stripe-ideal":           "stripe-ideal",
		"pp_stripe-przelewy24_stripe-przelewy24": "stripe-przelewy24",
		"pp_stripe-promptpay_stripe-promptpay":   "stripe-promptpay",
	}
	registry := &Registry{providers: make(map[string]domain.Provider, len(variants))}
	for id, variant := range variants {
		provider, err := New(options, variant)
		if err != nil {
			return nil, err
		}
		registry.providers[id] = provider
	}
	return registry, nil
}

func (r *Registry) RetrieveProvider(id string) (domain.Provider, error) {
	provider, ok := r.providers[id]
	if !ok {
		return nil, fmt.Errorf("payment provider %q is not registered", id)
	}
	return provider, nil
}

var _ domain.ProviderRegistry = (*Registry)(nil)
