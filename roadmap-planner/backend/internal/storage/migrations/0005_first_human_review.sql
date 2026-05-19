-- ----------------------------------------------------------------------
-- 0005_first_human_review — W2 (bot consolidation, 2026-05-19).
--
-- Adds two columns that let the team-analytics layer separate bot noise
-- from human review activity without dropping the bot rows themselves
-- (we still want "renovate merged 80 PRs this week" visible under the
-- synthetic `bot` member). The contributions/bot.go startup pass
-- backfills both columns from existing data; no separate manual SQL
-- step is required.
--
--   pr_reviews.is_bot
--     1 when the reviewer_login matches the configured bot allowlist
--     (team_analytics.bot_logins). NetworkDensity adds `AND is_bot = 0`
--     to its first-review-latency query so the metric reflects the
--     human review experience.
--
--   pull_requests.first_human_review_at
--     MIN(submitted_at) over non-bot reviews on the PR. Lets the
--     dashboards query human review latency with one column instead
--     of a window-function over pr_reviews on every panel load.
--     Populated on first sync after this migration applies; refreshed
--     on every subsequent upsert through UpsertPullRequests.
-- ----------------------------------------------------------------------

ALTER TABLE pr_reviews ADD COLUMN is_bot INTEGER NOT NULL DEFAULT 0;
ALTER TABLE pull_requests ADD COLUMN first_human_review_at TIMESTAMP;
