package workflows

import (
	"context"
	"encoding/json"
)

const ReindexSearchIndexesWorkflowID = "reindex-search-indexes"
const ValidateSearchIndexesExistStepID = "validate-search-indexes-exist"
const ReindexSearchIndexesStepID = "reindex-search-indexes"

type ReindexSearchIndexesInput struct {
	Index    any            `json:"index,omitempty"`
	Strategy string         `json:"strategy,omitempty"`
	Filters  map[string]any `json:"filters,omitempty"`
	Since    string         `json:"since,omitempty"`
}
type ReindexSearchIndexesResult struct {
	JobID   string   `json:"job_id"`
	Indexes []string `json:"indexes"`
}

// ReindexSearchIndexes is intentionally a direct call. The background job
// adapter must preserve Medusa's async reindex semantics before this is wired.
type SearchIndexReindexer interface {
	Reindex(context.Context, ReindexSearchIndexesInput) (ReindexSearchIndexesResult, error)
}

func ReindexSearchIndexes(ctx context.Context, service SearchIndexReindexer, input ReindexSearchIndexesInput) (ReindexSearchIndexesResult, error) {
	if input.Filters != nil {
		if _, err := json.Marshal(input.Filters); err != nil {
			return ReindexSearchIndexesResult{}, err
		}
	}
	return service.Reindex(ctx, input)
}
