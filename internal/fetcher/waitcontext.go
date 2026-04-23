package fetcher

import (
	"context"
	"errors"
)

type waitInterruptContextKey struct{}

var ErrWaitInterrupted = errors.New("wait interrupted by shutdown")

// WithWaitInterrupt attaches a context that should interrupt retry/rate-limit
// sleeps without cancelling the main request context.
func WithWaitInterrupt(ctx context.Context, interruptCtx context.Context) context.Context {
	if interruptCtx == nil {
		return ctx
	}
	return context.WithValue(ctx, waitInterruptContextKey{}, interruptCtx)
}

func waitInterruptContext(ctx context.Context) context.Context {
	if ctx == nil {
		return nil
	}
	interruptCtx, _ := ctx.Value(waitInterruptContextKey{}).(context.Context)
	return interruptCtx
}
