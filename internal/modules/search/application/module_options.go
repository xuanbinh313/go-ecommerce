package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/search/domain"
	"github.com/example/medusa-go-clone/internal/modules/search/domain/models"
)

// ModuleOptions mirrors Medusa Search's providers/default_provider/
// index_prefix/indexes module options. Indexes stay configuration-owned; this
// package does not discover or mutate definitions at runtime.
type ModuleOptions struct {
	Providers       []SearchProviderOptions `json:"providers,omitempty"`
	DefaultProvider string                  `json:"default_provider,omitempty"`
	IndexPrefix     string                  `json:"index_prefix,omitempty"`
	Indexes         []json.RawMessage       `json:"indexes,omitempty"`
}

type SearchProviderOptions struct {
	Resolve string         `json:"resolve"`
	ID      string         `json:"id"`
	Options map[string]any `json:"options,omitempty"`
}

// ResolveIndexDefinitions applies Medusa's source defaults and validation for
// the fields used by the Go Search module. Events, consume, and seed callbacks
// are application functions in Medusa and must be wired by Go callers rather
// than serialized as JSON.
func ResolveIndexDefinitions(options ModuleOptions, providers domain.ProviderRegistry) ([]models.IndexDefinition, error) {
	if len(options.Indexes) == 0 {
		return []models.IndexDefinition{}, nil
	}
	defaultProvider, err := providers.GetDefaultIdentifier(options.DefaultProvider)
	if err != nil {
		return nil, err
	}
	definitions := make([]models.IndexDefinition, 0, len(options.Indexes))
	seen := map[string]bool{}
	for _, raw := range options.Indexes {
		var source map[string]json.RawMessage
		if err := json.Unmarshal(raw, &source); err != nil {
			return nil, fmt.Errorf("decode Search index definition: %w", err)
		}
		var definition models.IndexDefinition
		if err := json.Unmarshal(raw, &definition); err != nil {
			return nil, fmt.Errorf("decode Search index definition: %w", err)
		}
		if definition.Name == "" {
			return nil, fmt.Errorf("a Search index definition requires a name")
		}
		if seen[definition.Name] {
			return nil, fmt.Errorf("duplicate Search index definition for %q", definition.Name)
		}
		seen[definition.Name] = true
		if definition.PrimaryKey == "" {
			definition.PrimaryKey = "id"
		}
		if _, ok := definition.Fields[definition.PrimaryKey]; !ok {
			return nil, fmt.Errorf("Search index %q must declare its primary key field %q", definition.Name, definition.PrimaryKey)
		}
		if definition.Provider == "" {
			definition.Provider = defaultProvider
		}
		if _, err := providers.Retrieve(definition.Provider); err != nil {
			return nil, err
		}
		definition.PhysicalName = strings.Join(nonempty(options.IndexPrefix, definition.Name), "_")
		if definition.Settings == nil {
			definition.Settings = map[string]any{}
		}
		hash, err := sourceDefinitionHash(source["fields"], source["settings"], definition.PrimaryKey, definition.PhysicalName)
		if err != nil {
			return nil, fmt.Errorf("hash Search index %q definition: %w", definition.Name, err)
		}
		definition.DefinitionHash = hash
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

func sourceDefinitionHash(fieldsRaw, settingsRaw json.RawMessage, primaryKey, physicalName string) (string, error) {
	if len(fieldsRaw) == 0 || string(fieldsRaw) == "null" {
		return "", fmt.Errorf("fields are required")
	}
	fields, err := canonicalJSON(fieldsRaw)
	if err != nil {
		return "", err
	}
	settings := []byte(`{}`)
	if len(settingsRaw) > 0 && string(settingsRaw) != "null" {
		settings, err = canonicalJSON(settingsRaw)
		if err != nil {
			return "", err
		}
	}
	// Map keys are sorted by encoding/json, matching Medusa's stableStringify.
	payload := map[string]json.RawMessage{
		"fields":        fields,
		"settings":      settings,
		"primary_key":   mustJSON(primaryKey),
		"physical_name": mustJSON(physicalName),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])[:32], nil
}

func canonicalJSON(raw []byte) (json.RawMessage, error) {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(value)
	return encoded, err
}

func mustJSON(value any) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}

func nonempty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
