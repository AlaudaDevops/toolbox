/*
Copyright 2026 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package gitlab

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
	"go.uber.org/zap"
)

// Linker resolves an MR to a Jira epic key. Mirrors the github.Linker
// interface but operates on a GitLab MergeRequest.
type Linker interface {
	Link(mr MergeRequest) string
}

// DefaultLinker matches against the source branch and the MR title.
//
// Compiled regexes are memoised by project-key so repeated DefaultLinker
// calls (e.g. across tests) skip the recompilation. The cache is small —
// one entry per Jira project key the process touches in its lifetime.
var (
	defaultLinkerCache = map[string]*defaultLinker{}
	defaultLinkerMu    sync.Mutex
)

func DefaultLinker(projectKey string) Linker {
	key := strings.ToUpper(projectKey)
	defaultLinkerMu.Lock()
	defer defaultLinkerMu.Unlock()
	if l, ok := defaultLinkerCache[key]; ok {
		return l
	}
	l := &defaultLinker{
		key: key,
		re:  regexp.MustCompile(`(?i)\b(` + regexp.QuoteMeta(projectKey) + `-\d+)\b`),
	}
	defaultLinkerCache[key] = l
	return l
}

type defaultLinker struct {
	key string
	re  *regexp.Regexp
}

func (d *defaultLinker) Link(mr MergeRequest) string {
	for _, candidate := range []string{mr.SourceBranch, mr.Title} {
		if m := d.re.FindString(candidate); m != "" {
			return strings.ToUpper(m)
		}
	}
	return ""
}

// Syncer pulls MRs and notes-as-reviews into the storage.Store. Mirrors
// the github.Syncer shape: one process-wide instance, run on a ticker.
//
// Reviews from notes:
//
//	GitLab has no first-class review event. We treat any non-system,
//	non-author MR note as a review touch. Bodies that match
//	approvalRegex (`/lgtm` on its own line) are stored as state="approved";
//	anything else as state="commented". Procedural prow commands
//	(/retest, /hold, /cherry-pick, /uncc, /assign, /unassign, /label,
//	/milestone, /retitle, /priority, /kind, /area, /sig) are skipped
//	entirely — they're noise on the dashboard.
type Syncer struct {
	client       *Client
	store        storage.Store
	specs        []GroupSpec
	linker       Linker
	logger       *zap.Logger
	backfillDays int

	WildcardTTL     time.Duration
	IncludeArchived bool
	// HydrateDiff controls whether merged MRs get a follow-up show-MR
	// call to populate additions/deletions/changed_files. Default true
	// since W6 (2026-05-19) — the data is cheap (~1 extra call per
	// merged MR, ~+2% of the per-cycle API budget) and the Dashboard
	// tab consumes it. Operators on tight rate budgets can disable via
	// `gitlab.hydrate_diff: false`.
	HydrateDiff bool

	wildcardCache *wildcardCache
	nowFn         func() time.Time
}

// NewSyncer builds a Syncer. backfillDays sets the first-run window;
// pass 0 to use the package default (180).
func NewSyncer(client *Client, store storage.Store, specs []GroupSpec, linker Linker, backfillDays int) *Syncer {
	if backfillDays <= 0 {
		backfillDays = 180
	}
	return &Syncer{
		client:        client,
		store:         store,
		specs:         specs,
		linker:        linker,
		logger:        logger.WithComponent("gitlab-syncer"),
		backfillDays:  backfillDays,
		WildcardTTL:   DefaultWildcardTTL,
		HydrateDiff:   true,
		wildcardCache: newWildcardCache(),
		nowFn:         time.Now,
	}
}

// approvalRegex matches a /lgtm body that's the entire line (allows
// trailing whitespace). Some users write "/lgtm cancel" — we don't treat
// that as approval.
var approvalRegex = regexp.MustCompile(`(?m)^\s*/lgtm\s*$`)

// procedureRegex matches prow commands we want to drop entirely (so
// they don't inflate the "PRs reviewed" count). The list mirrors what
// alauda's prow-style flow uses.
var procedureRegex = regexp.MustCompile(`(?m)^\s*/(retest|hold|cherry-pick|uncc|cc|assign|unassign|label|remove-label|milestone|retitle|priority|kind|area|sig|approve|close|reopen|wip)(\b.*)?$`)

// classifyNote returns the review state to store for a non-system note,
// or ("", false) if the note should be skipped entirely.
//
// Rules (in priority order):
//   - body contains an `/lgtm` line → "approved"
//   - body is purely a procedural prow command → skip
//   - empty/whitespace body → skip
//   - otherwise → "commented"
func classifyNote(body string) (string, bool) {
	if approvalRegex.MatchString(body) {
		return "approved", true
	}
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "", false
	}
	// A note that's exclusively prow commands (one or more lines, no
	// substantive content) is procedural. We approximate "exclusively"
	// by stripping all matching procedure lines and checking what's
	// left; if there's still text after the strip, treat it as a
	// regular comment.
	stripped := procedureRegex.ReplaceAllString(body, "")
	if strings.TrimSpace(stripped) == "" {
		return "", false
	}
	return "commented", true
}

// Sync runs one cycle: expand specs → list MRs updated since the
// previous run → upsert PRs + notes-as-reviews → write a collection_run.
//
// The shape mirrors github.Syncer.Sync — see those comments for first-
// run vs. incremental behaviour.
func (s *Syncer) Sync(ctx context.Context) error {
	if len(s.specs) == 0 {
		return nil
	}

	now := s.nowFn()
	resolved, resolveErr := s.resolveProjects(ctx, now)
	if resolveErr != nil {
		s.logger.Warn("group spec resolution failed (partial)", zap.Error(resolveErr))
	}
	if len(resolved) == 0 {
		if resolveErr != nil {
			return resolveErr
		}
		return nil
	}

	since := time.Now().AddDate(0, 0, -s.backfillDays)
	mode := "backfill"
	if last, err := s.store.LatestCollectionRun(ctx, "gitlab"); err == nil && last != nil {
		since = last.CapturedAt.Add(-1 * time.Hour)
		mode = "incremental"
	}
	s.logger.Info("gitlab sync starting",
		zap.String("mode", mode),
		zap.Time("since", since),
		zap.Int("specs", len(s.specs)),
		zap.Int("projects", len(resolved)))

	runStart := time.Now()
	runID := fmt.Sprintf("gl-%d", runStart.UnixNano())
	totalMRs, totalReviews := 0, 0
	firstErr := resolveErr

	members, err := s.store.ListMembers(ctx)
	if err != nil {
		return err
	}
	byUsername := make(map[string]string, len(members))
	for _, m := range members {
		if m.GitLabUsername != "" {
			byUsername[strings.ToLower(m.GitLabUsername)] = m.ID
		}
	}

	for _, p := range resolved {
		mrs, err := s.client.ListMergeRequests(ctx, p.ID, ListMergeRequestsOptions{
			State: "all", UpdatedAfter: since, MaxPages: 10,
		})
		if err != nil {
			s.logger.Warn("ListMergeRequests failed", zap.String("project", p.PathWithNamespace), zap.Error(err))
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		s.logger.Debug("Fetched MRs", zap.String("project", p.PathWithNamespace), zap.Int("count", len(mrs)))

		toStore := make([]storage.PullRequest, 0, len(mrs))
		reviewBatch := make([]storage.PRReview, 0)

		for _, mr := range mrs {
			state := "open"
			if mr.MergedAt != nil {
				state = "merged"
			} else if mr.State == "closed" || mr.State == "locked" {
				state = "closed"
			}
			authorLogin := strings.ToLower(mr.Author.Username)
			authorID := byUsername[authorLogin]
			epicKey := ""
			if s.linker != nil {
				epicKey = s.linker.Link(mr)
			}
			id := fmt.Sprintf("%s!%d", p.PathWithNamespace, mr.IID)
			rec := storage.PullRequest{
				ID:          id,
				Source:      "gitlab",
				RepoID:      p.PathWithNamespace,
				Number:      mr.IID,
				Title:       mr.Title,
				State:       state,
				AuthorID:    authorID,
				AuthorLogin: authorLogin,
				HeadBranch:  mr.SourceBranch,
				BaseBranch:  mr.TargetBranch,
				EpicKey:     epicKey,
				CreatedAt:   mr.CreatedAt,
				MergedAt:    mr.MergedAt,
				ClosedAt:    mr.ClosedAt,
				FetchedAt:   runStart,
			}
			// Optional diff hydration. Off by default — turn on once
			// the dashboard wants additions/deletions on MRs.
			if s.HydrateDiff && mr.MergedAt != nil {
				if full, err := s.client.GetMergeRequest(ctx, p.ID, mr.IID); err == nil {
					rec.Additions = full.Diff.Additions
					rec.Deletions = full.Diff.Deletions
					rec.ChangedFiles = full.Diff.ChangedFiles
				}
			}

			// Notes → reviews. Same skip rules as the github side: drop
			// drafts, drop closed-without-merge older than 7d.
			skipNotes := mr.Draft || mr.WorkInProg
			if !skipNotes && mr.MergedAt == nil && mr.ClosedAt != nil &&
				time.Since(*mr.ClosedAt) > 7*24*time.Hour {
				skipNotes = true
			}
			if !skipNotes {
				notes, err := s.client.ListMRNotes(ctx, p.ID, mr.IID)
				if err != nil {
					s.logger.Warn("ListMRNotes failed",
						zap.String("project", p.PathWithNamespace),
						zap.Int("iid", mr.IID), zap.Error(err))
				} else {
					var first *time.Time
					for _, n := range notes {
						if n.System {
							continue
						}
						reviewerLogin := strings.ToLower(n.Author.Username)
						if reviewerLogin == authorLogin {
							continue
						}
						st, ok := classifyNote(n.Body)
						if !ok {
							continue
						}
						reviewBatch = append(reviewBatch, storage.PRReview{
							ID:            fmt.Sprintf("%s!%d/n%d", p.PathWithNamespace, mr.IID, n.ID),
							PRID:          rec.ID,
							Source:        "gitlab",
							ReviewerID:    byUsername[reviewerLogin],
							ReviewerLogin: reviewerLogin,
							State:         st,
							SubmittedAt:   n.CreatedAt,
						})
						if first == nil || n.CreatedAt.Before(*first) {
							first = &n.CreatedAt
						}
					}
					rec.FirstReviewAt = first
				}
			}
			toStore = append(toStore, rec)
		}

		if err := s.store.UpsertPullRequests(ctx, toStore); err != nil {
			return fmt.Errorf("upsert MRs for %s: %w", p.PathWithNamespace, err)
		}
		if err := s.store.UpsertPRReviews(ctx, reviewBatch); err != nil {
			return fmt.Errorf("upsert MR reviews for %s: %w", p.PathWithNamespace, err)
		}
		totalMRs += len(toStore)
		totalReviews += len(reviewBatch)
	}

	run := storage.CollectionRun{
		ID:          runID,
		CapturedAt:  runStart,
		Source:      "gitlab",
		DurationMs:  time.Since(runStart).Milliseconds(),
		RecordCount: totalMRs + totalReviews,
	}
	if firstErr != nil {
		run.Error = firstErr.Error()
	}
	if err := s.store.WriteCollectionRun(ctx, run); err != nil {
		return fmt.Errorf("write collection run: %w", err)
	}

	s.logger.Info("gitlab sync complete",
		zap.Int("mrs", totalMRs),
		zap.Int("reviews", totalReviews),
		zap.Duration("duration", time.Since(runStart)))
	return firstErr
}

// resolveProjects expands every spec, dropping archived projects when
// IncludeArchived is false. Duplicates across overlapping specs are
// dropped here so we don't waste API budget syncing the same project
// twice in one cycle.
func (s *Syncer) resolveProjects(ctx context.Context, now time.Time) ([]Project, error) {
	if len(s.specs) == 0 {
		return nil, nil
	}
	seen := make(map[int64]struct{}, 32)
	out := make([]Project, 0, 64)
	var firstErr error
	for _, spec := range s.specs {
		projects, err := s.expandSpec(ctx, spec, now)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, p := range projects {
			if !s.IncludeArchived && p.Archived {
				continue
			}
			if _, ok := seen[p.ID]; ok {
				continue
			}
			seen[p.ID] = struct{}{}
			out = append(out, p)
		}
	}
	return out, firstErr
}
