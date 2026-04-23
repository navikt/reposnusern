package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoRequestWithRateLimitAndOptional404ReturnsNilOn404(t *testing.T) {
	originalClient := HttpClient
	originalBackoff := RetryBackoff
	t.Cleanup(func() {
		HttpClient = originalClient
		RetryBackoff = originalBackoff
	})
	RetryBackoff = func(_ int) time.Duration { return time.Millisecond }

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprintln(w, `{"message":"not found"}`)
	}))
	defer ts.Close()

	HttpClient = ts.Client()
	ctx := context.Background()
	var result any
	err := doRequestWithRateLimitAndOptional404(ctx, "GET", ts.URL, "token", nil, &result)
	if err != nil {
		t.Fatalf("expected nil error on optional 404, got %v", err)
	}
}

func TestDoRequestWithRateLimitAndOptional404ReturnsErrorOnNon404Failure(t *testing.T) {
	originalClient := HttpClient
	originalBackoff := RetryBackoff
	t.Cleanup(func() {
		HttpClient = originalClient
		RetryBackoff = originalBackoff
	})
	RetryBackoff = func(_ int) time.Duration { return time.Millisecond }

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprintln(w, `{"message":"forbidden"}`)
	}))
	defer ts.Close()

	HttpClient = ts.Client()
	ctx := context.Background()
	var result any
	err := doRequestWithRateLimitAndOptional404(ctx, "GET", ts.URL, "token", nil, &result)
	if err == nil {
		t.Fatal("expected error on non-404 failure")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected error to mention 403, got %v", err)
	}
}
