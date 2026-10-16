package redis

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/example/medusa-go-clone/internal/modules/caching/domain"
	cachememory "github.com/example/medusa-go-clone/internal/modules/caching/infrastructure/memory"
	redisv9 "github.com/redis/go-redis/v9"
)

const Identifier = "cache-redis"

type Options struct {
	RedisURL             string
	Prefix               string
	TTLSeconds           int
	CompressionThreshold int
}

type Provider struct {
	client               *redisv9.Client
	prefix               string
	ttl                  int
	compressionThreshold int
	logger               *slog.Logger
}

func NewProvider(options Options, logger *slog.Logger) (*Provider, error) {
	if strings.TrimSpace(options.RedisURL) == "" {
		return nil, errors.New("[caching-redis] redisUrl is required")
	}
	parsed, err := redisv9.ParseURL(options.RedisURL)
	if err != nil {
		return nil, err
	}
	if options.Prefix == "" {
		options.Prefix = "mc:"
	}
	if options.TTLSeconds == 0 {
		options.TTLSeconds = 3600
	}
	if options.CompressionThreshold == 0 {
		options.CompressionThreshold = 2048
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Provider{client: redisv9.NewClient(parsed), prefix: options.Prefix, ttl: options.TTLSeconds, compressionThreshold: options.CompressionThreshold, logger: logger}, nil
}
func (*Provider) Identifier() string { return Identifier }
func (p *Provider) Close() error     { return p.client.Close() }

func (p *Provider) Get(ctx context.Context, key string, tags []string) (any, error) {
	if key != "" {
		buffer, err := p.client.HGet(ctx, p.key(key), "data").Bytes()
		if errors.Is(err, redisv9.Nil) {
			return nil, nil
		}
		if err != nil {
			if p.connectionFailure(ctx, err, "get") {
				return nil, nil
			}
			return nil, err
		}
		decoded, err := decompress(buffer)
		if err != nil {
			return nil, err
		}
		var value any
		if err = json.Unmarshal([]byte(decoded), &value); err != nil {
			return nil, err
		}
		return value, nil
	}
	if len(tags) == 0 {
		return nil, nil
	}
	pipeline := p.client.Pipeline()
	commands := make([]*redisv9.StringSliceCmd, len(tags))
	for i, tag := range tags {
		commands[i] = pipeline.SMembers(ctx, p.tagKey(tag, false))
	}
	_, err := pipeline.Exec(ctx)
	if err != nil && !errors.Is(err, redisv9.Nil) {
		if p.connectionFailure(ctx, err, "get") {
			return nil, nil
		}
		return nil, err
	}
	keys := map[string]struct{}{}
	for _, command := range commands {
		members, cmdErr := command.Result()
		if cmdErr != nil && !errors.Is(cmdErr, redisv9.Nil) {
			if p.connectionFailure(ctx, cmdErr, "get") {
				return nil, nil
			}
			return nil, cmdErr
		}
		for _, keyName := range members {
			keys[keyName] = struct{}{}
		}
	}
	if len(keys) == 0 {
		return []any{}, nil
	}
	valuePipe := p.client.Pipeline()
	keyNames := make([]string, 0, len(keys))
	valueCommands := make([]*redisv9.StringCmd, 0, len(keys))
	for keyName := range keys {
		keyNames = append(keyNames, keyName)
		valueCommands = append(valueCommands, valuePipe.HGet(ctx, keyName, "data"))
	}
	_, err = valuePipe.Exec(ctx)
	if err != nil && !errors.Is(err, redisv9.Nil) {
		if p.connectionFailure(ctx, err, "get") {
			return nil, nil
		}
		return nil, err
	}
	results := make([]any, 0, len(keys))
	for i, command := range valueCommands {
		raw, cmdErr := command.Bytes()
		if cmdErr != nil {
			if errors.Is(cmdErr, redisv9.Nil) {
				continue
			}
			if p.connectionFailure(ctx, cmdErr, "get") {
				return nil, nil
			}
			return nil, cmdErr
		}
		decoded, deErr := decompress(raw)
		if deErr != nil {
			p.logger.Warn("Skipping corrupted cache entry", "key", keyNames[i], "error", deErr)
			continue
		}
		var value any
		if jsonErr := json.Unmarshal([]byte(decoded), &value); jsonErr != nil {
			p.logger.Warn("Skipping corrupted cache entry", "key", keyNames[i], "error", jsonErr)
			continue
		}
		if truthy(value) {
			results = append(results, value)
		}
	}
	return results, nil
}

func (p *Provider) Set(ctx context.Context, key string, tags []string, value any, ttl int, options map[string]any) error {
	serialized, err := json.Marshal(value)
	if err != nil {
		return err
	}
	encoded, err := compress(serialized, utf16Length(string(serialized)) > p.compressionThreshold)
	if err != nil {
		return err
	}
	keyName := p.key(key)
	effectiveTTL := ttl
	if effectiveTTL < 0 {
		effectiveTTL = p.ttl
	}
	pipe := p.client.Pipeline()
	pipe.HSet(ctx, keyName, "data", encoded)
	if len(options) > 0 {
		optionsJSON, e := json.Marshal(options)
		if e != nil {
			return e
		}
		pipe.HSet(ctx, keyName, "options", optionsJSON)
	}
	if effectiveTTL != 0 {
		pipe.Expire(ctx, keyName, time.Duration(effectiveTTL)*time.Second)
	}
	hashedTags := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, tag := range tags {
		hashed := cachememory.Hash(tag)
		if !seen[hashed] {
			seen[hashed] = true
			hashedTags = append(hashedTags, hashed)
		}
	}
	if len(hashedTags) > 0 {
		tagsKey := p.tagsKey(key)
		members := make([]any, len(hashedTags))
		for i, tag := range hashedTags {
			members[i] = tag
		}
		pipe.SAdd(ctx, tagsKey, members...)
		if effectiveTTL != 0 {
			pipe.Expire(ctx, tagsKey, time.Duration(effectiveTTL+60)*time.Second)
		}
		for _, hashed := range hashedTags {
			tagKey := p.tagKey(hashed, true)
			pipe.SAdd(ctx, tagKey, keyName)
			if effectiveTTL != 0 {
				pipe.Expire(ctx, tagKey, time.Duration(effectiveTTL+60)*time.Second)
			}
		}
	}
	_, err = pipe.Exec(ctx)
	if err != nil && p.connectionFailure(ctx, err, "set") {
		return nil
	}
	return err
}

func (p *Provider) Clear(ctx context.Context, key string, tags []string, options map[string]any) error {
	if key != "" {
		return p.deleteEntries(ctx, []string{p.key(key)})
	}
	if len(tags) == 0 {
		return nil
	}
	for _, tag := range tags {
		if tag == "*" {
			return p.flush(ctx)
		}
	}
	pipeline := p.client.Pipeline()
	commands := make([]*redisv9.StringSliceCmd, len(tags))
	for i, tag := range tags {
		commands[i] = pipeline.SMembers(ctx, p.tagKey(tag, false))
	}
	_, err := pipeline.Exec(ctx)
	if err != nil && !errors.Is(err, redisv9.Nil) {
		if p.connectionFailure(ctx, err, "clear") {
			return nil
		}
		return err
	}
	all := map[string]struct{}{}
	for _, command := range commands {
		members, cmdErr := command.Result()
		if cmdErr != nil && !errors.Is(cmdErr, redisv9.Nil) {
			if p.connectionFailure(ctx, cmdErr, "clear") {
				return nil
			}
			return cmdErr
		}
		for _, name := range members {
			all[name] = struct{}{}
		}
	}
	keys := make([]string, 0, len(all))
	for name := range all {
		keys = append(keys, name)
	}
	if options == nil {
		return p.deleteEntries(ctx, keys)
	}
	autoInvalidate, _ := options["autoInvalidate"].(bool)
	if !autoInvalidate {
		return nil
	}
	optionsPipe := p.client.Pipeline()
	optionCommands := make([]*redisv9.StringCmd, len(keys))
	for i, name := range keys {
		optionCommands[i] = optionsPipe.HGet(ctx, name, "options")
	}
	_, err = optionsPipe.Exec(ctx)
	if err != nil && !errors.Is(err, redisv9.Nil) {
		if p.connectionFailure(ctx, err, "clear") {
			return nil
		}
		return err
	}
	deleteKeys := make([]string, 0, len(keys))
	for i, command := range optionCommands {
		raw, cmdErr := command.Result()
		if errors.Is(cmdErr, redisv9.Nil) {
			deleteKeys = append(deleteKeys, keys[i])
			continue
		}
		if cmdErr != nil {
			if p.connectionFailure(ctx, cmdErr, "clear") {
				return nil
			}
			return cmdErr
		}
		var entryOptions map[string]any
		if jsonErr := json.Unmarshal([]byte(raw), &entryOptions); jsonErr != nil {
			deleteKeys = append(deleteKeys, keys[i])
			continue
		}
		if auto, exists := entryOptions["autoInvalidate"].(bool); !exists || auto {
			deleteKeys = append(deleteKeys, keys[i])
		}
	}
	return p.deleteEntries(ctx, deleteKeys)
}

func (p *Provider) deleteEntries(ctx context.Context, keyNames []string) error {
	if len(keyNames) == 0 {
		return nil
	}
	for _, keyName := range keyNames {
		logical := strings.TrimPrefix(keyName, p.prefix)
		tagsKey := p.tagsKey(logical)
		hashedTags, err := p.client.SMembers(ctx, tagsKey).Result()
		if err != nil && !errors.Is(err, redisv9.Nil) {
			if p.connectionFailure(ctx, err, "clear") {
				return nil
			}
			return err
		}
		cleanup := p.client.Pipeline()
		for _, tag := range hashedTags {
			cleanup.SRem(ctx, p.tagKey(tag, true), keyName)
		}
		cleanup.Unlink(ctx, tagsKey)
		if _, err = cleanup.Exec(ctx); err != nil {
			if p.connectionFailure(ctx, err, "clear") {
				return nil
			}
			return err
		}
	}
	return p.client.Unlink(ctx, keyNames...).Err()
}

func (p *Provider) flush(ctx context.Context) error {
	var cursor uint64
	pattern := p.prefix + "*"
	for {
		keys, next, err := p.client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			if p.connectionFailure(ctx, err, "flush") {
				return nil
			}
			return err
		}
		if len(keys) > 0 {
			if err = p.client.Unlink(ctx, keys...).Err(); err != nil {
				if p.connectionFailure(ctx, err, "flush") {
					return nil
				}
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}

func (p *Provider) key(key string) string { return p.prefix + key }
func (p *Provider) tagKey(tag string, hashed bool) string {
	if !hashed {
		tag = cachememory.Hash(tag)
	}
	return p.prefix + "tag:" + tag
}
func (p *Provider) tagsKey(key string) string { return p.prefix + "tags:" + key }

func compress(data []byte, should bool) ([]byte, error) {
	if !should {
		return append([]byte{0}, data...), nil
	}
	var buffer bytes.Buffer
	buffer.WriteByte(1)
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(data); err != nil {
		return append([]byte{0}, data...), nil
	}
	if err := writer.Close(); err != nil {
		return append([]byte{0}, data...), nil
	}
	return buffer.Bytes(), nil
}
func decompress(data []byte) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	switch data[0] {
	case 0:
		return string(data[1:]), nil
	case 1:
		reader, err := gzip.NewReader(bytes.NewReader(data[1:]))
		if err != nil {
			return string(data[1:]), nil
		}
		decoded, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			return string(data[1:]), nil
		}
		return string(decoded), nil
	default:
		return string(data), nil
	}
}
func utf16Length(value string) int { return len(utf16.Encode([]rune(value))) }
func truthy(value any) bool {
	switch item := value.(type) {
	case nil:
		return false
	case bool:
		return item
	case string:
		return item != ""
	case float64:
		return item != 0
	case int:
		return item != 0
	default:
		return true
	}
}

func (p *Provider) connectionFailure(ctx context.Context, err error, operation string) bool {
	if err == nil || ctx.Err() != nil {
		return false
	}
	var networkError net.Error
	message := strings.ToLower(err.Error())
	failed := errors.As(err, &networkError) || errors.Is(err, redisv9.ErrClosed) || strings.Contains(message, "connection refused") || strings.Contains(message, "no such host") || strings.Contains(message, "i/o timeout") || strings.Contains(message, "connection reset") || strings.Contains(message, "broken pipe") || strings.Contains(message, "command timed out") || strings.Contains(message, "maximum number of retries") || strings.Contains(message, "connection is closed")
	if failed {
		p.logger.Warn("Redis cache connection error; falling back to source data", "operation", operation, "error", err)
	}
	return failed
}

var _ domain.Provider = (*Provider)(nil)
