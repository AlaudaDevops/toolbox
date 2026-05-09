/*
Copyright 2026 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package gitlab

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// rateLimit tracks GitLab's per-token rate-limit budget. The headers
// GitLab returns are RateLimit-Remaining / RateLimit-Reset / RateLimit-Limit
// (no `X-` prefix; that's an old GitHub convention). Same shape as the
// github package's helper otherwise.
//
// `Retry-After` is honoured on 429 (GitLab's rate-limit response); on
// 403 we also try to recover but cap the wait at maxBackoff so a
// misconfigured token doesn't make us stall forever.
type rateLimit struct {
	mu sync.Mutex

	remaining int
	resetAt   time.Time

	minRemaining int
	maxBackoff   time.Duration
}

func newRateLimit() *rateLimit {
	return &rateLimit{
		minRemaining: 30,
		maxBackoff:   5 * time.Minute,
	}
}

func (r *rateLimit) observe(resp *http.Response) {
	if resp == nil {
		return
	}
	rem, ok1 := atoiHeader(resp.Header, "RateLimit-Remaining")
	rst, ok2 := atoiHeader(resp.Header, "RateLimit-Reset")
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

func (r *rateLimit) waitIfNeeded(ctx context.Context) error {
	r.mu.Lock()
	rem := r.remaining
	reset := r.resetAt
	r.mu.Unlock()

	if rem == 0 && reset.IsZero() {
		return nil
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
	if rst, ok := atoiHeader(resp.Header, "RateLimit-Reset"); ok {
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
