package db

import (
	"database/sql"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(16) NOT NULL DEFAULT 'user',
    app_scope     TEXT[] NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS categories (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       VARCHAR(64) NOT NULL,
    type       VARCHAR(8) NOT NULL,
    icon       VARCHAR(32) NOT NULL,
    color_hex  VARCHAR(7) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_system  BOOLEAN NOT NULL DEFAULT FALSE,
    parent_id  BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_categories_user_id ON categories(user_id);
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_type_parent ON categories(user_id, type, parent_id);

CREATE TABLE IF NOT EXISTS accounts (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(64) NOT NULL,
    type            VARCHAR(16) NOT NULL,
    icon            VARCHAR(32) NOT NULL,
    color_hex       VARCHAR(7) NOT NULL,
    initial_balance DECIMAL(15,2) NOT NULL DEFAULT 0,
    sort_order      INT NOT NULL DEFAULT 0,
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id);

CREATE TABLE IF NOT EXISTS transactions (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount        DECIMAL(15,2) NOT NULL,
    type          VARCHAR(8) NOT NULL,
    note          TEXT NOT NULL DEFAULT '',
    date          DATE NOT NULL,
    time          TIME,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    category_id   BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    account_id    BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    to_account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    source_id     VARCHAR(255),
    source_type   VARCHAR(32),
    vendor        VARCHAR(64)
);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_date ON transactions(user_id, date);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(user_id, type);
CREATE INDEX IF NOT EXISTS idx_transactions_category ON transactions(user_id, category_id);
CREATE INDEX IF NOT EXISTS idx_transactions_account ON transactions(user_id, account_id);

CREATE TABLE IF NOT EXISTS budgets (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount      DECIMAL(15,2) NOT NULL,
    month       INT NOT NULL,
    year        INT NOT NULL,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    UNIQUE(user_id, year, month, category_id)
);
CREATE INDEX IF NOT EXISTS idx_budgets_user_ym ON budgets(user_id, year, month);

CREATE TABLE IF NOT EXISTS recurring_transactions (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount        DECIMAL(15,2) NOT NULL,
    type          VARCHAR(8) NOT NULL,
    note          TEXT NOT NULL DEFAULT '',
    category_id   BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    account_id    BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    to_account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    frequency     VARCHAR(8) NOT NULL,
    interval      INT NOT NULL DEFAULT 1,
    day_of_month  INT,
    day_of_week   INT,
    next_date     DATE NOT NULL,
    start_date    DATE NOT NULL,
    end_date      DATE,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_recurring_user ON recurring_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_recurring_next_date ON recurring_transactions(next_date);
`

func Migrate(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	// 增量迁移：老库补列（pq 驱动需逐条 Exec）
	stmts := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE accounts ADD COLUMN IF NOT EXISTS parent_id BIGINT REFERENCES accounts(id) ON DELETE CASCADE`,
		`CREATE INDEX IF NOT EXISTS idx_accounts_parent_id ON accounts(user_id, parent_id)`,
		// 旅游账单：trips 表 + 分类 scope + 交易 trip_id
		`CREATE TABLE IF NOT EXISTS trips (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name VARCHAR(64) NOT NULL,
			start_date DATE,
			end_date DATE,
			budget DECIMAL(15,2) NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_trips_user_id ON trips(user_id)`,
		`ALTER TABLE categories ADD COLUMN IF NOT EXISTS scope VARCHAR(16) NOT NULL DEFAULT 'normal'`,
		`CREATE INDEX IF NOT EXISTS idx_categories_scope ON categories(user_id, scope)`,
		`ALTER TABLE transactions ADD COLUMN IF NOT EXISTS trip_id BIGINT REFERENCES trips(id) ON DELETE SET NULL`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_trip ON transactions(user_id, trip_id)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	// 业务迁移：老用户「支付宝」「微信」升级为分组+子账户
	if err := MigrateAccountsHierarchy(db); err != nil {
		return err
	}
	// 业务迁移：老用户「娱乐」分类补「其他」子分类
	if err := MigrateEntertainmentOther(db); err != nil {
		return err
	}
	// 业务迁移：老用户补「数字服务」顶级分类
	if err := migrateDigitalServiceTree(db); err != nil {
		return err
	}
	// 业务迁移：老用户补「生活」顶级分类
	if err := migrateLifeTree(db); err != nil {
		return err
	}
	// 业务迁移：把「生活」移到购物后、住房前，并改图标 🌿→🧴
	if err := migrateReorderLifeBeforeHousing(db); err != nil {
		return err
	}
	// 业务迁移：老用户「餐饮」补「夜宵」「小吃」「饮料」（晚餐后）
	if err := migrateInsertAfterParent(db, "餐饮", "晚餐", []seedNode{
		{Name: "夜宵", Icon: "🌙", Color: "#6366F1"},
		{Name: "小吃", Icon: "🍡", Color: "#8B5CF6"},
		{Name: "饮料", Icon: "🥤", Color: "#06B6D4"},
	}); err != nil {
		return err
	}
	// 业务迁移：老用户「交通」补「高铁」（打车后）
	if err := migrateInsertAfterParent(db, "交通", "打车", []seedNode{
		{Name: "高铁", Icon: "🚄", Color: "#6366F1"},
	}); err != nil {
		return err
	}
	// 业务迁移：老用户「影视」补「影院」（爱奇艺后）
	if err := migrateInsertAfterParent(db, "影视", "爱奇艺", []seedNode{
		{Name: "影院", Icon: "🎟️", Color: "#F59E0B"},
	}); err != nil {
		return err
	}
	// 业务迁移：老用户「餐饮」补「外卖」（饮料后）
	if err := migrateInsertAfterParent(db, "餐饮", "饮料", []seedNode{
		{Name: "外卖", Icon: "🛵", Color: "#F97316"},
	}); err != nil {
		return err
	}
	// 业务迁移：老用户「购物」补「外卖」（抖音后）
	if err := migrateInsertAfterParent(db, "购物", "抖音", []seedNode{
		{Name: "外卖", Icon: "🛵", Color: "#F97316"},
	}); err != nil {
		return err
	}
	// 业务迁移：老用户「购物」补「线下购物」（外卖后）
	if err := migrateInsertAfterParent(db, "购物", "外卖", []seedNode{
		{Name: "线下购物", Icon: "🏬", Color: "#8B5CF6"},
	}); err != nil {
		return err
	}
	// 业务迁移：老用户「住房」补「酒店」（物业后）
	if err := migrateInsertAfterParent(db, "住房", "物业", []seedNode{
		{Name: "酒店", Icon: "🏨", Color: "#3B82F6"},
	}); err != nil {
		return err
	}
	// 业务迁移：旅游专属分类升级为「组 + 叶子」两层结构（scope='trip'）
	if err := MigrateTripCategoriesV2(db); err != nil {
		return err
	}
	// 业务迁移：老用户「数字服务」补「通讯」（云服务后）
	if err := migrateInsertAfterParent(db, "数字服务", "云服务", []seedNode{
		{Name: "通讯", Icon: "📱", Color: "#3B82F6"},
	}); err != nil {
		return err
	}
	// 业务迁移：把「微信读书订阅」从「教育」移到「数字服务」（reparent）
	if err := migrateMoveWeixinReadSubscription(db); err != nil {
		return err
	}
	// 业务迁移：收入分类补齐 退款/报销/他人转入
	if err := migrateIncomeAddRefundReimburseTransferIn(db); err != nil {
		return err
	}
	// 业务迁移：平台子分类 icon 从 emoji 升级为品牌 slug（brand:<slug>）
	if err := migrateCategoryIconsToBrand(db); err != nil {
		return err
	}
	// 管理后台页面分析：前端埋点事件表
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS analytics_events (
    id          BIGSERIAL PRIMARY KEY,
    app         TEXT NOT NULL,
    user_id     BIGINT,
    event_type  TEXT NOT NULL,
    path        TEXT NOT NULL,
    title       TEXT,
    referrer    TEXT,
    user_agent  TEXT,
    ip          TEXT,
    session_id  TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ae_app_time ON analytics_events(app, created_at);
CREATE INDEX IF NOT EXISTS idx_ae_path ON analytics_events(path);
CREATE INDEX IF NOT EXISTS idx_ae_event_type ON analytics_events(app, event_type, created_at);
`); err != nil {
		return err
	}
	// 管理后台看板：dashboards + dashboard_widgets（admin 可编辑的仪表盘）。
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS dashboards (
    id          BIGSERIAL PRIMARY KEY,
    app         TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dashboard_widgets (
    id           BIGSERIAL PRIMARY KEY,
    dashboard_id BIGINT NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,
    title        TEXT NOT NULL,
    data_source  TEXT NOT NULL,
    config       JSONB DEFAULT '{}',
    grid_x       INT NOT NULL DEFAULT 0,
    grid_y       INT NOT NULL DEFAULT 0,
    grid_w       INT NOT NULL DEFAULT 4,
    grid_h       INT NOT NULL DEFAULT 3,
    sort_order   INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dw_dashboard ON dashboard_widgets(dashboard_id, sort_order);
	`); err != nil {
		return err
	}
	// FluxBlog：独立博客表族（与 users/transactions 完全隔离）。
	// blog_users 软删除 + token_version 使停用/删除账号的现有令牌立即失效。
	if err := MigrateBlog(db); err != nil {
		return err
	}
	// TypResume：用户简历云端同步（受 typresume scope 保护）。
	if err := MigrateTypResume(db); err != nil {
		return err
	}
	// Hub：私人工作台（bookmark/prompt/skill 统一管理）。
	if err := MigrateHub(db); err != nil {
		return err
	}
	// 默认看板 seed：为每个已知 app 创建默认 dashboard + widgets（幂等）。
	if err := SeedDashboards(db); err != nil {
		return err
	}
	return nil
}

// blogSchema 是 FluxBlog 的独立表族。与 finflow 的 users/transactions 等
// 完全隔离：blog 用独立账号、独立 JWT（iss=apppilot/aud=fluxblog/token_version）。
// 全部 IF NOT EXISTS，可被 db.Migrate 幂等重复调用。
const blogSchema = `
CREATE TABLE IF NOT EXISTS blog_users (
    id            BIGSERIAL PRIMARY KEY,
    username      VARCHAR(64) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    token_version BIGINT NOT NULL DEFAULT 0,
    deleted_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- username 唯一性仅约束未删除账号，软删除后可重建同名账号。
CREATE UNIQUE INDEX IF NOT EXISTS uq_blog_users_username ON blog_users(username) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_blog_users_enabled ON blog_users(is_enabled) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS blog_drafts (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES blog_users(id) ON DELETE RESTRICT,
    slug                VARCHAR(128) NOT NULL UNIQUE,
    title               VARCHAR(256) NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    tags                TEXT[] NOT NULL DEFAULT '{}',
    cover               VARCHAR(512),
    markdown            TEXT NOT NULL DEFAULT '',
    status              VARCHAR(16) NOT NULL DEFAULT 'draft',
    visibility          VARCHAR(16) NOT NULL DEFAULT 'private',
    version             BIGINT NOT NULL DEFAULT 1,
    published_commit_sha VARCHAR(64),
    published_version   BIGINT,
    published_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_blog_drafts_user ON blog_drafts(user_id);
CREATE INDEX IF NOT EXISTS idx_blog_drafts_status ON blog_drafts(status);

-- 自动保存快照；单篇最多保留最近 100 个（由 repository 在写入时裁剪）。
CREATE TABLE IF NOT EXISTS blog_draft_versions (
    id          BIGSERIAL PRIMARY KEY,
    draft_id    BIGINT NOT NULL REFERENCES blog_drafts(id) ON DELETE CASCADE,
    version     BIGINT NOT NULL,
    title       VARCHAR(256) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    tags        TEXT[] NOT NULL DEFAULT '{}',
    cover       VARCHAR(512),
    markdown    TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_blog_draft_versions_draft ON blog_draft_versions(draft_id, version DESC);

-- 草稿图片：暂存路径（受保护）+ 发布路径（公开 /blog/media/...）。
CREATE TABLE IF NOT EXISTS blog_assets (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT NOT NULL REFERENCES blog_users(id) ON DELETE RESTRICT,
    draft_id       BIGINT REFERENCES blog_drafts(id) ON DELETE SET NULL,
    sha256         CHAR(64) NOT NULL,
    filename       VARCHAR(255) NOT NULL,
    mime           VARCHAR(128) NOT NULL,
    size           BIGINT NOT NULL,
    staging_path   TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_blog_assets_user ON blog_assets(user_id);
CREATE INDEX IF NOT EXISTS idx_blog_assets_sha ON blog_assets(sha256);

CREATE TABLE IF NOT EXISTS blog_audit_logs (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT,
    action     VARCHAR(32) NOT NULL,
    detail     TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_blog_audit_logs_user ON blog_audit_logs(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS blog_projects (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES blog_users(id) ON DELETE RESTRICT,
    name       VARCHAR(128) NOT NULL,
    intro      TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_blog_projects_user_name ON blog_projects(user_id, name);
CREATE INDEX IF NOT EXISTS idx_blog_projects_user_order ON blog_projects(user_id, sort_order);
`

// MigrateBlog 创建 FluxBlog 独立表族。幂等，由 db.Migrate 调用。
func MigrateBlog(db *sql.DB) error {
	if _, err := db.Exec(blogSchema); err != nil {
		return err
	}
	// 增量列（老库补齐）
	stmts := []string{
		`ALTER TABLE blog_drafts ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ`,
		`ALTER TABLE blog_drafts ADD COLUMN IF NOT EXISTS published_version BIGINT`,
		`ALTER TABLE blog_drafts ADD COLUMN IF NOT EXISTS visibility VARCHAR(16) NOT NULL DEFAULT 'private'`,
		`CREATE INDEX IF NOT EXISTS idx_blog_drafts_visibility ON blog_drafts(visibility, status)`,
		// blog_publish_jobs 已废弃：发布改为 DB 内同步翻转，不再有 job 表。
		`DROP TABLE IF EXISTS blog_publish_jobs`,
		// blog_assets 的 publish_path/published_path 已废弃（图片不再经 Git 提交）。
		`ALTER TABLE blog_assets DROP COLUMN IF EXISTS publish_path`,
		`ALTER TABLE blog_assets DROP COLUMN IF EXISTS published_path`,
		// blog_draft_versions 去重：保留每 (draft_id, version) id 最大的一条，
		// 再建唯一索引，保证检查点版本唯一。
		`DELETE FROM blog_draft_versions a USING blog_draft_versions b
		 WHERE a.draft_id = b.draft_id AND a.version = b.version AND a.id < b.id`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_blog_draft_versions_draft_version
		 ON blog_draft_versions(draft_id, version)`,
		// 老库可能用列级 UNIQUE(username) 全局约束，软删除后无法重建同名账号。
		// 若存在则删除（仅当存在；PG 支持 IF EXISTS）。
		`ALTER TABLE blog_users DROP CONSTRAINT IF EXISTS blog_users_username_key`,
		// blog_projects 表 + blog_drafts.project_id 外键（project 管理功能）。
		`ALTER TABLE blog_drafts ADD COLUMN IF NOT EXISTS project_id BIGINT REFERENCES blog_projects(id) ON DELETE SET NULL`,
		`CREATE INDEX IF NOT EXISTS idx_blog_drafts_project ON blog_drafts(project_id) WHERE project_id IS NOT NULL`,
		// 定时发布：scheduled_publish_at 非 nil 表示未来发布时间，到点由 scheduler 提升为 published。
		`ALTER TABLE blog_drafts ADD COLUMN IF NOT EXISTS scheduled_publish_at TIMESTAMPTZ`,
		`CREATE INDEX IF NOT EXISTS idx_blog_drafts_scheduled ON blog_drafts(scheduled_publish_at) WHERE scheduled_publish_at IS NOT NULL`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

// hubSchema 是 Hub 私人工作台的表族定义（hub_items + hub_folders）。
// 全部 IF NOT EXISTS，幂等，由 db.Migrate 调用。
// hub_items.folder 为文件夹名（空串 = 未分类），命名空间按 (user_id, type) 隔离，
// 与 hub_folders 按名字松散关联（不建外键：允许条目先落地、文件夹记录由 upsert 补齐）。
const hubSchema = `
CREATE TABLE IF NOT EXISTS hub_items (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        VARCHAR(16) NOT NULL,
    title       VARCHAR(500) NOT NULL,
    url         TEXT,
    content     TEXT,
    tags        TEXT[] NOT NULL DEFAULT '{}',
    favorite    BOOLEAN NOT NULL DEFAULT FALSE,
    folder      VARCHAR(200) NOT NULL DEFAULT '',
    icon        TEXT NOT NULL DEFAULT '',
    position    INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT hub_items_type_valid CHECK (type IN ('bookmark','prompt','skill'))
);
CREATE INDEX IF NOT EXISTS idx_hub_items_fav      ON hub_items(user_id) WHERE favorite = TRUE;
CREATE INDEX IF NOT EXISTS idx_hub_items_tags     ON hub_items USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_hub_items_updated  ON hub_items(user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS hub_folders (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        VARCHAR(16) NOT NULL,
    name        VARCHAR(200) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT hub_folders_type_valid CHECK (type IN ('bookmark','prompt','skill')),
    CONSTRAINT hub_folders_unique UNIQUE (user_id, type, name)
);
CREATE INDEX IF NOT EXISTS idx_hub_folders_user ON hub_folders(user_id, type);
`

// MigrateHub 创建 Hub 表族。幂等，由 db.Migrate 调用。
func MigrateHub(db *sql.DB) error {
	if _, err := db.Exec(hubSchema); err != nil {
		return err
	}
	// 增量迁移：老库 hub_items 补 folder / icon / position 列
	stmts := []string{
		`ALTER TABLE hub_items ADD COLUMN IF NOT EXISTS folder VARCHAR(200) NOT NULL DEFAULT ''`,
		`ALTER TABLE hub_items ADD COLUMN IF NOT EXISTS icon TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE hub_items ADD COLUMN IF NOT EXISTS position INTEGER NOT NULL DEFAULT 0`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}
