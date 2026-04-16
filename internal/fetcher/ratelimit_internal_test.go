package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestResourceRateLimiterBlocksOnlyMatchingResource(t *testing.T) {
	limiter := NewResourceRateLimiter()
	limiter.BlockFor(RateLimitResourceGraphQL, 40*time.Millisecond)

	coreStart := time.Now()
	if err := limiter.Wait(context.Background(), RateLimitResourceCore); err != nil {
		t.Fatalf("core wait returned error: %v", err)
	}
	if elapsed := time.Since(coreStart); elapsed >= 20*time.Millisecond {
		t.Fatalf("core wait took too long: %s", elapsed)
	}

	graphQLStart := time.Now()
	if err := limiter.Wait(context.Background(), RateLimitResourceGraphQL); err != nil {
		t.Fatalf("graphql wait returned error: %v", err)
	}
	if elapsed := time.Since(graphQLStart); elapsed < 30*time.Millisecond {
		t.Fatalf("graphql wait was too short: %s", elapsed)
	}
}

func TestRateLimitWaitPrefersRetryAfter(t *testing.T) {
	headers := http.Header{}
	headers.Set("Retry-After", "1")
	headers.Set("X-RateLimit-Remaining", "0")
	headers.Set("X-RateLimit-Reset", fmt.Sprint(time.Now().Add(time.Minute).Unix()))

	wait, ok := rateLimitWait(headers, http.StatusTooManyRequests)
	if !ok {
		t.Fatal("expected rate-limit wait to be detected")
	}
	if wait < time.Second {
		t.Fatalf("expected wait >= 1s, got %s", wait)
	}
	if wait >= 2*time.Second {
		t.Fatalf("expected wait < 2s, got %s", wait)
	}
}
