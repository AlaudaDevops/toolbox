/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package github

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// rateLimit tracks GitHub's primary rate-limit budget across requests.
//
// GitHub returns three relevant headers on every API response:
//
//	X-RateLimit-Remaining   — calls left in the current window.
//	X-RateLimit-Reset       — Unix epoch seconds when the window resets.
//	X-RateLimit-Limit       — total budget (we track but don't act on).
//
// On 403/429 with a Retry-After header, the *secondary* rate limit is
// in play and Retry-After is the authoritative wait — handled in
// client.go.
//
// The rate-limit logic does two things:
//
//  1. **Proactive sleep before request.** If the most recent response
//     left us below MinRemaining, we sleep until reset+1s before the
//     next call. Polite, avoids ever hitting 403.
//
//  2. **Reactive sleep on 403/429.** Honors Retry-After, capped at
//     maxBackoff to avoid pathological waits.
//
// Threshold of 50 is a default — overkill for our scale, but cheap.
type rateLimit struct {
	mu sync.Mutex

	// Last observed rate-limit state. Zero values mean "no info yet".
	remaining int
	resetAt   time.Time

	// Configuration knobs.
	minRemaining int           // sleep when remaining drops below this
	maxBackoff   time.Duration // hard cap on a single wait
}

func newRateLimit() *rateLimit {
	return &rateLimit{
		minRemaining: 50,
		maxBackoff:   10 * time.Minute,
	}
}

// observe stores the rate-limit headers from a response. Safe to call
// even when headers are absent (modernc.org-style 0-defaults).
func (r *rateLimit) observe(resp *http.Response) {
	if resp == nil {
		return
	}
	rem, ok1 := atoiHeader(resp.Header, "X-RateLimit-Remaining")
	rst, ok2 := atoiHeader(resp.Header, "X-RateLimit-Reset")
	if !ok1 && !ok2 {
		return
	}
	r.mu.Lock()
	if ok1 {
		r.remaining = rem
	}
	if ok2 {
		r.resetAt = time.Unix(int64(rst), 0)
	}
	r.mu.Unlock()
}

// waitIfNeeded blocks before the next request if the primary budget is
// near-exhausted. Returns nil when ready to proceed; ctx.Err() if the
// context expired during the wait.
//
// We add a 1s pad on top of resetAt so we don't race the window flip.
func (r *rateLimit) waitIfNeeded(ctx context.Context) error {
	r.mu.Lock()
	rem := r.remaining
	reset := r.resetAt
	r.mu.Unlock()

	if rem == 0 && reset.IsZero() {
		return nil // never observed
	}
	if rem >= r.minRemaining {
		return nil
	}
	wait := time.Until(reset) + time.Second
	if wait <= 0 {
		return nil
	}
	if wait > r.maxBackoff {
		wait = r.maxBackoff
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// retryAfter returns the wait recommended by a 403/429 response. If
// the Retry-After header is present (seconds-since-now), that wins.
// Otherwise we fall back to X-RateLimit-Reset.
//
// Returns (0, false) if the response isn't a rate-limit signal we can
// act on.
func (r *rateLimit) retryAfter(resp *http.Response) (time.Duration, bool) {
	if resp == nil {
		return 0, false
	}
	if resp.StatusCode != http.StatusTooManyRequests &&
		resp.StatusCode != http.StatusForbidden {
		return 0, false
	}
	if v := resp.Header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
			d := time.Duration(secs) * time.Second
			if d > r.maxBackoff {
				d = r.maxBackoff
			}
			return d, true
		}
	}
	if rst, ok := atoiHeader(resp.Header, "X-RateLimit-Reset"); ok {
		until := time.Until(time.Unix(int64(rst), 0)) + time.Second
		if until > 0 {
			if until > r.maxBackoff {
				until = r.maxBackoff
			}
			return until, true
		}
	}
	return 0, false
}

func atoiHeader(h http.Header, key string) (int, bool) {
	v := h.Get(key)
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}
