package db

import "database/sql"

// resumes 表：TypResume 用户简历云端同步。
// 每份简历由客户端生成 UUID (client_id) 作为稳定引用，允许离线创建后同步。
// files 用 JSONB 存 { "main.typ": "...", "template.typ": "..." } 键值对。
// mode: 'form' 或 'typst'（存量简历升级后默认 typst，保持行为）
// content: form 模式下的结构化简历数据（basics + sections）；typst 模式下忽略
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

// resumeVisualColumns 添加可视化编辑器所需的 mode + content 字段（幂等）。
// 存量简历默认 mode='typst' 保持原行为，content='{}'::jsonb 无害。
const resumeVisualColumns = `
ALTER TABLE resumes ADD COLUMN IF NOT EXISTS mode VARCHAR(16) NOT NULL DEFAULT 'typst';
ALTER TABLE resumes ADD COLUMN IF NOT EXISTS content JSONB NOT NULL DEFAULT '{}'::jsonb;
`

// MigrateTypResume creates the resumes table for TypResume cloud sync,
// then applies incremental column additions. Idempotent.
func MigrateTypResume(db *sql.DB) error {
	if _, err := db.Exec(typResumeSchema); err != nil {
		return err
	}
	_, err := db.Exec(resumeVisualColumns)
	return err
}
