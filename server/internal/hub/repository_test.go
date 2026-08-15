package hub

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// 沿用 blog 包相同的集成测试策略：无 PG_TEST_DSN 就跳过。
func testDB(t *testing.T) *sql.DB {
	dsn := os.Getenv("PG_TEST_DSN")
	if dsn == "" {
		t.Skip("PG_TEST_DSN not set; skipping integration test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	// 每次测试重建 hub_items 表（依赖 users 表已存在，且预置 user_id=1）
	_, err = db.Exec(`DROP TABLE IF EXISTS hub_items`)
	if err != nil {
		t.Fatalf("drop: %v", err)
	}
	_, err = db.Exec(`
CREATE TABLE hub_items (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    type VARCHAR(16) NOT NULL,
    title VARCHAR(500) NOT NULL,
    url TEXT, content TEXT,
    tags TEXT[] NOT NULL DEFAULT '{}',
    favorite BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return db
}

func TestRepositoryCRUD(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)

	// Create
	url := "https://example.com"
	created, err := repo.Create(1, &Item{Type: "bookmark", Title: "Ex", URL: &url, Tags: []string{"a", "b"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected non-zero id")
	}
	if created.UserID != 1 {
		t.Fatalf("user_id = %d", created.UserID)
	}

	// List
	items, err := repo.List(1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || items[0].Title != "Ex" {
		t.Fatalf("list mismatch: %+v", items)
	}

	// Update
	newTitle := "Ex2"
	fav := true
	_, err = repo.Update(1, created.ID, UpdatePatch{Title: &newTitle, Favorite: &fav})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	items, _ = repo.List(1)
	if items[0].Title != "Ex2" || !items[0].Favorite {
		t.Fatalf("update didn't persist: %+v", items[0])
	}

	// Delete
	if err := repo.Delete(1, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	items, _ = repo.List(1)
	if len(items) != 0 {
		t.Fatalf("expected empty after delete, got %d", len(items))
	}
}

func TestRepositoryUserScope(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)

	if _, err := repo.Create(1, &Item{Type: "bookmark", Title: "u1-item"}); err != nil {
		t.Fatalf("seed u1: %v", err)
	}
	if _, err := repo.Create(2, &Item{Type: "bookmark", Title: "u2-item"}); err != nil {
		t.Fatalf("seed u2: %v", err)
	}

	items1, _ := repo.List(1)
	if len(items1) != 1 || items1[0].Title != "u1-item" {
		t.Fatalf("user 1 scope: %+v", items1)
	}
	items2, _ := repo.List(2)
	if len(items2) != 1 || items2[0].Title != "u2-item" {
		t.Fatalf("user 2 scope: %+v", items2)
	}
}

func TestUpdateReturnsNotFound(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)

	newTitle := "x"
	_, err := repo.Update(1, 999999, UpdatePatch{Title: &newTitle})
	if err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestDeleteReturnsNotFound(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)

	if err := repo.Delete(1, 999999); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
