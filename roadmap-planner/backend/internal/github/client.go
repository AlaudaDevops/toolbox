/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

// Package github is a minimal, stdlib-only client for the slice of the
// GitHub REST API we need for team analytics.
//
// Why not google/go-github?
//   - We need only two endpoints (PR list, reviews list). Pulling in the
//     full SDK adds ~300 transitive dependencies for a 600-line wrapper.
//   - Stdlib gives us first-class testability via http.RoundTripper
//     injection without an SDK-specific test client.
//
// If we ever start needing labels, files, statuses, etc., revisit this
// decision; until then the surface stays narrow on purpose.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client is a tiny GitHub REST client.
//
// Auth is pluggable via TokenSource — a static PAT or a refreshing
// GitHub-App installation token both fit. Rate-limit handling is
// always-on:
//
//   - Pre-flight: if the previous response left us below MinRemaining,
//     we sleep until the reset window (capped at maxBackoff).
//   - On 403/429: we honor Retry-After (or X-RateLimit-Reset) and retry
//     once. After that we surface the error to the caller.
type Client struct {
	baseURL     string
	tokenSource TokenSource
	http        *http.Client
	rl          *rateLimit
}

// New builds a client with a static PAT (or empty for unauthenticated
// access — only useful against public read-only routes). For GitHub
// App auth, use NewWithTokenSource.
//
// baseURL defaults to api.github.com; pass
// https://github.example.com/api/v3 for GHES.
func New(baseURL, token string, httpClient *http.Client) *Client {
	return NewWithTokenSource(baseURL, StaticTokenSource{Value: token}, httpClient)
}

// NewWithTokenSource builds a client backed by an arbitrary TokenSource
// — typically a *AppTokenSource minted via NewAppTokenSource.
func NewWithTokenSource(baseURL string, ts TokenSource, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if ts == nil {
		ts = StaticTokenSource{}
	}
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		tokenSource: ts,
		http:        httpClient,
		rl:          newRateLimit(),
	}
}

// PullRequest is the subset of the PR resource we actually persist.
// Field tags mirror GitHub's JSON; helpers below normalise into our
// storage shape.
type PullRequest struct {
	Number       int       `json:"number"`
	Title        string    `json:"title"`
	State        string    `json:"state"` // "open" | "closed"
	Draft        bool      `json:"draft"`
	HTMLURL      string    `json:"html_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ClosedAt     *time.Time `json:"closed_at"`
	MergedAt     *time.Time `json:"merged_at"`
	Additions    int       `json:"additions"`
	Deletions    int       `json:"deletions"`
	ChangedFiles int       `json:"changed_files"`
	User         struct {
		Login string `json:"login"`
	} `json:"user"`
	Head struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

// Review is the subset of the review resource we persist.
type Review struct {
	ID          int64     `json:"id"`
	State       string    `json:"state"` // "APPROVED" | "CHANGES_REQUESTED" | "COMMENTED" | "DISMISSED"
	SubmittedAt time.Time `json:"submitted_at"`
	User        struct {
		Login string `json:"login"`
	} `json:"user"`
}

// ListPullRequestsOptions controls the PR listing query.
type ListPullRequestsOptions struct {
	State     string // "open" | "closed" | "all" (default "all")
	Sort      string // "created" | "updated" | "popularity" | "long-running"
	Direction string // "asc" | "desc"
	Since     time.Time
	PerPage   int
	MaxPages  int // 0 = unlimited
}

// ListPullRequests returns PRs for a repo, paginating through GitHub's
// REST. We do not stream — the typical batch is a few hundred PRs and
// holding them in memory is harmless.
//
// `Since` is implemented client-side (the PR list endpoint does not
// support `since`). We sort by `updated desc` and stop early once we see
// a record older than Since.
func (c *Client) ListPullRequests(ctx context.Context, owner, repo string, opts ListPullRequestsOptions) ([]PullRequest, error) {
	if opts.State == "" {
		opts.State = "all"
	}
	if opts.Sort == "" {
		opts.Sort = "updated"
	}
	if opts.Direction == "" {
		opts.Direction = "desc"
	}
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}

	var out []PullRequest
	for page := 1; opts.MaxPages == 0 || page <= opts.MaxPages; page++ {
		q := url.Values{}
		q.Set("state", opts.State)
		q.Set("sort", opts.Sort)
		q.Set("direction", opts.Direction)
		q.Set("per_page", strconv.Itoa(opts.PerPage))
		q.Set("page", strconv.Itoa(page))

		path := fmt.Sprintf("/repos/%s/%s/pulls?%s", owner, repo, q.Encode())
		var page []PullRequest
		if err := c.do(ctx, "GET", path, nil, &page); err != nil {
			return nil, fmt.Errorf("list PRs %s/%s page %d: %w", owner, repo, len(out)/opts.PerPage+1, err)
		}
		if len(page) == 0 {
			break
		}
		stop := false
		for _, p := range page {
			if !opts.Since.IsZero() && p.UpdatedAt.Before(opts.Since) {
				stop = true
				break
			}
			out = append(out, p)
		}
		if stop || len(page) < opts.PerPage {
			break
		}
	}
	return out, nil
}

// ListReviews returns the reviews on one PR.
func (c *Client) ListReviews(ctx context.Context, owner, repo string, number int) ([]Review, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews?per_page=100", owner, repo, number)
	var out []Review
	if err := c.do(ctx, "GET", path, nil, &out); err != nil {
		return nil, fmt.Errorf("list reviews %s/%s#%d: %w", owner, repo, number, err)
	}
	return out, nil
}

// do is the request engine. It:
//
//   1. Sleeps before issuing if the rate-limit budget is near-exhausted.
//   2. Resolves a token from the configured TokenSource (PAT or App).
//   3. On 403/429 with a Retry-After hint, sleeps and retries ONCE.
//      After that, surfaces the error so the caller can decide.
//
// JSON decoding only runs on 2xx with a non-nil out.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader, out interface{}) error {
	const maxAttempts = 2
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := c.rl.waitIfNeeded(ctx); err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		token, err := c.tokenSource.Token(ctx)
		if err != nil {
			return fmt.Errorf("github auth: %w", err)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		c.rl.observe(resp)

		// Rate-limited? Decide whether we'll retry.
		if wait, ok := c.rl.retryAfter(resp); ok && attempt < maxAttempts-1 {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			t := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			case <-t.C:
			}
			continue
		}

		// Non-rate-limit error.
		if resp.StatusCode >= 300 {
			raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			resp.Body.Close()
			return fmt.Errorf("github %s %s: %s — %s", method, path, resp.Status, strings.TrimSpace(string(raw)))
		}

		// Success.
		if out == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return nil
		}
		err = json.NewDecoder(resp.Body).Decode(out)
		resp.Body.Close()
		return err
	}
	return fmt.Errorf("github %s %s: rate-limited beyond retry budget", method, path)
}
