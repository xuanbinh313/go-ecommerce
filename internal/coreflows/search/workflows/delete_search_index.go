package workflows

import (
	"context"

	"github.com/example/medusa-go-clone/internal/modules/search/domain/models"
)

const DeleteSearchIndexWorkflowID = "delete-search-index"
const DeleteSearchIndexStepID = "delete-search-index"

type SearchIndexDeleter interface {
	DeleteIndex(context.Context, string) (models.SearchIndexDeleteResult, error)
}

type DeleteSearchIndexInput struct {
	Index string `json:"index"`
}

// DeleteSearchIndex follows Medusa's single delete step. Dropping provider
// indexes and then deleting index/version/sync records has no compensation.
func DeleteSearchIndex(ctx context.Context, service SearchIndexDeleter, input DeleteSearchIndexInput) (models.SearchIndexDeleteResult, error) {
	return service.DeleteIndex(ctx, input.Index)
}
