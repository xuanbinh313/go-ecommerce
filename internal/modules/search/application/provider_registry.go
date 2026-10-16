package application

import (
	"fmt"
	"sort"

	"github.com/example/medusa-go-clone/internal/modules/search/domain"
)

type ProviderRegistry struct {
	providers map[string]domain.SearchProvider
}

func NewProviderRegistry(providers ...domain.SearchProvider) (*ProviderRegistry, error) {
	registry := &ProviderRegistry{providers: make(map[string]domain.SearchProvider, len(providers))}
	for _, provider := range providers {
		if provider == nil || provider.Identifier() == "" {
			continue
		}
		id := provider.Identifier()
		if _, exists := registry.providers[id]; exists {
			return nil, fmt.Errorf("the search provider %q is registered twice", id)
		}
		registry.providers[id] = provider
	}
	return registry, nil
}

func (r *ProviderRegistry) Retrieve(id string) (domain.SearchProvider, error) {
	if provider := r.providers[id]; provider != nil {
		return provider, nil
	}
	return nil, fmt.Errorf("search provider %q is not registered; registered providers: %v", id, r.identifiers())
}
func (r *ProviderRegistry) List() []domain.SearchProvider {
	ids := r.identifiers()
	out := make([]domain.SearchProvider, 0, len(ids))
	for _, id := range ids {
		out = append(out, r.providers[id])
	}
	return out
}
func (r *ProviderRegistry) identifiers() []string {
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func (r *ProviderRegistry) GetDefaultIdentifier(configured string) (string, error) {
	if configured != "" {
		if _, err := r.Retrieve(configured); err != nil {
			return "", err
		}
		return configured, nil
	}
	ids := r.identifiers()
	if len(ids) == 1 {
		return ids[0], nil
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("the Search Module requires at least one provider to be registered")
	}
	return "", fmt.Errorf("several search providers are registered (%v), so default_provider must be set or every index definition must name its provider", ids)
}
