-- ----------------------------------------------------------------------
-- 0002 — Manual Jira → GitHub identity linking, raw login on PR rows.
--
-- Why this migration exists
-- -------------------------
-- v1 stored only the resolved member_id (`pull_requests.author_id`) on
-- ingest, derived by looking up `members.github_login` at sync time.
-- That made the linkage immutable: when the operator later filled in a
-- member's `github_login` via the new PATCH endpoint, the aggregator
-- still saw a NULL `author_id` on every historical PR and the rollup
-- never updated.
--
-- The fix is to keep the *raw* login the API returned alongside the
-- resolved id. The aggregator can then join `members` at rollup time,
-- so editing a member's `github_login` is enough to re-link history.
-- The same shape is added to pr_reviews so reviewer linkage is symmetric.
--
-- Existing rows have NULL in the new columns and there is no way to
-- recover the raw login from any other column — the GitHub API result
-- was thrown away at write time. So we wipe the GitHub side of the
-- store on this migration:
--
--   * `collection_runs WHERE source='github'` — drop the last-run pointer
--     so the next start treats this as a fresh first run.
--   * `pull_requests` / `pr_reviews` — empty them. Without this, the
--     aggregator's new JOIN finds NULL `github_author_login` on every
--     historical row, the rollup stays empty, and a github_login PATCH
--     looks like it does nothing.
--   * `member_week_metrics` — recompute is implicit on the next
--     aggregator pass; we drop the rows that would otherwise survive
--     the wipe and confuse the dashboard during the gap.
--
-- The next github sync (≤ a few minutes after the deploy lands) replays
-- the last `github.backfill_days` of activity and the new columns get
-- populated for every PR / review row.
-- ----------------------------------------------------------------------

ALTER TABLE pull_requests ADD COLUMN github_author_login TEXT;
ALTER TABLE pr_reviews    ADD COLUMN github_reviewer_login TEXT;

CREATE INDEX IF NOT EXISTS idx_pr_gh_author       ON pull_requests(github_author_login);
CREATE INDEX IF NOT EXISTS idx_review_gh_reviewer ON pr_reviews(github_reviewer_login);

DELETE FROM pr_reviews;
DELETE FROM pull_requests;
DELETE FROM collection_runs WHERE source = 'github';
DELETE FROM member_week_metrics WHERE prs_merged > 0 OR prs_reviewed > 0;
