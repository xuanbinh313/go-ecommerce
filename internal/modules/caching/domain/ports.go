package domain

import "context"

// Provider is a configured Medusa caching provider. Provider IDs are resolved
// from the module's registry, never inferred from the provider implementation.
type Provider interface {
	Get(context.Context, string, []string) (any, error)
	Set(context.Context, string, []string, any, int, map[string]any) error
	Clear(context.Context, string, []string, map[string]any) error
}

type ProviderRegistry interface {
	RetrieveProvider(string) (Provider, error)
}

// Strategy supplies Medusa's key and entity-tag derivation. The GraphQL schema
// based default strategy is an adapter concern and must be configured by the
// host application.
type Strategy interface {
	ComputeKey(context.Context, any) (string, error)
	ComputeTags(context.Context, any, map[string]any) ([]string, error)
}

type Logger interface {
	Warn(string, ...any)
}
