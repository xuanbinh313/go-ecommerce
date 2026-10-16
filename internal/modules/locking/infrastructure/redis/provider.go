package redis

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	mathrand "math/rand"
	"strings"
	"sync"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/locking/domain"
	redisv9 "github.com/redis/go-redis/v9"
)

const Identifier = "locking-redis"
const defaultExpiration = 60 * time.Second

var acquireScript = redisv9.NewScript(`
local key = KEYS[1]
local ownerId = ARGV[1]
local ttl = tonumber(ARGV[2])
local awaitQueue = ARGV[3] == 'true'
local setArgs = {key, ownerId, 'NX'}
if ttl > 0 then table.insert(setArgs, 'EX'); table.insert(setArgs, ttl) end
local setResult = redis.call('SET', unpack(setArgs))
if setResult then return 1 end
if awaitQueue then return 0 end
local currentOwnerId = redis.call('GET', key)
if currentOwnerId == '*' then return 0 end
if currentOwnerId == ownerId then
  setArgs = {key, ownerId, 'XX'}
  if ttl > 0 then table.insert(setArgs, 'EX'); table.insert(setArgs, ttl) end
  redis.call('SET', unpack(setArgs))
  return 1
end
return 0
`)

var releaseScript = redisv9.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('DEL', KEYS[1]) end
return 0
`)
var extendScript = redisv9.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('EXPIRE', KEYS[1], ARGV[2]) end
return 0
`)

type Options struct {
	WaitLockingTimeout   time.Duration
	DefaultRetryInterval time.Duration
	MaximumRetryInterval time.Duration
	BackoffFactor        float64
}

type Provider struct {
	client  *redisv9.Client
	prefix  string
	options Options
	logger  *slog.Logger
}

func NewProvider(redisURL, prefix string, options Options, logger *slog.Logger) (*Provider, error) {
	if strings.TrimSpace(redisURL) == "" {
		return nil, errors.New(`No "redisUrl" provided for locking-redis provider`)
	}
	parsed, err := redisv9.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	if prefix == "" {
		prefix = "medusa_lock:"
	}
	if options.WaitLockingTimeout <= 0 {
		options.WaitLockingTimeout = 5 * time.Second
	}
	if options.DefaultRetryInterval <= 0 {
		options.DefaultRetryInterval = 20 * time.Millisecond
	}
	if options.MaximumRetryInterval <= 0 {
		options.MaximumRetryInterval = time.Second
	}
	if options.BackoffFactor <= 0 {
		options.BackoffFactor = 2
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Provider{client: redisv9.NewClient(parsed), prefix: prefix, options: options, logger: logger}, nil
}
func (*Provider) Identifier() string      { return Identifier }
func (p *Provider) Close() error          { return p.client.Close() }
func (p *Provider) key(key string) string { return p.prefix + key }

func (p *Provider) Execute(ctx context.Context, keys []string, timeout time.Duration, job func(context.Context) error) error {
	if timeout <= 0 {
		timeout = p.options.WaitLockingTimeout
	}
	if timeout < time.Second {
		timeout = time.Second
	}
	lockCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	owner := "execute:" + newUUID()
	if err := p.acquire(lockCtx, keys, domain.LockOptions{OwnerID: owner, Expire: defaultExpiration, AwaitQueue: true}); err != nil {
		return err
	}
	jobCtx, jobCancel := context.WithCancelCause(ctx)
	renewDone := make(chan struct{})
	go p.renewUntilStopped(jobCtx, keys, owner, defaultExpiration, jobCancel, renewDone)
	jobErr := job(jobCtx)
	renewalErr := context.Cause(jobCtx)
	jobCancel(context.Canceled)
	<-renewDone
	releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer releaseCancel()
	if _, err := p.Release(releaseCtx, keys, owner); err != nil && jobErr == nil {
		jobErr = err
	}
	if renewalErr != nil {
		return renewalErr
	}
	return jobErr
}

func (p *Provider) renewUntilStopped(ctx context.Context, keys []string, owner string, expire time.Duration, cancel context.CancelCauseFunc, done chan<- struct{}) {
	defer close(done)
	allKeys := normalize(keys)
	interval := expire / 3
	if interval < 250*time.Millisecond {
		interval = 250 * time.Millisecond
	}
	timer := time.NewTicker(interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			for _, key := range allKeys {
				result, err := extendScript.Run(ctx, p.client, []string{p.key(key)}, owner, int64(expire.Seconds())).Int()
				if err != nil {
					p.logger.Warn("Failed to renew lock", "key", key, "error", err)
					continue
				}
				if result != 1 {
					message := fmt.Sprintf(`Lost the lock for key(s) %q while the job was still running: the lease expired before it could be renewed. Increase "expire" if the job is expected to run this long.`, key)
					p.logger.Warn(message)
					cancel(errors.New(message))
					return
				}
			}
		}
	}
}

