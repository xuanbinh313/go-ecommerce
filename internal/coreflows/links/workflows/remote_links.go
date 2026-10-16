package workflows

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/links/domain"
	"github.com/example/medusa-go-clone/internal/modules/links/domain/models"
)

const (
	CreateLinksWorkflowID  = "create-link"
	CreateLinksStepID      = "create-remote-links"
	UpdateLinksWorkflowID  = "update-link"
	UpdateLinksStepID      = "update-remote-links-step"
	DismissLinksWorkflowID = "dismiss-link"
	DismissLinksStepID     = "dismiss-remote-links"
	BatchLinksWorkflowID   = "batch-links"
)

type Definition struct {
	Name    string
	Filters map[string]any
	Fields  map[string]any
	Data    map[string]any
}

func definition(name string) (models.Definition, error) {
	for _, d := range models.Definitions {
		if d.Name == name {
			return d, nil
		}
	}
	return models.Definition{}, fmt.Errorf("unknown link definition %q", name)
}

func CreateLinks(ctx context.Context, repo domain.Repository, inputs []Definition) ([]models.Link, error) {
	out := []models.Link{}
	for _, input := range inputs {
		d, err := definition(input.Name)
		if err != nil {
			return nil, errors.Join(err, rollbackLinkCreates(ctx, repo, out))
		}
		created, err := repo.Create(ctx, d, []map[string]any{input.Fields})
		if err != nil {
			return nil, errors.Join(err, rollbackLinkCreates(ctx, repo, out))
		}
		out = append(out, created...)
	}
	return out, nil
}

func UpdateLinks(ctx context.Context, repo domain.Repository, inputs []Definition) ([]models.Link, error) {
	out := []models.Link{}
	updatedBefore := []models.Link{}
	for _, input := range inputs {
		d, err := definition(input.Name)
		if err != nil {
			return nil, errors.Join(err, restoreLinkUpdates(ctx, repo, updatedBefore))
		}
		before, err := repo.List(ctx, d, input.Filters)
		if err != nil {
			return nil, errors.Join(err, restoreLinkUpdates(ctx, repo, updatedBefore))
		}
		if len(before) != 1 {
			return nil, errors.Join(fmt.Errorf("could not find link to update"), restoreLinkUpdates(ctx, repo, updatedBefore))
		}
		updated, err := repo.Update(ctx, d, input.Filters, input.Data)
		if err != nil {
			return nil, errors.Join(err, restoreLinkUpdates(ctx, repo, updatedBefore))
		}
		out = append(out, updated...)
		updatedBefore = append(updatedBefore, before...)
	}
	return out, nil
}

func DismissLinks(ctx context.Context, repo domain.Repository, inputs []Definition) ([]models.Link, error) {
	out := []models.Link{}
	for _, input := range inputs {
		d, err := definition(input.Name)
		if err != nil {
			return nil, errors.Join(err, restoreDismissedLinks(ctx, repo, out))
		}
		removed, err := repo.Dismiss(ctx, d, []map[string]any{input.Filters})
		if err != nil {
			return nil, errors.Join(err, restoreDismissedLinks(ctx, repo, out))
		}
		out = append(out, removed...)
	}
	return out, nil
}

type BatchInput struct{ Create, Update, Delete []Definition }
type BatchOutput struct {
	Created []models.Link `json:"created"`
	Updated []models.Link `json:"updated"`
	Deleted []models.Link `json:"deleted"`
}

func BatchLinks(ctx context.Context, repo domain.Repository, input BatchInput) (BatchOutput, error) {
	out := BatchOutput{}
	created, err := CreateLinks(ctx, repo, input.Create)
	if err != nil {
		return out, err
	}
	out.Created = created
	beforeUpdates, err := snapshotLinks(ctx, repo, input.Update)
	if err != nil {
		return out, errors.Join(err, rollbackLinkCreates(ctx, repo, created))
	}
	updated, err := UpdateLinks(ctx, repo, input.Update)
	if err != nil {
		return out, errors.Join(err, rollbackLinkCreates(ctx, repo, created))
	}
	out.Updated = updated
	deleted, err := DismissLinks(ctx, repo, input.Delete)
	if err != nil {
		return out, errors.Join(err, restoreLinkUpdates(ctx, repo, beforeUpdates), rollbackLinkCreates(ctx, repo, created))
	}
	out.Deleted = deleted
	return out, nil
}

func rollbackLinkCreates(ctx context.Context, repo domain.Repository, links []models.Link) error {
	var joined error
	for i := len(links) - 1; i >= 0; i-- {
		d, err := definition(links[i].Definition)
		if err != nil {
			joined = errors.Join(joined, err)
			continue
		}
		filters := map[string]any{"id": links[i].ID}
		_, err = repo.Dismiss(ctx, d, []map[string]any{filters})
		joined = errors.Join(joined, err)
	}
	return joined
}

func snapshotLinks(ctx context.Context, repo domain.Repository, inputs []Definition) ([]models.Link, error) {
	var rows []models.Link
	for _, input := range inputs {
		d, err := definition(input.Name)
		if err != nil {
			return nil, err
		}
		found, err := repo.List(ctx, d, input.Filters)
		if err != nil {
			return nil, err
		}
		if len(found) != 1 {
			return nil, fmt.Errorf("could not find link to update")
		}
		rows = append(rows, found...)
	}
	return rows, nil
}

func restoreLinkUpdates(ctx context.Context, links domain.Repository, before []models.Link) error {
	var joined error
	for i := len(before) - 1; i >= 0; i-- {
		d, err := definition(before[i].Definition)
		if err != nil {
			joined = errors.Join(joined, err)
			continue
		}
		data := map[string]any{}
		for _, field := range d.ExtraFields {
			if value, ok := before[i].Fields[field]; ok {
				data[field] = value
			}
		}
		_, err = links.Update(ctx, d, map[string]any{"id": before[i].ID}, data)
		joined = errors.Join(joined, err)
	}
	return joined
}

func restoreDismissedLinks(ctx context.Context, repo domain.Repository, before []models.Link) error {
	var joined error
	for _, link := range before {
		d, err := definition(link.Definition)
		if err != nil {
			joined = errors.Join(joined, err)
			continue
		}
		fields := map[string]any{}
		for _, key := range append(append([]string{}, d.ForeignKeys...), d.ExtraFields...) {
			if value, ok := link.Fields[key]; ok {
				fields[key] = value
			}
		}
		_, err = repo.Create(ctx, d, []map[string]any{fields})
		joined = errors.Join(joined, err)
	}
	return joined
}
