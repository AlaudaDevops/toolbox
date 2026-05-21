/*
Copyright 2026 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

// Package gitlab is a minimal, stdlib-only client for the slice of the
// GitLab REST API we need for team analytics — list projects under a
// group (with optional recursion), list merge requests, list MR notes
// (used as the prow-style /lgtm review surrogate), and search users by
// email (used for one-shot identity prefills).
//
// Why not xanzy/go-gitlab? Same reasoning as the github package: we need
// four endpoints, the SDK pulls in a forest of dependencies, and stdlib
// gives us http.RoundTripper-based testing without an SDK shim.
package gitlab

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

// Client is the GitLab REST client. Auth is a static PAT; rotate via
// config + restart. The PAT scope we need is read_api over the projects
// the operator wants tracked — typically one developer-account PAT
// scoped to the relevant top-level groups.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
	rl      *rateLimit
}

// New builds a client. baseURL example: "https://gitlab-ce.alauda.cn".
// Trailing slashes are trimmed; the package always appends "/api/v4/...".
func New(baseURL, token string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    httpClient,
		rl:      newRateLimit(),
	}
}

// Project is the slim subset we use for sync.
type Project struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
	Archived          bool   `json:"archived"`
	DefaultBranch     string `json:"default_branch"`
}

// MergeRequest mirrors what we persist into the shared pull_requests table.
type MergeRequest struct {
	IID          int        `json:"iid"`
	ProjectID    int64      `json:"project_id"`
	Title        string     `json:"title"`
	State        string     `json:"state"` // "opened" | "closed" | "merged" | "locked"
	WebURL       string     `json:"web_url"`
	Draft        bool       `json:"draft"`
	WorkInProg   bool       `json:"work_in_progress"`
	SourceBranch string     `json:"source_branch"`
	TargetBranch string     `json:"target_branch"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	MergedAt     *time.Time `json:"merged_at"`
	ClosedAt     *time.Time `json:"closed_at"`
	Author       struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	} `json:"author"`
	// References.Full is the GitLab-canonical "<group>/<sub>/<proj>!<iid>"
	// pointer. We rely on it in the instance-wide MR sweep to recover
	// the project path-with-namespace without a per-MR show call.
	References struct {
		Full string `json:"full"`
	} `json:"references"`
	// Diff stats are only populated when we explicitly request them on
	// the show endpoint; the list endpoint omits them. We hydrate a
	// follow-up call only for merged MRs (the few that drive the
	// dashboard).
	Diff struct {
		Additions    int `json:"additions"`
		Deletions    int `json:"deletions"`
		ChangedFiles int `json:"changes_count_int"`
	} `json:"-"`
}

// ProjectPath returns the path-with-namespace ("group/sub/proj") for an
// MR returned by the instance-wide endpoint. It first tries the
// References.Full pointer ("group/sub/proj!iid"), then falls back to
// parsing the web_url. Returns the empty string if neither path works.
func (m MergeRequest) ProjectPath() string {
	if full := m.References.Full; full != "" {
		if idx := strings.Index(full, "!"); idx > 0 {
			return full[:idx]
		}
	}
	// web_url is of the form
	// "https://gitlab.example.com/<group>/<sub>/<proj>/-/merge_requests/<iid>".
	// Strip the scheme+host, then trim "/-/merge_requests/...".
	if u := m.WebURL; u != "" {
		// Skip past "scheme://host/".
		const sep = "://"
		if i := strings.Index(u, sep); i >= 0 {
			rest := u[i+len(sep):]
			if j := strings.Index(rest, "/"); j >= 0 {
				rest = rest[j+1:]
				if k := strings.Index(rest, "/-/merge_requests/"); k > 0 {
					return rest[:k]
				}
			}
		}
	}
	return ""
}

// Note is one MR comment. We only persist non-system non-author notes;
// the classifier in sync.go derives review state from Body.
type Note struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	System    bool      `json:"system"`
	CreatedAt time.Time `json:"created_at"`
	Author    struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	} `json:"author"`
}

// User is the search result shape. Only used for the optional
// email-based gitlab_username prefill.
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	State    string `json:"state"`
}

// ListGroupProjectsOptions controls the projects-under-group listing.
type ListGroupProjectsOptions struct {
	IncludeSubgroups bool
	IncludeArchived  bool
	PerPage          int
	MaxPages         int // 0 = unlimited
}

// ListGroupProjects pages through GET /groups/:id/projects.
//
// The group identifier is URL-path-escaped — `devops/edge` works as the
// path lookup. For very large groups bump PerPage to 100 (default).
//
// `with_shared=false` is enforced server-side: GitLab's default is
// `true`, which makes the endpoint return every shared project the
// group has access to. That was the root cause of the W1/B1 audit
// finding — prod's gitlab.groups=["devops/**"] pulled in
// `container-platform/*`, `ops/*` etc. and credited their authors as
// devops members. We never want that fan-out for ingest; per-member
// instance-wide coverage now lives in ListInstanceMergeRequests.
func (c *Client) ListGroupProjects(ctx context.Context, group string, opts ListGroupProjectsOptions) ([]Project, error) {
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	out := make([]Project, 0, 64)
	for page := 1; opts.MaxPages == 0 || page <= opts.MaxPages; page++ {
		q := url.Values{}
		q.Set("per_page", strconv.Itoa(opts.PerPage))
		q.Set("page", strconv.Itoa(page))
		q.Set("include_subgroups", strconv.FormatBool(opts.IncludeSubgroups))
		q.Set("with_shared", "false")
		if !opts.IncludeArchived {
			q.Set("archived", "false")
		}
		path := fmt.Sprintf("/api/v4/groups/%s/projects?%s", url.PathEscape(group), q.Encode())
		var batch []Project
		if err := c.do(ctx, "GET", path, nil, &batch); err != nil {
			return nil, fmt.Errorf("list group %s projects page %d: %w", group, page, err)
		}
		out = append(out, batch...)
		if len(batch) < opts.PerPage {
			break
		}
	}
	return out, nil
}

// GetProject resolves a single project by `group/sub/proj` path.
func (c *Client) GetProject(ctx context.Context, fullPath string) (*Project, error) {
	path := fmt.Sprintf("/api/v4/projects/%s", url.PathEscape(fullPath))
	var p Project
	if err := c.do(ctx, "GET", path, nil, &p); err != nil {
		return nil, fmt.Errorf("get project %s: %w", fullPath, err)
	}
	return &p, nil
}

// ListMergeRequestsOptions controls the MR listing query.
type ListMergeRequestsOptions struct {
	State        string // "opened" | "closed" | "merged" | "all" (default "all")
	UpdatedAfter time.Time
	OrderBy      string // "updated_at" (default)
	Sort         string // "asc" | "desc" (default "desc")
	PerPage      int
	MaxPages     int // 0 = unlimited
}

// ListMergeRequests returns MRs for one project (by ID), paginating
// until either MaxPages or a short page is hit.
//
// `updated_after` is server-side here (unlike GitHub's /pulls) so we
// don't need the client-side stop-early dance.
func (c *Client) ListMergeRequests(ctx context.Context, projectID int64, opts ListMergeRequestsOptions) ([]MergeRequest, error) {
	if opts.State == "" {
		opts.State = "all"
	}
	if opts.OrderBy == "" {
		opts.OrderBy = "updated_at"
	}
	if opts.Sort == "" {
		opts.Sort = "desc"
	}
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	var out []MergeRequest
	for page := 1; opts.MaxPages == 0 || page <= opts.MaxPages; page++ {
		q := url.Values{}
		q.Set("state", opts.State)
		q.Set("order_by", opts.OrderBy)
		q.Set("sort", opts.Sort)
		q.Set("per_page", strconv.Itoa(opts.PerPage))
		q.Set("page", strconv.Itoa(page))
		if !opts.UpdatedAfter.IsZero() {
			q.Set("updated_after", opts.UpdatedAfter.UTC().Format(time.RFC3339))
		}
		path := fmt.Sprintf("/api/v4/projects/%d/merge_requests?%s", projectID, q.Encode())
		var batch []MergeRequest
		if err := c.do(ctx, "GET", path, nil, &batch); err != nil {
			return nil, fmt.Errorf("list MRs project %d page %d: %w", projectID, page, err)
		}
		out = append(out, batch...)
		if len(batch) < opts.PerPage {
			break
		}
	}
	return out, nil
}

// ListInstanceMergeRequestsOptions controls the instance-wide MR sweep
// (the per-member Pass B in W1). The endpoint backing this is
// `GET /api/v4/merge_requests?scope=all&author_username=<u>`.
//
// `with_shared` doesn't apply at this endpoint — GitLab always scopes
// the result by what the auth token can see. For our bot user with
// `read_api` over the alauda devops members, that's the full instance
// up to project visibility limits.
type ListInstanceMergeRequestsOptions struct {
	AuthorUsername string
	State          string // "opened" | "closed" | "merged" | "all" (default "all")
	UpdatedAfter   time.Time
	OrderBy        string // "updated_at" (default)
	Sort           string // "asc" | "desc" (default "desc")
	PerPage        int
	MaxPages       int // 0 = unlimited
}

// ListInstanceMergeRequests returns every MR the token can see that
// matches the supplied filter. We page until either MaxPages or a
// short page is hit. Pagination semantics mirror ListMergeRequests; the
// endpoint accepts the same `updated_after`/`order_by`/`sort`/`state`
// query knobs server-side.
//
// Author scoping is required — the unscoped endpoint returns the
// caller's MRs only, which isn't what Pass B needs. Pass the configured
// member's gitlab username here.
func (c *Client) ListInstanceMergeRequests(ctx context.Context, opts ListInstanceMergeRequestsOptions) ([]MergeRequest, error) {
	if opts.AuthorUsername == "" {
		return nil, fmt.Errorf("ListInstanceMergeRequests: AuthorUsername required")
	}
	if opts.State == "" {
		opts.State = "all"
	}
	if opts.OrderBy == "" {
		opts.OrderBy = "updated_at"
	}
	if opts.Sort == "" {
		opts.Sort = "desc"
	}
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	var out []MergeRequest
	for page := 1; opts.MaxPages == 0 || page <= opts.MaxPages; page++ {
		q := url.Values{}
		q.Set("scope", "all")
		q.Set("author_username", opts.AuthorUsername)
		q.Set("state", opts.State)
		q.Set("order_by", opts.OrderBy)
		q.Set("sort", opts.Sort)
		q.Set("per_page", strconv.Itoa(opts.PerPage))
		q.Set("page", strconv.Itoa(page))
		if !opts.UpdatedAfter.IsZero() {
			q.Set("updated_after", opts.UpdatedAfter.UTC().Format(time.RFC3339))
		}
		path := "/api/v4/merge_requests?" + q.Encode()
		var batch []MergeRequest
		if err := c.do(ctx, "GET", path, nil, &batch); err != nil {
			return nil, fmt.Errorf("list instance MRs author=%s page %d: %w", opts.AuthorUsername, page, err)
		}
		out = append(out, batch...)
		if len(batch) < opts.PerPage {
			break
		}
	}
	return out, nil
}

// GetMergeRequest fetches one MR with diff stats — the list endpoint
// omits additions/deletions/changes_count, so we hydrate per-MR for
// the merged ones we care about.
func (c *Client) GetMergeRequest(ctx context.Context, projectID int64, iid int) (*MergeRequest, error) {
	path := fmt.Sprintf("/api/v4/projects/%d/merge_requests/%d", projectID, iid)
	// Custom decode because Diff fields land at top-level on the show
	// endpoint, not nested.
	var raw struct {
		MergeRequest
		Additions    int    `json:"additions"`
		Deletions    int    `json:"deletions"`
		ChangesCount string `json:"changes_count"` // "27" — a string for some reason
	}
	if err := c.do(ctx, "GET", path, nil, &raw); err != nil {
		return nil, fmt.Errorf("get MR %d!%d: %w", projectID, iid, err)
	}
	mr := raw.MergeRequest
	mr.Diff.Additions = raw.Additions
	mr.Diff.Deletions = raw.Deletions
	if n, err := strconv.Atoi(raw.ChangesCount); err == nil {
		mr.Diff.ChangedFiles = n
	}
	return &mr, nil
}

// ListMRNotes returns all notes on one MR. We always fetch the full
// thread because the prow `/lgtm` we look for can be on any comment.
func (c *Client) ListMRNotes(ctx context.Context, projectID int64, iid int) ([]Note, error) {
	out := make([]Note, 0, 16)
	const perPage = 100
	for page := 1; ; page++ {
		path := fmt.Sprintf("/api/v4/projects/%d/merge_requests/%d/notes?per_page=%d&page=%d&sort=asc",
			projectID, iid, perPage, page)
		var batch []Note
		if err := c.do(ctx, "GET", path, nil, &batch); err != nil {
			return nil, fmt.Errorf("list notes %d!%d page %d: %w", projectID, iid, page, err)
		}
		out = append(out, batch...)
		if len(batch) < perPage {
			break
		}
	}
	return out, nil
}

// MRCommit is the slice of the MR-commits resource we use for Lead Time.
// We only need the earliest commit's authored_date (see Lead Time
// calculator), so the struct stays narrow on purpose.
type MRCommit struct {
	SHA          string    `json:"id"`
	AuthoredDate time.Time `json:"authored_date"`
}

// ListMRCommits returns the commits associated with one MR. Mirrors the
// GitHub client's ListPRCommits: we read one page (up to 100 commits)
// and let the caller pick min(authored_date) for
// `pull_requests.first_commit_at` in the DORA Lead Time calculator.
//
// GitLab's default per_page is 20 — override matters.
func (c *Client) ListMRCommits(ctx context.Context, projectID int64, iid int) ([]MRCommit, error) {
	path := fmt.Sprintf("/api/v4/projects/%d/merge_requests/%d/commits?per_page=100", projectID, iid)
	var out []MRCommit
	if err := c.do(ctx, "GET", path, nil, &out); err != nil {
		return nil, fmt.Errorf("list MR commits %d!%d: %w", projectID, iid, err)
	}
	return out, nil
}

// SearchUsers does a /users?search=<email> lookup. GitLab returns an
// array even for an exact-email match, so callers should pick the entry
// whose email matches case-insensitively (when their token has the
// admin scope; otherwise the email field is omitted on the response and
// callers fall back to the first 'active' result).
func (c *Client) SearchUsers(ctx context.Context, query string) ([]User, error) {
	q := url.Values{}
	q.Set("search", query)
	q.Set("per_page", "20")
	path := "/api/v4/users?" + q.Encode()
	var out []User
	if err := c.do(ctx, "GET", path, nil, &out); err != nil {
		return nil, fmt.Errorf("search users %q: %w", query, err)
	}
	return out, nil
}

// do is the request engine. Same shape as the github client's:
//  1. wait if the rate-limit budget is near-exhausted
//  2. attach PAT
//  3. honour Retry-After once on 429
//  4. surface non-2xx with a body excerpt
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
		req.Header.Set("Accept", "application/json")
		if c.token != "" {
			req.Header.Set("PRIVATE-TOKEN", c.token)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		c.rl.observe(resp)

		if wait, ok := c.rl.retryAfter(resp); ok && attempt < maxAttempts-1 {
			_, _ = io.Copy(io.Discard, resp.Body)
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
		if resp.StatusCode >= 300 {
			raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			resp.Body.Close()
			return fmt.Errorf("gitlab %s %s: %s — %s", method, path, resp.Status, strings.TrimSpace(string(raw)))
		}
		if out == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return nil
		}
		err = json.NewDecoder(resp.Body).Decode(out)
		resp.Body.Close()
		return err
	}
	return fmt.Errorf("gitlab %s %s: rate-limited beyond retry budget", method, path)
}
