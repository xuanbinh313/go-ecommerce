package posthog

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/analytics/domain/models"
)

const Identifier = "analytics-posthog"

type event struct {
	DistinctID     string         `json:"distinct_id"`
	Event          string         `json:"event"`
	Properties     map[string]any `json:"properties"`
	Type           string         `json:"type"`
	Library        string         `json:"library"`
	LibraryVersion string         `json:"library_version"`
	Timestamp      string         `json:"timestamp"`
	UUID           string         `json:"uuid"`
}

type Provider struct {
	apiKey    string
	host      string
	client    *http.Client
	logger    *slog.Logger
	mu        sync.Mutex
	queue     []event
	wake      chan struct{}
	stop      chan struct{}
	done      chan struct{}
	closed    bool
	closeOnce sync.Once
}

func New(apiKey, host string, client *http.Client, logger *slog.Logger) (*Provider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("Posthog API key is not set, but is required")
	}
	if host == "" {
		host = "https://eu.i.posthog.com"
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if logger == nil {
		logger = slog.Default()
	}
	p := &Provider{apiKey: apiKey, host: strings.TrimRight(host, "/"), client: client, logger: logger, wake: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
	go p.run()
	return p, nil
}
func (*Provider) Identifier() string { return Identifier }

func (p *Provider) Track(_ context.Context, input models.TrackEvent) error {
	if input.Event == "" {
		return errors.New("Event name is required when tracking an event with Posthog")
	}
	if input.ActorID == "" {
		return errors.New("Actor ID is required when tracking an event with Posthog")
	}
	props := copyProperties(input.Properties)
	if input.Group != nil && input.Group.ID != "" {
		if input.Group.Type == "" {
			return errors.New("Group type is required if passing group id when tracking an event with Posthog")
		}
		props["$groups"] = map[string]string{input.Group.Type: input.Group.ID}
	}
	return p.enqueue(event{DistinctID: input.ActorID, Event: input.Event, Properties: props})
}

func (p *Provider) Identify(_ context.Context, input models.IdentifyEvent) error {
	props := copyProperties(input.Properties)
	identity := input.ActorID
	eventName := "$identify"
	if input.Group != nil {
		if input.Group.ID == "" || input.Group.Type == "" {
			return errors.New("Group type and id are required when identifying a group with Posthog")
		}
		identity = "$" + input.Group.Type + "_" + input.Group.ID
		eventName = "$groupidentify"
		groupProperties := copyProperties(props)
		props["$group_type"] = input.Group.Type
		props["$group_key"] = input.Group.ID
		props["$group_set"] = groupProperties
	} else if identity == "" {
		return errors.New("Actor or group is required when identifying an entity with Posthog")
	}
	return p.enqueue(event{DistinctID: identity, Event: eventName, Properties: props})
}

func (p *Provider) enqueue(item event) error {
	if item.Properties == nil {
		item.Properties = map[string]any{}
	}
	item.Properties["$geoip_disable"] = true
	uuid, err := uuidv7(time.Now())
	if err != nil {
		return fmt.Errorf("create Posthog event UUID: %w", err)
	}
	item.Type = "capture"
	item.Library = "posthog-node"
	item.LibraryVersion = "4.17.1"
	item.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	item.UUID = uuid
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return errors.New("Posthog provider is shut down")
	}
	p.queue = append(p.queue, item)
	shouldWake := len(p.queue) >= 100
	p.mu.Unlock()
	if shouldWake {
		select {
		case p.wake <- struct{}{}:
		default:
		}
	}
	return nil
}

func uuidv7(now time.Time) (string, error) {
	var raw [16]byte
	ms := uint64(now.UnixMilli())
	for i := 5; i >= 0; i-- {
		raw[i] = byte(ms)
		ms >>= 8
	}
	if _, err := rand.Read(raw[6:]); err != nil {
		return "", err
	}
	raw[6] = (raw[6] & 0x0f) | 0x70
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s", hex.EncodeToString(raw[0:4]), hex.EncodeToString(raw[4:6]), hex.EncodeToString(raw[6:8]), hex.EncodeToString(raw[8:10]), hex.EncodeToString(raw[10:16])), nil
}

func (p *Provider) run() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	defer close(p.done)
	for {
		select {
		case <-p.wake:
			p.flush()
		case <-ticker.C:
			p.flush()
		case <-p.stop:
			for p.flush() {
			}
			return
		}
	}
}

func (p *Provider) flush() bool {
	p.mu.Lock()
	if len(p.queue) == 0 {
		p.mu.Unlock()
		return false
	}
	batchSize := len(p.queue)
	if batchSize > 100 {
		batchSize = 100
	}
	batch := append([]event(nil), p.queue[:batchSize]...)
	p.queue = p.queue[batchSize:]
	p.mu.Unlock()
	body, err := json.Marshal(map[string]any{"api_key": p.apiKey, "batch": batch, "sent_at": time.Now().UTC().Format(time.RFC3339Nano)})
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, p.host+"/batch/", bytes.NewReader(body))
		if reqErr != nil {
			err = reqErr
		} else {
			req.Header.Set("Content-Type", "application/json")
			resp, sendErr := p.client.Do(req)
			if sendErr != nil {
				err = sendErr
			} else {
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					responseBody, _ := io.ReadAll(resp.Body)
					err = fmt.Errorf("Posthog batch returned %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
				}
				resp.Body.Close()
			}
		}
		cancel()
	}
	if err != nil {
		p.logger.Error("Posthog analytics batch failed", "error", err)
		p.mu.Lock()
		p.queue = append(batch, p.queue...)
		p.mu.Unlock()
		return false
	}
	p.mu.Lock()
	pending := len(p.queue) > 0
	p.mu.Unlock()
	return pending
}

func (p *Provider) Shutdown(ctx context.Context) error {
	p.closeOnce.Do(func() { p.mu.Lock(); p.closed = true; p.mu.Unlock(); close(p.stop) })
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func copyProperties(input map[string]any) map[string]any {
	out := make(map[string]any, len(input)+4)
	for key, value := range input {
		out[key] = value
	}
	return out
}
