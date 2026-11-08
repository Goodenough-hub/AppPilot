package db

import "database/sql"

// OJ tables are intentionally separate from every other application's data.
// A submission keeps the version it was judged against; editing a problem
// inserts a new immutable version and leaves in-flight jobs untouched.
const ojSchema = `
CREATE TABLE IF NOT EXISTS oj_problems (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(80) NOT NULL UNIQUE,
    title VARCHAR(200) NOT NULL,
    difficulty VARCHAR(16) NOT NULL CHECK (difficulty IN ('easy','medium','hard')),
    tags TEXT[] NOT NULL DEFAULT '{}',
    current_version INTEGER NOT NULL DEFAULT 1,
    published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS oj_problem_versions (
    problem_id BIGINT NOT NULL REFERENCES oj_problems(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    statement TEXT NOT NULL,
    sample_input TEXT NOT NULL,
    sample_output TEXT NOT NULL,
    time_limit_ms INTEGER NOT NULL CHECK (time_limit_ms BETWEEN 100 AND 2000),
    memory_limit_mb INTEGER NOT NULL CHECK (memory_limit_mb BETWEEN 64 AND 256),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (problem_id, version)
);
CREATE TABLE IF NOT EXISTS oj_cases (
    problem_id BIGINT NOT NULL,
    version INTEGER NOT NULL,
    ordinal INTEGER NOT NULL,
    input TEXT NOT NULL,
    expected_output TEXT NOT NULL,
    PRIMARY KEY (problem_id, version, ordinal),
    FOREIGN KEY (problem_id, version) REFERENCES oj_problem_versions(problem_id, version) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS oj_submissions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    problem_id BIGINT NOT NULL,
    version INTEGER NOT NULL,
    kind VARCHAR(12) NOT NULL CHECK (kind IN ('run','submit')),
    code TEXT NOT NULL,
    custom_input TEXT NOT NULL DEFAULT '',
    status VARCHAR(12) NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','running','done')),
    verdict VARCHAR(24),
    compile_output TEXT NOT NULL DEFAULT '',
    stdout TEXT NOT NULL DEFAULT '',
    time_ms INTEGER NOT NULL DEFAULT 0,
    memory_kb INTEGER NOT NULL DEFAULT 0,
    passed INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (problem_id, version) REFERENCES oj_problem_versions(problem_id, version)
);
CREATE INDEX IF NOT EXISTS idx_oj_submissions_user_time ON oj_submissions(user_id, created_at DESC);
CREATE TABLE IF NOT EXISTS oj_jobs (
    id BIGSERIAL PRIMARY KEY,
    submission_id BIGINT NOT NULL UNIQUE REFERENCES oj_submissions(id) ON DELETE CASCADE,
    state VARCHAR(12) NOT NULL DEFAULT 'queued' CHECK (state IN ('queued','running','done')),
    lease_token TEXT,
    lease_until TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_oj_jobs_queue ON oj_jobs(state, lease_until, id);
`

func MigrateOJ(db *sql.DB) error {
	_, err := db.Exec(ojSchema)
	return err
}
