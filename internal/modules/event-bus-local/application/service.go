package application

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/event-bus-local/domain"
	"github.com/example/medusa-go-clone/internal/modules/event-bus-local/domain/models"
)

type subscriber struct {
	id string
	fn domain.Subscriber
}
type Service struct {
	mu          sync.RWMutex
	subscribers map[string][]subscriber
	grouped     map[string][]models.Event
	logger      *slog.Logger
}

func NewService(logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{subscribers: map[string][]subscriber{}, grouped: map[string][]models.Event{}, logger: logger}
}
func (s *Service) Subscribe(event, id string, fn domain.Subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.subscribers[event]
	for _, v := range list {
		if id != "" && v.id == id {
			return
		}
	}
	s.subscribers[event] = append(list, subscriber{id: id, fn: fn})
}
func (s *Service) Unsubscribe(event, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.subscribers[event]
	next := list[:0]
	for _, v := range list {
		if v.id != id {
			next = append(next, v)
		}
	}
	if len(next) == 0 {
		delete(s.subscribers, event)
	} else {
		s.subscribers[event] = next
	}
}
func (s *Service) Emit(ctx context.Context, events []models.Event, options models.Options) error {
	for i := range events {
		e := events[i]
		if e.Metadata == nil {
			e.Metadata = map[string]any{}
		}
		if _, exists := e.Metadata["created_at"]; !exists {
			e.Metadata["created_at"] = time.Now().UTC().Format(time.RFC3339Nano)
		}
		if group, ok := e.Metadata["eventGroupId"].(string); ok && group != "" {
			s.mu.Lock()
			s.grouped[group] = append(s.grouped[group], e)
			s.mu.Unlock()
			continue
		}
		s.dispatchLater(ctx, e, options)
	}
	return nil
}

// Publish adapts the direct workflow publisher port to Medusa-shaped event records.
func (s *Service) Publish(ctx context.Context, name string, rows []map[string]any) error {
	events := make([]models.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, models.Event{Name: name, Data: row})
	}
	return s.Emit(ctx, events, models.Options{})
}
func (s *Service) dispatchLater(ctx context.Context, e models.Event, opts models.Options) {
	delay := opts.Delay
	if delay < 0 {
		delay = 0
	}
	time.AfterFunc(delay, func() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		e.Metadata = cloneMetadata(e.Metadata)
		e.Metadata["published_at"] = time.Now().UTC().Format(time.RFC3339Nano)
		s.mu.RLock()
		listeners := append([]subscriber(nil), s.subscribers[e.Name]...)
		listeners = append(listeners, s.subscribers["*"]...)
		s.mu.RUnlock()
		for _, sub := range listeners {
			if err := sub.fn(ctx, e); err != nil {
				s.logger.Error("event subscriber failed", "event", e.Name, "subscriber", sub.id, "error", err)
			}
		}
	})
}

func cloneMetadata(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}
func (s *Service) ReleaseGroupedEvents(ctx context.Context, id string) error {
	s.mu.Lock()
	events := append([]models.Event(nil), s.grouped[id]...)
	delete(s.grouped, id)
	s.mu.Unlock()
	for _, e := range events {
		s.dispatchLater(ctx, e, models.Options{})
	}
	return nil
}
func (s *Service) ClearGroupedEvents(id string, names []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(names) == 0 {
		delete(s.grouped, id)
		return
	}
	skip := map[string]bool{}
	for _, n := range names {
		skip[n] = true
	}
	kept := s.grouped[id][:0]
	for _, e := range s.grouped[id] {
		if !skip[e.Name] {
			kept = append(kept, e)
		}
	}
	s.grouped[id] = kept
}

var _ domain.EventBus = (*Service)(nil)