func (p *Provider) Acquire(ctx context.Context, keys []string, options domain.LockOptions) error {
	return p.acquire(ctx, keys, options)
}
func (p *Provider) acquire(ctx context.Context, keys []string, options domain.LockOptions) error {
	keys = normalize(keys)
	if len(keys) == 0 {
		return errors.New("at least one lock key is required")
	}
	owner := options.OwnerID
	if owner == "" {
		owner = "*"
	}
	var wg sync.WaitGroup
	errorsCh := make(chan error, len(keys))
	for _, key := range keys {
		key := key
		wg.Add(1)
		go func() {
			defer wg.Done()
			retry := p.options.DefaultRetryInterval
			for {
				select {
				case <-ctx.Done():
					errorsCh <- ctx.Err()
					return
				default:
				}
				ttl := int64(0)
				if options.Expire > 0 {
					ttl = int64(options.Expire.Seconds())
					if ttl < 1 {
						ttl = 1
					}
				}
				result, err := acquireScript.Run(ctx, p.client, []string{p.key(key)}, owner, ttl, options.AwaitQueue).Int()
				if err != nil {
					errorsCh <- err
					return
				}
				if result == 1 {
					return
				}
				if !options.AwaitQueue {
					errorsCh <- fmt.Errorf("Failed to acquire lock for key %q", key)
					return
				}
				jitter := time.Duration(float64(retry) * (0.5 + mathrand.Float64()*0.5))
				timer := time.NewTimer(jitter)
				select {
				case <-ctx.Done():
					timer.Stop()
					errorsCh <- ctx.Err()
					return
				case <-timer.C:
				}
				next := time.Duration(float64(retry) * p.options.BackoffFactor)
				if next > p.options.MaximumRetryInterval {
					next = p.options.MaximumRetryInterval
				}
				retry = next
			}
		}()
	}
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) Release(ctx context.Context, keys []string, owner string) (bool, error) {
	if owner == "" {
		owner = "*"
	}
	keys = normalize(keys)
	results := make(chan bool, len(keys))
	errCh := make(chan error, len(keys))
	var wg sync.WaitGroup
	for _, key := range keys {
		key := key
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := releaseScript.Run(ctx, p.client, []string{p.key(key)}, owner).Int()
			if err != nil {
				errCh <- err
				return
			}
			results <- n == 1
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return false, err
		}
	}
	close(results)
	ok := true
	for result := range results {
		ok = ok && result
	}
	return ok, nil
}

func (p *Provider) ReleaseAll(ctx context.Context, ownerID *string) error {
	owner := "*"
	if ownerID != nil {
		owner = *ownerID
	}
	pattern := p.prefix + "*"
	var cursor uint64
	for {
		keys, next, err := p.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			pipe := p.client.Pipeline()
			gets := make([]*redisv9.StringCmd, len(keys))
			for i, key := range keys {
				gets[i] = pipe.Get(ctx, key)
			}
			_, _ = pipe.Exec(ctx)
			del := p.client.Pipeline()
			for i, key := range keys {
				if gets[i].Val() == owner {
					del.Unlink(ctx, key)
				}
			}
			if _, err := del.Exec(ctx); err != nil && err != redisv9.Nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}

func normalize(keys []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if key != "" && !seen[key] {
			seen[key] = true
			out = append(out, key)
		}
	}
	return out
}
func newUUID() string {
	var id [16]byte
	if _, err := crand.Read(id[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:16])
}

var _ domain.Provider = (*Provider)(nil)
