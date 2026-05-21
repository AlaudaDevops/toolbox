/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package github

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
	"go.uber.org/zap"
)

// Linker resolves a PR to a Jira epic key, or returns "" if it cannot.
//
// Three built-in strategies are composed by Default:
//   - branch-name regex (e.g. "DEVOPS-12345-frobnicate")
//   - PR title leading token (e.g. "DEVOPS-12345: …")
//   - body / commit-message scan (any "[A-Z]+-\d+" mention)
//
// Custom Linkers are easy to add — implement Link.
type Linker interface {
	Link(pr PullRequest) string
}

// DefaultLinker composes branch + title strategies. Commit-scan is
// deferred to B3 because it requires an extra API call per PR.
func DefaultLinker(projectKey string) Linker {
	return &defaultLinker{
		key: strings.ToUpper(projectKey),
		re:  regexp.MustCompile(`(?i)\b(` + regexp.QuoteMeta(projectKey) + `-\d+)\b`),
	}
}

type defaultLinker struct {
	key string
	re  *regexp.Regexp
}

func (d *defaultLinker) Link(pr PullRequest) string {
	for _, candidate := range []string{pr.Head.Ref, pr.Title} {
		if m := d.re.FindString(candidate); m != "" {
			return strings.ToUpper(m)
		}
	}
	return ""
}

// Syncer pulls PRs and reviews into the storage.Store on a cadence.
//
// One Syncer per process; safe to call Sync from a goroutine driven by a
// time.Ticker.
//
// First-run behaviour: if collection_runs has no row for source="github",
// the Syncer treats this as the initial backfill and pulls PRs updated
// in the last BackfillDays days. Subsequent runs resume from one hour
// before the previous run's CapturedAt (the overlap absorbs any clock
// skew or PRs that mutated state without bumping `updated_at`).
//
// Wildcard expansion: a RepoConfig with Name == "*" (e.g. "AlaudaDevops/*")
// is resolved at Sync time against /orgs/{owner}/repos. By default
// archived repos and forks are filtered out and the resolved list is
// cached for WildcardTTL (24 h by default). Toggle the filters and TTL
// on this struct after construction if needed.
type Syncer struct {
	client       *Client
	store        storage.Store
	repos        []RepoConfig
	linker       Linker
	logger       *zap.Logger
	backfillDays int

	// Wildcard expansion knobs. All have sensible defaults; flip them
	// from the caller (e.g. cmd/server) before Sync is invoked.
	WildcardTTL     time.Duration // 0 -> DefaultWildcardTTL (24 h)
	IncludeArchived bool          // default false (archived repos skipped)
	IncludeForks    bool          // default false (forks skipped)

	// IsBotLogin is the W2 predicate that reclassifies a raw login as
	// the synthetic `bot` member. When non-nil and the predicate returns
	// true, PR `author_id` / review `reviewer_id` are set to `"bot"`
	// (so the dashboard credits the volume to one row) and
	// `pr_reviews.is_bot` is set to 1 (so NetworkDensity excludes the
	// row from human review-latency stats). Nil predicate disables
	// the feature.
	IsBotLogin func(login string) bool

	wildcardCache *wildcardCache
	nowFn         func() time.Time // injectable for cache-TTL tests
}

// RepoConfig describes one repo to track. Pillar/Component are optional
// classification labels; if set, they propagate onto every PR row at
// query time via the repos table (loaded by the aggregator in B2).
type RepoConfig struct {
	Owner     string
	Name      string
	PillarID  string
	Component string
}

// FullName returns "owner/name".
func (r RepoConfig) FullName() string { return r.Owner + "/" + r.Name }

// NewSyncer builds a Syncer. backfillDays sets the first-run window;
// pass 0 to use the package default (180).
func NewSyncer(client *Client, store storage.Store, repos []RepoConfig, linker Linker, backfillDays int) *Syncer {
	if backfillDays <= 0 {
		backfillDays = 180
	}
	return &Syncer{
		client:        client,
		store:         store,
		repos:         repos,
		linker:        linker,
		logger:        logger.WithComponent("github-syncer"),
		backfillDays:  backfillDays,
		WildcardTTL:   DefaultWildcardTTL,
		wildcardCache: newWildcardCache(),
		nowFn:         time.Now,
	}
}

