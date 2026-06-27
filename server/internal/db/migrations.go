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
	_, err := db.Exec(schema)
	return err
}
