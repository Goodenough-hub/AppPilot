package db

import "database/sql"

// resumes 表：TypResume 用户简历云端同步。
// 每份简历由客户端生成 UUID (client_id) 作为稳定引用，允许离线创建后同步。
// files 用 JSONB 存 { "main.typ": "...", "template.typ": "..." } 键值对。
const typResumeSchema = `
CREATE TABLE IF NOT EXISTS resumes (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id    VARCHAR(64) NOT NULL,
    name         TEXT NOT NULL,
    active_file  VARCHAR(255) NOT NULL DEFAULT 'main.typ',
    files        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, client_id)
);
CREATE INDEX IF NOT EXISTS idx_resumes_user_id ON resumes(user_id);
CREATE INDEX IF NOT EXISTS idx_resumes_updated ON resumes(user_id, updated_at DESC);
`

// MigrateTypResume creates the resumes table for TypResume cloud sync.
// Idempotent, safe to call every startup.
func MigrateTypResume(db *sql.DB) error {
	_, err := db.Exec(typResumeSchema)
	return err
}
