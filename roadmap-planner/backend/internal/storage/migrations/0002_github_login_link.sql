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
-- Existing rows: the new columns default to NULL. They get populated as
-- the GitHub syncer re-fetches each PR. To make sure the integration
-- environment picks up the new shape on the very next sync (rather than
-- waiting for slow incremental drift), we also clear the github entry
-- in `collection_runs` so the next start is treated as a fresh
-- backfill. The PR table itself is untouched — UPSERT on the natural
-- key (`org/name#number`) refills the new columns in place.
-- ----------------------------------------------------------------------

ALTER TABLE pull_requests ADD COLUMN github_author_login TEXT;
ALTER TABLE pr_reviews    ADD COLUMN github_reviewer_login TEXT;

CREATE INDEX IF NOT EXISTS idx_pr_gh_author      ON pull_requests(github_author_login);
CREATE INDEX IF NOT EXISTS idx_review_gh_reviewer ON pr_reviews(github_reviewer_login);

DELETE FROM collection_runs WHERE source = 'github';
