/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

// TestClient_RetriesOnRateLimit verifies that a 429 with a small
// Retry-After triggers exactly one retry, after which the second
// (200) response succeeds.
func TestClient_RetriesOnRateLimit(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "0") // sleep zero, retry immediately
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Second).Unix(), 10))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("X-RateLimit-Remaining", "5000")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "tok", srv.Client())
	// Override the maxBackoff so a misconfigured Retry-After can't hang the test.
	c.rl.maxBackoff = 2 * time.Second

	var out struct {
		OK bool `json:"ok"`
	}
	if err := c.do(context.Background(), "GET", "/whatever", nil, &out); err != nil {
		t.Fatalf("do: %v", err)
	}
	if !out.OK {
		t.Fatal("expected ok=true")
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("hits=%d, want 2 (one rate-limit + one success)", got)
	}
}

// TestClient_PreflightWaitWhenLowBudget shows that after a response
// reports Remaining < threshold, the *next* request blocks until the
// reset window, instead of marching into a 429.
func TestClient_PreflightWaitWhenLowBudget(t *testing.T) {
	var (
		firstAt  time.Time
		secondAt time.Time
		hits     int32
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		now := time.Now()
		if n == 1 {
			firstAt = now
			// Report we have 1 remaining; reset in 200 ms.
			w.Header().Set("X-RateLimit-Remaining", "1")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(now.Add(200*time.Millisecond).Unix(), 10))
		} else {
			secondAt = now
			w.Header().Set("X-RateLimit-Remaining", "5000")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(now.Add(time.Hour).Unix(), 10))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "tok", srv.Client())
	c.rl.minRemaining = 5 // any "remaining" below 5 triggers a wait

	ctx := context.Background()
	if err := c.do(ctx, "GET", "/a", nil, nil); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := c.do(ctx, "GET", "/b", nil, nil); err != nil {
		t.Fatalf("second: %v", err)
	}
	if firstAt.IsZero() || secondAt.IsZero() {
		t.Fatal("missing timestamps")
	}
	gap := secondAt.Sub(firstAt)
	if gap < 200*time.Millisecond {
		t.Fatalf("expected pre-flight sleep of at least 200ms, got %s", gap)
	}
}

// TestRetryAfterRespectsMaxBackoff prevents a misconfigured server
// (Retry-After: 99999) from hanging the client indefinitely.
func TestRetryAfterRespectsMaxBackoff(t *testing.T) {
	rl := newRateLimit()
	rl.maxBackoff = time.Second

	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Retry-After": []string{"99999"}},
	}
	d, ok := rl.retryAfter(resp)
	if !ok {
		t.Fatal("expected rate-limit signal")
	}
	if d > rl.maxBackoff {
		t.Fatalf("wait %s exceeds maxBackoff %s", d, rl.maxBackoff)
	}
}
