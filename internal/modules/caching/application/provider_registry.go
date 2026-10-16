package application

import (
	"fmt"
	"sort"

	"github.com/example/medusa-go-clone/internal/modules/caching/domain"
)

type ProviderRegistry struct {
	providers map[string]domain.Provider
}

func NewProviderRegistry(providers map[string]domain.Provider) (*ProviderRegistry, error) {
	registry := &ProviderRegistry{providers: make(map[string]domain.Provider, len(providers))}
	for id, provider := range providers {
		if id == "" || provider == nil {
			return nil, fmt.Errorf("caching provider id and implementation are required")
		}
		registry.providers[id] = provider
	}
	return registry, nil
}

func (r *ProviderRegistry) Register(id string, provider domain.Provider) error {
	if id == "" || provider == nil {
		return fmt.Errorf("caching provider id and implementation are required")
	}
	if _, exists := r.providers[id]; exists {
		return fmt.Errorf("the caching provider %q is registered twice", id)
	}
	r.providers[id] = provider
	return nil
}

func (r *ProviderRegistry) RetrieveProvider(id string) (domain.Provider, error) {
	if provider := r.providers[id]; provider != nil {
		return provider, nil
	}
	return nil, fmt.Errorf("caching provider %q is not registered; registered providers: %v", id, r.identifiers())
}

func (r *ProviderRegistry) identifiers() []string {
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

var _ domain.ProviderRegistry = (*ProviderRegistry)(nil)
