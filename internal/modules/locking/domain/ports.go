package domain

import (
	"context"
	"time"
)

type LockOptions struct {
	OwnerID    string
	Expire     time.Duration
	AwaitQueue bool
}
type Provider interface {
	Execute(context.Context, []string, time.Duration, func(context.Context) error) error
	Acquire(context.Context, []string, LockOptions) error
	Release(context.Context, []string, string) (bool, error)
	ReleaseAll(context.Context, *string) error
}
