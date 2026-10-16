package application

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/event-bus-redis/domain"
	"github.com/example/medusa-go-clone/internal/modules/event-bus-redis/domain/models"
)

const (
	DefaultQueueName        = "events-queue"
	DefaultPriority         = 100
	InternalPriority        = 2097152
	MinimumPriority         = 1
	MaximumPriority         = InternalPriority
	DefaultGroupedEventsTTL = 600
	groupedEventsChunkSize  = 1000
)

type Service struct {
	queue       domain.Queue
	redis       domain.StagingRedis
	interceptor domain.Interceptor
	logger      domain.Logger
	jobDefaults models.JobOptions
	mu          sync.RWMutex
	subscribers map[string][]subscriber
}

type subscriber struct {
	id string
	fn domain.Subscriber
}

func New(queue domain.Queue, redis domain.StagingRedis, interceptor domain.Interceptor, logger domain.Logger, jobDefaults models.JobOptions) *Service {
	return &Service{queue: queue, redis: redis, interceptor: interceptor, logger: logger, jobDefaults: jobDefaults, subscribers: map[string][]subscriber{}}
}

func (s *Service) Subscribe(eventName, id string, fn domain.Subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.subscribers[eventName] {
		if id != "" && existing.id == id {
			return
		}
	}
	s.subscribers[eventName] = append(s.subscribers[eventName], subscriber{id: id, fn: fn})
}
func (s *Service) Unsubscribe(eventName, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.subscribers[eventName]
	next := list[:0]
	for _, sub := range list {
		if sub.id != id {
			next = append(next, sub)
		}
	}
	if len(next) == 0 {
		delete(s.subscribers, eventName)
	} else {
		s.subscribers[eventName] = next
	}
}
func (s *Service) hasSubscribers(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subscribers[name])+len(s.subscribers["*"]) > 0
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
func stampMetadata(in map[string]any, key string) map[string]any {
	out := cloneMap(in)
	if _, ok := out[key]; !ok {
		out[key] = time.Now().UTC().Format(time.RFC3339Nano)
	}
	return out
}
func (s *Service) buildJob(e models.Event, options models.EmitOptions) models.Job {
	// Medusa's precedence is message-level, emit-level, module-level, then its
	// internal/default priority. Preserve extension options for the BullMQ adapter.
	priority := options.Priority
	if priority == 0 {
		priority = s.jobDefaults.Priority
	}
	if priority == 0 {
		if options.Internal {
			priority = InternalPriority
		} else {
			priority = DefaultPriority
		}
	}
	if p, ok := e.Options["priority"].(int); ok {
		priority = p
	} else if p, ok := e.Options["priority"].(float64); ok {
		priority = int(p)
	}
	if priority < MinimumPriority || priority > MaximumPriority {
		if s.logger != nil {
			s.logger.Warn("invalid event priority %d for %s; using default", priority, e.Name)
		}
		priority = DefaultPriority
	}
	attempts := options.Attempts
	if attempts == 0 {
		attempts = s.jobDefaults.Attempts
	}
	if attempts == 0 {
		attempts = 1
	}
	data := map[string]any{"data": e.Data, "metadata": e.Metadata}
	return models.Job{Name: e.Name, Data: data, Opts: models.JobOptions{Attempts: attempts, Priority: priority, RemoveOnComplete: s.jobDefaults.RemoveOnComplete, Delay: options.Delay, Extra: cloneMap(s.jobDefaults.Extra)}}
}

