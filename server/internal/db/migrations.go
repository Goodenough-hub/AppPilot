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
	// 业务迁移：旅游专属分类升级为「组 + 叶子」两层结构（scope='trip'）
	return MigrateTripCategoriesV2(db)
}
