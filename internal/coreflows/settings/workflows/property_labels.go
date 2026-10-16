package workflows

import (
	"context"
	"encoding/json"
	"github.com/example/medusa-go-clone/internal/modules/settings/domain/models"
	"sync"
)

const (
	CreatePropertyLabelsWorkflowID = "create-property-labels"
	UpdatePropertyLabelsWorkflowID = "update-property-labels"
	DeletePropertyLabelsWorkflowID = "delete-property-labels"
	BatchPropertyLabelsWorkflowID  = "batch-property-labels"
	CreatePropertyLabelsStepID     = "create-property-labels"
	UpdatePropertyLabelsStepID     = "update-property-labels"
	DeletePropertyLabelsStepID     = "delete-property-labels"
)

type PropertyLabelService interface {
	ListPropertyLabels(context.Context, models.PropertyLabelFilter, int, int) ([]models.PropertyLabel, int, error)
	GetPropertyLabel(context.Context, string) (models.PropertyLabel, error)
	CreatePropertyLabels(context.Context, []models.PropertyLabelInput) ([]models.PropertyLabel, error)
	UpdatePropertyLabel(context.Context, string, map[string]any) (models.PropertyLabel, error)
	DeletePropertyLabels(context.Context, []string) ([]models.PropertyLabel, error)
}

type NullableString struct {
	Set   bool
	Value *string
}

func (n *NullableString) UnmarshalJSON(data []byte) error {
	n.Set = true
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	n.Value = &value
	return nil
}

type PropertyLabelUpdate struct {
	ID          string         `json:"id"`
	Label       *string        `json:"label,omitempty"`
	Description NullableString `json:"description,omitempty"`
}
type PropertyLabelBatch struct {
	Create []models.PropertyLabelInput `json:"create,omitempty"`
	Update []PropertyLabelUpdate       `json:"update,omitempty"`
	Delete []string                    `json:"delete,omitempty"`
}
type PropertyLabelBatchResult struct {
	Created []models.PropertyLabel `json:"created"`
	Updated []models.PropertyLabel `json:"updated"`
	Deleted []models.PropertyLabel `json:"deleted"`
}

func CreatePropertyLabels(ctx context.Context, s PropertyLabelService, input []models.PropertyLabelInput) ([]models.PropertyLabel, error) {
	return s.CreatePropertyLabels(ctx, input)
}
func UpdatePropertyLabel(ctx context.Context, s PropertyLabelService, u PropertyLabelUpdate) (models.PropertyLabel, error) {
	fields := map[string]any{}
	if u.Label != nil {
		fields["label"] = *u.Label
	}
	if u.Description.Set {
		fields["description"] = u.Description.Value
	}
	return s.UpdatePropertyLabel(ctx, u.ID, fields)
}
func DeletePropertyLabels(ctx context.Context, s PropertyLabelService, ids []string) error {
	_, err := s.DeletePropertyLabels(ctx, ids)
	return err
}

func BatchPropertyLabels(ctx context.Context, s PropertyLabelService, in PropertyLabelBatch) (PropertyLabelBatchResult, error) {
	type branch struct {
		created, updated, deleted, previous []models.PropertyLabel
		err                                 error
	}
	branches := make([]branch, 3)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); branches[0].created, branches[0].err = s.CreatePropertyLabels(ctx, in.Create) }()
	go func() {
		defer wg.Done()
		for _, u := range in.Update {
			old, err := s.GetPropertyLabel(ctx, u.ID)
			if err != nil {
				branches[1].err = err
				return
			}
			branches[1].previous = append(branches[1].previous, old)
			v, err := UpdatePropertyLabel(ctx, s, u)
			if err != nil {
				branches[1].err = err
				return
			}
			branches[1].updated = append(branches[1].updated, v)
		}
	}()
	go func() { defer wg.Done(); branches[2].deleted, branches[2].err = s.DeletePropertyLabels(ctx, in.Delete) }()
	wg.Wait()
	failed := false
	for _, b := range branches {
		if b.err != nil {
			failed = true
		}
	}
	if failed {
		compensatePropertyLabels(ctx, s, branches[0].created, branches[1].previous)
		if len(branches[2].deleted) > 0 {
			restore := make([]models.PropertyLabelInput, 0, len(branches[2].deleted))
			for _, v := range branches[2].deleted {
				restore = append(restore, models.PropertyLabelInput{Entity: v.Entity, Property: v.Property, Label: v.Label, Description: v.Description})
			}
			_, _ = s.CreatePropertyLabels(ctx, restore)
		}
		for _, b := range branches {
			if b.err != nil {
				return PropertyLabelBatchResult{}, b.err
			}
		}
	}
	return PropertyLabelBatchResult{Created: branches[0].created, Updated: branches[1].updated, Deleted: branches[2].deleted}, nil
}

func compensatePropertyLabels(ctx context.Context, s PropertyLabelService, created, updated []models.PropertyLabel) {
	ids := make([]string, 0, len(created))
	for _, v := range created {
		ids = append(ids, v.ID)
	}
	if len(ids) > 0 {
		_, _ = s.DeletePropertyLabels(ctx, ids)
	}
	for _, v := range updated {
		_, _ = s.UpdatePropertyLabel(ctx, v.ID, map[string]any{"label": v.Label, "description": v.Description})
	}
}
