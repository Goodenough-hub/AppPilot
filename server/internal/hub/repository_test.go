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

func TestExportImportMerge(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)

	// 初始 2 条
	a, _ := repo.Create(1, &Item{Type: "bookmark", Title: "A"})
	_, _ = repo.Create(1, &Item{Type: "prompt", Title: "B"})

	// export
	dump, err := repo.ExportAll(1)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(dump) != 2 {
		t.Fatalf("dump len = %d", len(dump))
	}

	// 修改 dump：更新 A（title = A2），新增一条 C
	dump[0].Title = "A2"
	dump = append(dump, Item{Type: "skill", Title: "C"})

	// import merge
	n, err := repo.ImportBatch(1, dump, "merge")
	if err != nil {
		t.Fatalf("import merge: %v", err)
	}
	if n != 3 {
		t.Fatalf("import merge affected = %d, want 3", n)
	}
	items, _ := repo.List(1)
	if len(items) != 3 {
		t.Fatalf("after merge got %d", len(items))
	}
	// 确认 A 被更新，不是被复制
	got, _ := repo.List(1)
	found := false
	for _, it := range got {
		if it.ID == a.ID && it.Title == "A2" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected id=%d title=A2, got %+v", a.ID, got)
	}
}

func TestExportImportReplace(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)

	_, _ = repo.Create(1, &Item{Type: "bookmark", Title: "Old"})

	// replace 传入完全不同的一批
	n, err := repo.ImportBatch(1, []Item{
		{Type: "prompt", Title: "New1"},
		{Type: "skill", Title: "New2"},
	}, "replace")
	if err != nil {
		t.Fatalf("import replace: %v", err)
	}
	if n != 2 {
		t.Fatalf("import replace n=%d, want 2", n)
	}
	items, _ := repo.List(1)
	if len(items) != 2 {
		t.Fatalf("after replace got %d", len(items))
	}
	for _, it := range items {
		if it.Title == "Old" {
			t.Fatalf("Old still present, replace failed")
		}
	}
}
