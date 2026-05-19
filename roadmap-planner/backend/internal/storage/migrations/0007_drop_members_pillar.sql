-- ----------------------------------------------------------------------
-- 0007_drop_members_pillar — W9 (2026-05-19).
--
-- The `members.pillar_id` column was added back in B1's first cut as a
-- single-pillar tag for the operator drawer. Two problems with it:
--
--   1. Engineers contribute across pillars regularly. The per-member
--      pillar was a forced choice that didn't reflect reality.
--   2. The audit showed all 21 members had `pillar_id = NULL` in
--      prod, so `cross_pillar_review_pct` (which compared
--      `ma.pillar_id <> mr.pillar_id`) was structurally always 0%.
--
-- W9 (per Q9 / 2026-05-19 maintainer answer = option (i)) drops the
-- column outright. `cross_pillar_review_pct` is rewritten to derive
-- both sides from the PR's repo (via PillarMap.PillarsForRepo) and
-- the reviewer's pillar set (union of pillars across their recent PRs
-- in the same window). See service_extras.go::NetworkDensity.
--
-- Postgres + SQLite ≥ 3.35 support `ALTER TABLE … DROP COLUMN`. We
-- already require SQLite ≥3.25 for `RENAME COLUMN` (migration 0003),
-- and the alauda devpod ships with 3.46. Production runs on
-- modernc.org/sqlite which embeds 3.50+.
-- ----------------------------------------------------------------------

-- SQLite refuses to DROP a column with a dependent index. Drop the
-- index first; both engines tolerate the explicit order.
DROP INDEX IF EXISTS idx_members_pillar;

ALTER TABLE members DROP COLUMN pillar_id;
