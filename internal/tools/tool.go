package tools

import (
	"context"
	"time"
)

type Tool interface {
	Name() string
	Timeout() time.Duration
	Execute(ctx context.Context, input any) (any, error)
}
