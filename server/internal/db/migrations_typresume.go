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

// resumeVisualColumns 添加可视化编辑器及独立资源持久化所需字段（幂等）。
// 存量 form 记录里的头像会迁移到 assets；已丢失 content 的源码记录无法恢复原图。
const resumeVisualColumns = `
ALTER TABLE resumes ADD COLUMN IF NOT EXISTS mode VARCHAR(16) NOT NULL DEFAULT 'typst';
ALTER TABLE resumes ADD COLUMN IF NOT EXISTS content JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE resumes ADD COLUMN IF NOT EXISTS assets JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE resumes
SET assets = jsonb_build_object(
    CASE
        WHEN content #>> '{basics,avatarBase64}' LIKE 'data:image/png;base64,%' THEN 'avatar.png'
        WHEN content #>> '{basics,avatarBase64}' LIKE 'data:image/webp;base64,%' THEN 'avatar.webp'
        ELSE 'avatar.jpg'
    END,
    content #>> '{basics,avatarBase64}'
)
WHERE assets = '{}'::jsonb
  AND content #>> '{basics,avatarBase64}' ~ '^data:image/(jpeg|jpg|png|webp);base64,.+';
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
