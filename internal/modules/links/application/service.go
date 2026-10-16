package application

import (
	"context"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/links/domain"
	"github.com/example/medusa-go-clone/internal/modules/links/domain/models"
)

type Service struct {
	repository  domain.Repository
	definitions map[string]models.Definition
}

func NewService(repository domain.Repository) *Service {
	definitions := map[string]models.Definition{}
	for _, definition := range models.Definitions {
		definitions[definition.Name] = definition
	}
	return &Service{repository: repository, definitions: definitions}
}
func (s *Service) definition(name string) (models.Definition, error) {
	d, ok := s.definitions[name]
	if !ok {
		return d, fmt.Errorf("unknown link definition %q", name)
	}
	return d, nil
}
func (s *Service) Create(ctx context.Context, name string, links []map[string]any) ([]models.Link, error) {
	d, err := s.definition(name)
	if err != nil {
		return nil, err
	}
	for _, fields := range links {
		for _, key := range d.ForeignKeys {
			if fields[key] == nil || fields[key] == "" {
				return nil, fmt.Errorf("link %s requires %s", name, key)
			}
		}
	}
	return s.repository.Create(ctx, d, links)
}

func (s *Service) List(ctx context.Context, name string, filters map[string]any) ([]models.Link, error) {
	d, err := s.definition(name)
	if err != nil {
		return nil, err
	}
	return s.repository.List(ctx, d, filters)
}
func (s *Service) Update(ctx context.Context, name string, filters, data map[string]any) ([]models.Link, error) {
	d, err := s.definition(name)
	if err != nil {
		return nil, err
	}
	return s.repository.Update(ctx, d, filters, data)
}
func (s *Service) Dismiss(ctx context.Context, name string, filters []map[string]any) ([]models.Link, error) {
	d, err := s.definition(name)
	if err != nil {
		return nil, err
	}
	return s.repository.Dismiss(ctx, d, filters)
}
func (s *Service) Delete(ctx context.Context, name string, filters []map[string]any) error {
	d, err := s.definition(name)
	if err != nil {
		return err
	}
	return s.repository.Delete(ctx, d, filters)
}
func (s *Service) Restore(ctx context.Context, name string, filters []map[string]any) ([]models.Link, error) {
	d, err := s.definition(name)
	if err != nil {
		return nil, err
	}
	return s.repository.Restore(ctx, d, filters)
}