// Sync pulls PRs updated since the last successful run for each repo,
// resolves authors against members, attempts epic linking, and persists.
//
// Reviews are fetched only for PRs we have not seen before (new) or that
// merged since the previous sync — this caps the per-cycle review-API
// fan-out to "new + recently-merged" rather than the entire history.
func (s *Syncer) Sync(ctx context.Context) error {
	if len(s.repos) == 0 {
		return nil
	}

	// Resolve OWNER/* specs against the org-repos endpoint before we
	// start the per-repo loop. A wildcard-expansion failure is recorded
	// but does not abort the cycle: any explicit + already-resolved
	// repos are still synced.
	resolved, resolveErr := s.resolveRepos(ctx)
	if resolveErr != nil {
		s.logger.Warn("wildcard repo resolution failed (partial)", zap.Error(resolveErr))
	}
	if len(resolved) == 0 {
		if resolveErr != nil {
			return resolveErr
		}
		return nil
	}

	// First run: reach back BackfillDays. Subsequent runs: resume from
	// one hour before the previous CapturedAt to absorb clock skew.
	since := time.Now().AddDate(0, 0, -s.backfillDays)
	mode := "backfill"
	if last, err := s.store.LatestCollectionRun(ctx, "github"); err == nil && last != nil {
		since = last.CapturedAt.Add(-1 * time.Hour)
		mode = "incremental"
	}
	s.logger.Info("github sync starting",
		zap.String("mode", mode),
		zap.Time("since", since),
		zap.Int("specs", len(s.repos)),
		zap.Int("repos", len(resolved)))

	runStart := time.Now()
	runID := fmt.Sprintf("gh-%d", runStart.UnixNano())
	totalPRs, totalReviews := 0, 0
	firstErr := resolveErr

	for _, repo := range resolved {
		prs, err := s.client.ListPullRequests(ctx, repo.Owner, repo.Name, ListPullRequestsOptions{
			State: "all", Sort: "updated", Direction: "desc",
			Since: since, MaxPages: 10,
		})
		if err != nil {
			s.logger.Warn("ListPullRequests failed", zap.String("repo", repo.FullName()), zap.Error(err))
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		s.logger.Debug("Fetched PRs", zap.String("repo", repo.FullName()), zap.Int("count", len(prs)))

		// Resolve members lazily — list once, build a login->id map.
		members, err := s.store.ListMembers(ctx)
		if err != nil {
			return err
		}
		byLogin := make(map[string]string, len(members))
		for _, m := range members {
			if m.GitHubLogin != "" {
				byLogin[strings.ToLower(m.GitHubLogin)] = m.ID
			}
		}

		toStore := make([]storage.PullRequest, 0, len(prs))
		reviewBatch := make([]storage.PRReview, 0)
		for _, pr := range prs {
			state := "open"
			if pr.MergedAt != nil {
				state = "merged"
			} else if pr.State == "closed" {
				state = "closed"
			}
			authorLogin := strings.ToLower(pr.User.Login)
			authorID := byLogin[authorLogin]
			if s.IsBotLogin != nil && s.IsBotLogin(authorLogin) {
				authorID = "bot"
			}
			epicKey := ""
			if s.linker != nil {
				epicKey = s.linker.Link(pr)
			}
			full := repo.FullName()
			rec := storage.PullRequest{
				ID:           fmt.Sprintf("%s#%d", full, pr.Number),
				Source:       "github",
				RepoID:       full,
				Number:       pr.Number,
				Title:        pr.Title,
				State:        state,
				AuthorID:     authorID,
				AuthorLogin:  authorLogin,
				HeadBranch:   pr.Head.Ref,
				BaseBranch:   pr.Base.Ref,
				Additions:    pr.Additions,
				Deletions:    pr.Deletions,
				ChangedFiles: pr.ChangedFiles,
				JiraKey:      epicKey,
				CreatedAt:    pr.CreatedAt,
				MergedAt:     pr.MergedAt,
				ClosedAt:     pr.ClosedAt,
				FetchedAt:    runStart,
			}

			// Reviews + commits — skip the API calls for:
			//   - drafts (no useful review data; first_commit_at meaningless
			//     until the PR is opened for review)
			//   - PRs closed without merge >7d ago (rarely accrue new
			//     reviews, and they dominate the call budget on noisy
			//     repos during backfill)
			//
			// We always fetch reviews + commits for merged PRs in window —
			// those drive the review-latency rollup and the Lead Time
			// first_commit_at field (DORA Phase 2).
			skipPRAPIs := pr.Draft
			if !skipPRAPIs && pr.MergedAt == nil && pr.ClosedAt != nil &&
				time.Since(*pr.ClosedAt) > 7*24*time.Hour {
				skipPRAPIs = true
			}
			if !skipPRAPIs {
				// Commits → first_commit_at (DORA Lead Time Dev-stage start).
				// Failures are logged but never block the PR upsert:
				// COALESCE in UpsertPullRequests preserves any earlier
				// value, and a later sync retries.
				commits, err := s.client.ListPRCommits(ctx, repo.Owner, repo.Name, pr.Number)
				if err != nil {
					s.logger.Warn("ListPRCommits failed",
						zap.String("repo", full), zap.Int("pr", pr.Number), zap.Error(err))
				} else {
					var firstCommit *time.Time
					for _, cm := range commits {
						d := cm.Commit.Author.Date
						if firstCommit == nil || d.Before(*firstCommit) {
							firstCommit = &d
						}
					}
					rec.FirstCommitAt = firstCommit
				}
			}
			if !skipPRAPIs {
				reviews, err := s.client.ListReviews(ctx, repo.Owner, repo.Name, pr.Number)
				if err != nil {
					s.logger.Warn("ListReviews failed",
						zap.String("repo", full), zap.Int("pr", pr.Number), zap.Error(err))
				} else {
					var first, firstHuman *time.Time
					for _, rev := range reviews {
						st := strings.ToLower(rev.State)
						reviewerLogin := strings.ToLower(rev.User.Login)
						reviewerID := byLogin[reviewerLogin]
						isBot := s.IsBotLogin != nil && s.IsBotLogin(reviewerLogin)
						if isBot {
							reviewerID = "bot"
						}
						reviewBatch = append(reviewBatch, storage.PRReview{
							ID:            fmt.Sprintf("%s#%d/r%d", full, pr.Number, rev.ID),
							PRID:          rec.ID,
							Source:        "github",
							ReviewerID:    reviewerID,
							ReviewerLogin: reviewerLogin,
							State:         st,
							SubmittedAt:   rev.SubmittedAt,
							IsBot:         isBot,
						})
						if first == nil || rev.SubmittedAt.Before(*first) {
							first = &rev.SubmittedAt
						}
						if !isBot && (firstHuman == nil || rev.SubmittedAt.Before(*firstHuman)) {
							firstHuman = &rev.SubmittedAt
						}
					}
					rec.FirstReviewAt = first
					rec.FirstHumanReviewAt = firstHuman
				}
			}
			toStore = append(toStore, rec)
		}

		if err := s.store.UpsertPullRequests(ctx, toStore); err != nil {
			return fmt.Errorf("upsert PRs for %s: %w", repo.FullName(), err)
		}
		if err := s.store.UpsertPRReviews(ctx, reviewBatch); err != nil {
			return fmt.Errorf("upsert reviews for %s: %w", repo.FullName(), err)
		}
		totalPRs += len(toStore)
		totalReviews += len(reviewBatch)
	}

	run := storage.CollectionRun{
		ID:          runID,
		CapturedAt:  runStart,
		Source:      "github",
		DurationMs:  time.Since(runStart).Milliseconds(),
		RecordCount: totalPRs + totalReviews,
	}
	if firstErr != nil {
		run.Error = firstErr.Error()
	}
	if err := s.store.WriteCollectionRun(ctx, run); err != nil {
		return fmt.Errorf("write collection run: %w", err)
	}

	s.logger.Info("github sync complete",
		zap.Int("prs", totalPRs),
		zap.Int("reviews", totalReviews),
		zap.Duration("duration", time.Since(runStart)))
	return firstErr
}
