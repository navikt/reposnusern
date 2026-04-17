package fetcher

import (
	"context"
	"sync"
	"time"
)

type RateLimitResource string

const (
	RateLimitResourceCore    RateLimitResource = "core"
	RateLimitResourceGraphQL RateLimitResource = "graphql"
)

// RateLimitStats summarizes how often a resource was blocked and waited on.
type RateLimitStats struct {
	Hits      int64
	Waits     int64
	TotalWait time.Duration
}

type rateLimitState struct {
	blockedUntil time.Time
	hits         int64
	waits        int64
	totalWait    time.Duration
}

type ResourceRateLimiter struct {
	mu     sync.Mutex
	states map[RateLimitResource]*rateLimitState
}

// SharedRateLimiter coordinates cooldowns across all fetcher requests.
var SharedRateLimiter = NewResourceRateLimiter()

// NewResourceRateLimiter creates limiter state for the GitHub resources we track.
func NewResourceRateLimiter() *ResourceRateLimiter {
	return &ResourceRateLimiter{
		states: map[RateLimitResource]*rateLimitState{
			RateLimitResourceCore:    {},
			RateLimitResourceGraphQL: {},
		},
	}
}

// Wait blocks until the selected resource is no longer in cooldown.
func (l *ResourceRateLimiter) Wait(ctx context.Context, resource RateLimitResource) error {
	for {
		l.mu.Lock()
		state := l.state(resource)
		wait := time.Until(state.blockedUntil)
		if wait <= 0 {
			l.mu.Unlock()
			return nil
		}
		state.waits++
		l.mu.Unlock()

		if err := sleepWithContext(ctx, wait); err != nil {
			return err
		}
	}
}

// BlockFor marks a resource as unavailable for the next wait duration.
// It returns true when this call started a new active cooldown window.
func (l *ResourceRateLimiter) BlockFor(resource RateLimitResource, wait time.Duration) bool {
	if wait <= 0 {
		return false
	}
	return l.BlockUntil(resource, time.Now().Add(wait))
}

// BlockUntil extends a resource cooldown if the new deadline is later.
// It returns true when this call started a new active cooldown window.
func (l *ResourceRateLimiter) BlockUntil(resource RateLimitResource, until time.Time) bool {
	if until.IsZero() {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	state := l.state(resource)
	now := time.Now()
	startedNewBlock := !state.blockedUntil.After(now) && until.After(now)
	if until.After(state.blockedUntil) {
		start := now
		if state.blockedUntil.After(start) {
			start = state.blockedUntil
		}
		state.blockedUntil = until
		state.totalWait += until.Sub(start)
	}
	state.hits++
	return startedNewBlock
}

// Stats returns a snapshot of the accumulated limiter counters per resource.
func (l *ResourceRateLimiter) Stats() map[RateLimitResource]RateLimitStats {
	l.mu.Lock()
	defer l.mu.Unlock()

	stats := make(map[RateLimitResource]RateLimitStats, len(l.states))
	for resource, state := range l.states {
		stats[resource] = RateLimitStats{
			Hits:      state.hits,
			Waits:     state.waits,
			TotalWait: state.totalWait,
		}
	}
	return stats
}

// Reset clears all tracked cooldowns and counters.
func (l *ResourceRateLimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.states = map[RateLimitResource]*rateLimitState{
		RateLimitResourceCore:    {},
		RateLimitResourceGraphQL: {},
	}
}

func (l *ResourceRateLimiter) state(resource RateLimitResource) *rateLimitState {
	state, ok := l.states[resource]
	if !ok {
		state = &rateLimitState{}
		l.states[resource] = state
	}
	return state
}

// GetRateLimitStats returns the shared limiter stats used by the runner summary.
func GetRateLimitStats() map[RateLimitResource]RateLimitStats {
	return SharedRateLimiter.Stats()
}

// ResetRateLimitStats clears shared limiter state before a new run starts.
func ResetRateLimitStats() {
	SharedRateLimiter.Reset()
}
