package domain

import "context"

type Cache interface {
	Get(context.Context, string) (any, bool, error)
	Set(context.Context, string, any, int) error
	Invalidate(context.Context, string) error
	Clear(context.Context) error
}
