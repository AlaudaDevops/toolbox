-- ----------------------------------------------------------------------
-- 0003 — GitLab MR ingestion: add `source` to PR/review tables, add
-- `gitlab_username` to members.
--
-- Why this migration exists
-- -------------------------
-- We keep the `pull_requests` and `pr_reviews` tables as the single home
-- for code-review activity across both GitHub and GitLab. The shapes are
-- 95% identical and the aggregator's member-week rollup already collapses
-- by (member_id, week_start, pillar_id, component) — the source of the
-- row is not part of that key, so the existing rollup picks up GitLab
-- data without any SQL change.
--
-- A single `source` column with a `'github'` default keeps existing rows
-- valid and makes the GitLab path a strict additive change.
--
-- `members.gitlab_username` mirrors `github_login`: nullable, unique when
-- set. The same operator-edit-wins semantics apply to it via PATCH.
-- ----------------------------------------------------------------------

ALTER TABLE pull_requests ADD COLUMN source TEXT NOT NULL DEFAULT 'github';
ALTER TABLE pr_reviews    ADD COLUMN source TEXT NOT NULL DEFAULT 'github';

-- The login columns were named for GitHub in 0002 but now hold the raw
-- author/reviewer login from whichever source the row came from. Rename
-- them to source-agnostic identifiers; the aggregator branches on
-- `source` to pick the right `members` column for the JOIN.
ALTER TABLE pull_requests RENAME COLUMN github_author_login TO author_login;
ALTER TABLE pr_reviews    RENAME COLUMN github_reviewer_login TO reviewer_login;

DROP INDEX IF EXISTS idx_pr_gh_author;
DROP INDEX IF EXISTS idx_review_gh_reviewer;
CREATE INDEX IF NOT EXISTS idx_pr_author          ON pull_requests(author_login);
CREATE INDEX IF NOT EXISTS idx_review_reviewer_l  ON pr_reviews(reviewer_login);
CREATE INDEX IF NOT EXISTS idx_pr_source_merged   ON pull_requests(source, merged_at);
CREATE INDEX IF NOT EXISTS idx_review_source      ON pr_reviews(source, submitted_at);

ALTER TABLE members ADD COLUMN gitlab_username TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS uq_members_gitlab ON members(gitlab_username) WHERE gitlab_username IS NOT NULL;