func (s *Service) Emit(ctx context.Context, events []models.Event, options models.EmitOptions) error {
	if options.GroupedEventsTTL == 0 {
		options.GroupedEventsTTL = DefaultGroupedEventsTTL
	}
	groups := map[string][]models.Job{}
	var normal []models.Job
	for _, event := range events {
		event.Metadata = stampMetadata(event.Metadata, "created_at")
		job := s.buildJob(event, options)
		groupID, _ := event.Metadata["eventGroupId"].(string)
		if groupID != "" {
			groups[groupID] = append(groups[groupID], job)
			continue
		}
		if s.interceptor != nil {
			s.interceptor.BeforeEmit(ctx, event, false, "")
		}
		if s.hasSubscribers(event.Name) {
			metadata := stampMetadata(event.Metadata, "published_at")
			job.Data["metadata"] = metadata
			normal = append(normal, job)
		}
	}
	if len(normal) > 0 {
		if err := s.queue.AddBulk(ctx, normal); err != nil {
			return err
		}
	}
	for groupID, jobs := range groups {
		if err := s.stage(ctx, groupID, jobs, time.Duration(options.GroupedEventsTTL)*time.Second); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) stage(ctx context.Context, groupID string, jobs []models.Job, ttl time.Duration) error {
	encoded := make([]string, len(jobs))
	for i, j := range jobs {
		b, err := json.Marshal(j)
		if err != nil {
			return err
		}
		encoded[i] = string(b)
	}
	key := "staging:" + groupID
	if err := s.redis.RPush(ctx, key, encoded...); err != nil {
		return err
	}
	return s.redis.Expire(ctx, key, ttl)
}

func (s *Service) readGroup(ctx context.Context, groupID string) ([]models.Job, error) {
	rows, err := s.redis.LRange(ctx, "staging:"+groupID, 0, -1)
	if err != nil {
		return nil, err
	}
	jobs := make([]models.Job, 0, len(rows))
	for _, row := range rows {
		var job models.Job
		if err = json.Unmarshal([]byte(row), &job); err != nil {
			return nil, fmt.Errorf("decode staged event: %w", err)
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (s *Service) ReleaseGroupedEvents(ctx context.Context, groupID string) error {
	jobs, err := s.readGroup(ctx, groupID)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if s.interceptor != nil {
			event := models.Event{Name: job.Name}
			if data, ok := job.Data["data"]; ok {
				event.Data = data
			}
			if md, ok := job.Data["metadata"].(map[string]any); ok {
				event.Metadata = md
			}
			s.interceptor.BeforeEmit(ctx, event, true, groupID)
		}
	}
	queued := make([]models.Job, 0, len(jobs))
	for _, job := range jobs {
		if !s.hasSubscribers(job.Name) {
			continue
		}
		if metadata, ok := job.Data["metadata"].(map[string]any); ok {
			job.Data["metadata"] = stampMetadata(metadata, "published_at")
		}
		queued = append(queued, job)
	}
	if len(queued) > 0 {
		if err = s.queue.AddBulk(ctx, queued); err != nil {
			return err
		}
	}
	return s.ClearGroupedEvents(ctx, groupID, nil)
}

// ProcessJob mirrors the source worker's subscriber checkpointing: successful
// subscribers are recorded and skipped on a retry; failed subscribers may be
// retried only when the emitted job configured multiple attempts.
func (s *Service) ProcessJob(ctx context.Context, job models.Job, attemptsMade int) (models.Job, []any, error) {
	name := job.Name
	data, metadata := job.Data["data"], job.Data["metadata"]
	event := models.Event{Name: name, Data: data}
	if values, ok := metadata.(map[string]any); ok {
		event.Metadata = values
	}
	s.mu.RLock()
	all := append([]subscriber(nil), s.subscribers[name]...)
	all = append(all, s.subscribers["*"]...)
	s.mu.RUnlock()
	completed := map[string]bool{}
	for _, id := range job.CompletedSubscriberIDs {
		completed[id] = true
	}
	results := make([]any, 0, len(all))
	newlyCompleted := []string{}
	failed := false
	for _, sub := range all {
		if sub.id == "" || completed[sub.id] {
			continue
		}
		if err := sub.fn(ctx, event); err != nil {
			results = append(results, err)
			failed = true
			if s.logger != nil {
				s.logger.Warn("event subscriber %s failed for %s: %v", sub.id, name, err)
			}
		} else {
			newlyCompleted = append(newlyCompleted, sub.id)
		}
	}
	job.CompletedSubscriberIDs = append(job.CompletedSubscriberIDs, newlyCompleted...)
	finalAttempt := attemptsMade >= job.Opts.Attempts
	if failed && job.Opts.Attempts > 1 && !finalAttempt {
		return job, results, fmt.Errorf("one or more subscribers of %s failed; retrying", name)
	}
	return job, results, nil
}

func (s *Service) ClearGroupedEvents(ctx context.Context, groupID string, eventNames []string) error {
	if groupID == "" {
		return nil
	}
	key := "staging:" + groupID
	if len(eventNames) == 0 {
		return s.redis.Unlink(ctx, key)
	}
	rows, err := s.redis.LRange(ctx, key, 0, -1)
	if err != nil {
		return err
	}
	ttl, err := s.redis.PTTL(ctx, key)
	if err != nil {
		return err
	}
	remove := map[string]bool{}
	for _, name := range eventNames {
		remove[name] = true
	}
	kept := make([]models.Job, 0, len(rows))
	for _, row := range rows {
		var job models.Job
		if err = json.Unmarshal([]byte(row), &job); err != nil {
			return err
		}
		if !remove[job.Name] {
			kept = append(kept, job)
		}
	}
	if err = s.redis.Del(ctx, key); err != nil {
		return err
	}
	for start := 0; start < len(kept); start += groupedEventsChunkSize {
		end := start + groupedEventsChunkSize
		if end > len(kept) {
			end = len(kept)
		}
		batch := make([]string, 0, end-start)
		for _, job := range kept[start:end] {
			b, e := json.Marshal(job)
			if e != nil {
				return e
			}
			batch = append(batch, string(b))
		}
		if err = s.redis.RPush(ctx, key, batch...); err != nil {
			return err
		}
	}
	if len(kept) > 0 && ttl > 0 {
		return s.redis.Expire(ctx, key, ttl)
	}
	return nil
}
