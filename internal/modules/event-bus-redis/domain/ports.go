package domain

import (
	"context"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/event-bus-redis/domain/models"
)

// Queue represents the BullMQ boundary. The host supplies the durable worker
// queue adapter so the Redis event bus never degrades to in-process delivery.
type Queue interface {
	AddBulk(context.Context, []models.Job) error
}

// StagingRedis exposes the Redis list operations used for grouped event staging.
type StagingRedis interface {
	RPush(context.Context, string, ...string) error
	Expire(context.Context, string, time.Duration) error
	LRange(context.Context, string, int64, int64) ([]string, error)
	PTTL(context.Context, string) (time.Duration, error)
	Del(context.Context, string) error
	Unlink(context.Context, string) error
}

type Interceptor interface {
	BeforeEmit(context.Context, models.Event, bool, string)
}
type Logger interface{ Warn(string, ...any) }

type Subscriber func(context.Context, models.Event) error
