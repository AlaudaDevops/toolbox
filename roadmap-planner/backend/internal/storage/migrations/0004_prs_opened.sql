-- ----------------------------------------------------------------------
-- 0004_prs_opened — add a parallel "PRs opened" weekly metric so the
-- Dashboard tab's Inflow vs Outflow board, and the team table's PRs-opened
-- column, can read straight from member_week_metrics. Bucketed by
-- pull_requests.created_at (mirror of the existing prs_merged column,
-- which buckets by merged_at).
-- ----------------------------------------------------------------------

ALTER TABLE member_week_metrics ADD COLUMN prs_opened INTEGER NOT NULL DEFAULT 0;

-- The aggregator's Rebuild pass populates this on next run; nothing to
-- backfill explicitly here. Existing rows pick up the DEFAULT 0 and
-- update on the next collection cycle's rebuild window.
