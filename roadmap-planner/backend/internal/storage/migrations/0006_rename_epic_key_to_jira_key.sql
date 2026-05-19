-- ----------------------------------------------------------------------
-- 0006_rename_epic_key_to_jira_key — W5 (2026-05-19).
--
-- The column was misnamed at birth: the linker pulls *any* matching Jira
-- key out of the source branch / title regex, not specifically an Epic
-- key. In a 4411-PR-linked sample from prod, the distribution was:
--   Story 2817, Bug 996, Job 199, Technical Debt 195, Document 118,
--   Sub-task 57, Task 25, Improvement 4, **Epic 0**.
--
-- Zero rows ever pointed at an actual Epic. The misnomer wasted future
-- reader time and was load-bearing in one place — `sprintCounts` joins
-- `pull_requests.epic_key = issue_snapshots.issue_key` and happened to
-- be correct only because most sprint members are Stories.
--
-- This rename is internal — no API client reads the JSON tag (verified
-- by grep on `frontend/` + the docs/). Safe to flip in one migration.
-- ----------------------------------------------------------------------

ALTER TABLE pull_requests RENAME COLUMN epic_key TO jira_key;

DROP INDEX IF EXISTS idx_pr_epic;
CREATE INDEX IF NOT EXISTS idx_pr_jira ON pull_requests(jira_key);
