package fetcher

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDoRequestWithRateLimit(t *testing.T) {
	// Returnerer JSON + rate limit headers
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// Simuler rate limit første gang
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprint(time.Now().Add(50*time.Millisecond).Unix()))
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`)
			return
		}

		// Andre kall er suksess
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"message": "ok"}`)
	}))
	defer ts.Close()

	// Midlertidig bytt ut httpClient
	orig := httpClient
	httpClient = ts.Client()
	defer func() { httpClient = orig }()

	type response struct {
		Message string `json:"message"`
	}
	var result response

	err := doRequestWithRateLimit("GET", ts.URL, "dummy-token", nil, &result)
	if err != nil {
		t.Fatalf("doRequestWithRateLimit failed: %v", err)
	}
	if result.Message != "ok" {
		t.Errorf("unexpected result: %+v", result)
	}
	if callCount < 2 {
		t.Errorf("expected 2 calls due to rate limit, got %d", callCount)
	}
}

func TestDoRequestWithRateLimit_POSTContentType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("unexpected Content-Type: %s", ct)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"message": "ok"}`)
	}))
	defer ts.Close()

	orig := httpClient
	httpClient = ts.Client()
	defer func() { httpClient = orig }()

	var result struct{ Message string }
	err := doRequestWithRateLimit("POST", ts.URL, "token", []byte(`{}`), &result)
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
}

func TestDoRequestWithRateLimit_RequestFail(t *testing.T) {
	var out any
	err := doRequestWithRateLimit("GET", "http://invalid-url", "token", nil, &out)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestDoRequestWithRateLimit_BadRequest(t *testing.T) {
	var result any
	err := doRequestWithRateLimit("GET", ":", "token", nil, &result)
	if err == nil {
		t.Errorf("expected error from bad request, got nil")
	}
}
