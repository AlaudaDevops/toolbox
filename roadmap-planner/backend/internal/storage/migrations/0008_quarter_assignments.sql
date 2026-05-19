-- ----------------------------------------------------------------------
-- 0008_quarter_assignments — W7 (2026-05-19).
--
-- Stores the (issue_key, quarter_label) mapping used by the
-- release-cadence "quarter" bucketing on the dashboards. In the
-- DEVOPS Jira project, quarter labels are stored as a prefix on
-- Milestone issue summaries — e.g. "2026Q1：Security patches in
-- 2026Q1" — and Epics are linked to the right Milestone via a Blocks
-- link. This table flattens that chain so the contributions API can
-- look up a quarter for any PR or issue in O(1).
--
-- Schema:
--
--   issue_key      — primary key; either the Epic key (from the
--                    Milestone-Blocks pass) or any Jira key the
--                    operator has explicitly tagged.
--   quarter_label  — "<YYYY>Q<1-4>" string as it appears on the
--                    Milestone (case-sensitive, ASCII-only).
--   source         — "milestone" when the row came from the
--                    automated Milestone scan, "fallback" when it
--                    was inferred from calendar-quarter of the
--                    issue's resolution_date (used as a hot path
--                    for dangling issues until the Milestone pass
--                    fills in the proper assignment).
--
-- The Milestone-prefix Jira sync pass that populates this table from
-- the live Jira graph is a follow-up to W7; until it runs, the
-- API/service falls back to `2026Q<calendar-quarter-of-merged_at>`.
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS quarter_assignments (
    issue_key     TEXT PRIMARY KEY,
    quarter_label TEXT NOT NULL,
    source        TEXT NOT NULL DEFAULT 'milestone'
);

CREATE INDEX IF NOT EXISTS idx_quarter_label ON quarter_assignments(quarter_label);
