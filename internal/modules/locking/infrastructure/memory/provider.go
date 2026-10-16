package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/locking/domain"
)

type lock struct {
	owner   string
	expires time.Time
	changed chan struct{}
}
type Provider struct {
	mu    sync.Mutex
	locks map[string]*lock
}

func NewProvider() *Provider { return &Provider{locks: map[string]*lock{}} }
func normalize(keys []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != "" && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}
func (p *Provider) Acquire(ctx context.Context, keys []string, opt domain.LockOptions) error {
	keys = normalize(keys)
	if len(keys) == 0 {
		return errors.New("at least one lock key is required")
	}
	for {
		p.mu.Lock()
		now := time.Now()
		var wait <-chan struct{}
		var conflict string
		for _, key := range keys {
			l := p.locks[key]
			if l == nil || (!l.expires.IsZero() && !l.expires.After(now)) {
				if l != nil {
					close(l.changed)
				}
				expires := time.Time{}
				if opt.Expire > 0 {
					expires = now.Add(opt.Expire)
				}
				p.locks[key] = &lock{owner: opt.OwnerID, expires: expires, changed: make(chan struct{})}
				continue
			}
			if l.owner == opt.OwnerID && opt.OwnerID != "" {
				if opt.Expire > 0 {
					l.expires = now.Add(opt.Expire)
				}
				continue
			}
			conflict = key
			wait = l.changed
			break
		}
		p.mu.Unlock()
		if conflict == "" {
			return nil
		}
		if !opt.AwaitQueue {
			return fmt.Errorf("failed to acquire lock for key %q", conflict)
		}
		var expiry <-chan time.Time
		if l := p.expiration(conflict); !l.IsZero() {
			d := time.Until(l)
			if d < 0 {
				d = 0
			}
			t := time.NewTimer(d)
			defer t.Stop()
			expiry = t.C
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-wait:
		case <-expiry:
		}
	}
}
func (p *Provider) expiration(key string) time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	if l := p.locks[key]; l != nil {
		return l.expires
	}
	return time.Time{}
}
func (p *Provider) Release(ctx context.Context, keys []string, owner string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	success := true
	for _, key := range normalize(keys) {
		l := p.locks[key]
		if l == nil || (owner != "" && l.owner != owner) {
			success = false
			continue
		}
		delete(p.locks, key)
		close(l.changed)
	}
	return success, nil
}
func (p *Provider) ReleaseAll(ctx context.Context, owner *string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, l := range p.locks {
		if owner == nil || l.owner == *owner {
			delete(p.locks, k)
			close(l.changed)
		}
	}
	return nil
}
func (p *Provider) Execute(ctx context.Context, keys []string, timeout time.Duration, job func(context.Context) error) error {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	lockCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	owner := fmt.Sprintf("execute:%p:%d", p, time.Now().UnixNano())
	if err := p.Acquire(lockCtx, keys, domain.LockOptions{OwnerID: owner, AwaitQueue: true}); err != nil {
		return err
	}
	defer p.Release(context.Background(), keys, owner)
	return job(ctx)
}

var _ domain.Provider = (*Provider)(nil)
