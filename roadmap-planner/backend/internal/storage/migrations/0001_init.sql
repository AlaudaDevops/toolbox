-- ----------------------------------------------------------------------
-- Roadmap Planner — Team Analytics, schema v1
-- See docs/team-analytics/PROPOSAL.md §6 for rationale.
--
-- Portable SQL: no SQLite-isms. Same file is intended to apply against
-- Postgres once we outgrow SQLite (see PROPOSAL.md §9).
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS members (
    id              TEXT PRIMARY KEY,
    display_name    TEXT NOT NULL,
    email           TEXT,
    jira_account_id TEXT,
    github_login    TEXT,
    pillar_id       TEXT,
    active          INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_members_jira   ON members(jira_account_id) WHERE jira_account_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_members_github ON members(github_login)    WHERE github_login    IS NOT NULL;
CREATE        INDEX IF NOT EXISTS idx_members_pillar ON members(pillar_id);

CREATE TABLE IF NOT EXISTS collection_runs (
    id           TEXT PRIMARY KEY,
    captured_at  TIMESTAMP NOT NULL,
    source       TEXT NOT NULL,
    duration_ms  INTEGER NOT NULL,
    record_count INTEGER NOT NULL,
    error        TEXT
);
CREATE INDEX IF NOT EXISTS idx_runs_source_captured ON collection_runs(source, captured_at DESC);

CREATE TABLE IF NOT EXISTS issue_snapshots (
    run_id        TEXT NOT NULL,
    issue_key     TEXT NOT NULL,
    issue_type    TEXT NOT NULL,
    status        TEXT NOT NULL,
    assignee_id   TEXT,
    pillar_id     TEXT,
    components    TEXT,                  -- JSON array
    versions      TEXT,                  -- JSON array
    sprint_id     TEXT,
    story_points  REAL,
    created_at    TIMESTAMP NOT NULL,
    resolved_at   TIMESTAMP,
    PRIMARY KEY (run_id, issue_key)
);
CREATE INDEX IF NOT EXISTS idx_issues_assignee_run ON issue_snapshots(assignee_id, run_id);
CREATE INDEX IF NOT EXISTS idx_issues_pillar_run   ON issue_snapshots(pillar_id, run_id);
CREATE INDEX IF NOT EXISTS idx_issues_sprint       ON issue_snapshots(sprint_id);

CREATE TABLE IF NOT EXISTS repos (
    id        TEXT PRIMARY KEY,          -- "org/name"
    org       TEXT NOT NULL,
    name      TEXT NOT NULL,
    pillar_id TEXT,
    component TEXT,
    active    INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS pull_requests (
    id              TEXT PRIMARY KEY,    -- "org/name#number"
    repo_id         TEXT NOT NULL,
    number          INTEGER NOT NULL,
    title           TEXT NOT NULL,
    state           TEXT NOT NULL,
    author_id       TEXT,
    head_branch     TEXT,
    base_branch     TEXT,
    additions       INTEGER,
    deletions       INTEGER,
    changed_files   INTEGER,
    epic_key        TEXT,
    created_at      TIMESTAMP NOT NULL,
    first_review_at TIMESTAMP,
    merged_at       TIMESTAMP,
    closed_at       TIMESTAMP,
    fetched_at      TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_pr_author_merged ON pull_requests(author_id, merged_at);
CREATE INDEX IF NOT EXISTS idx_pr_repo_merged   ON pull_requests(repo_id, merged_at);
CREATE INDEX IF NOT EXISTS idx_pr_epic          ON pull_requests(epic_key);

CREATE TABLE IF NOT EXISTS pr_reviews (
    id           TEXT PRIMARY KEY,
    pr_id        TEXT NOT NULL,
    reviewer_id  TEXT,
    state        TEXT NOT NULL,
    submitted_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_review_reviewer ON pr_reviews(reviewer_id, submitted_at);
CREATE INDEX IF NOT EXISTS idx_review_pr       ON pr_reviews(pr_id);

CREATE TABLE IF NOT EXISTS member_week_metrics (
    member_id        TEXT NOT NULL,
    week_start       DATE NOT NULL,       -- Monday
    pillar_id        TEXT NOT NULL DEFAULT '',
    component        TEXT NOT NULL DEFAULT '',
    jira_issues_done INTEGER NOT NULL DEFAULT 0,
    jira_points_done REAL    NOT NULL DEFAULT 0,
    prs_merged       INTEGER NOT NULL DEFAULT 0,
    prs_reviewed     INTEGER NOT NULL DEFAULT 0,
    review_latency_p50_hours REAL,
    PRIMARY KEY (member_id, week_start, pillar_id, component)
);
CREATE INDEX IF NOT EXISTS idx_mwm_week_pillar ON member_week_metrics(week_start, pillar_id);

-- Schema version bookkeeping.
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    INTEGER PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL
);
