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
	// 每次测试重建 hub_items / hub_folders 表（依赖 users 表已存在，且预置 user_id=1）
	_, err = db.Exec(`DROP TABLE IF EXISTS hub_items`)
	if err != nil {
		t.Fatalf("drop items: %v", err)
	}
	_, err = db.Exec(`DROP TABLE IF EXISTS hub_folders`)
	if err != nil {
		t.Fatalf("drop folders: %v", err)
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
    folder VARCHAR(200) NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT '',
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`)
	if err != nil {
		t.Fatalf("create items: %v", err)
	}
	_, err = db.Exec(`
CREATE TABLE hub_folders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    type VARCHAR(16) NOT NULL,
    name VARCHAR(200) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT hub_folders_unique UNIQUE (user_id, type, name)
)`)
	if err != nil {
		t.Fatalf("create folders: %v", err)
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
	a, err := repo.Create(1, &Item{Type: "bookmark", Title: "A"})
	if err != nil {
		t.Fatalf("seed A: %v", err)
	}
	if _, err := repo.Create(1, &Item{Type: "prompt", Title: "B"}); err != nil {
		t.Fatalf("seed B: %v", err)
	}

	// export
	dump, err := repo.ExportAll(1)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(dump) != 2 {
		t.Fatalf("dump len = %d", len(dump))
	}

	// 修改 dump：更新 A（title = A2），新增一条 C
	// 注意：dump 顺序按 updated_at 降序，A 不一定在 dump[0]，必须按 ID 定位
	for i := range dump {
		if dump[i].ID == a.ID {
			dump[i].Title = "A2"
		}
	}
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

	if _, err := repo.Create(1, &Item{Type: "bookmark", Title: "Old"}); err != nil {
		t.Fatalf("seed Old: %v", err)
	}

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

func TestImportBatchUserScope(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)

	// user 2 预置一条，作为不该被 user 1 的 import 影响的"外部"数据
	u2Item, err := repo.Create(2, &Item{Type: "bookmark", Title: "u2-untouched"})
	if err != nil {
		t.Fatalf("seed u2: %v", err)
	}

	// merge：即使传入 u2Item.ID，也不该 hijack 到 user 1 名下改写 u2 的条目
	// user 1 侧 merge 一条包含 u2Item.ID 的 item —— 期望：u2 的原条目不变，
	// user 1 得到一条新插入的（因为 id 未命中 user 1 scope，走 fallthrough INSERT）
	_, err = repo.ImportBatch(1, []Item{
		{ID: u2Item.ID, Type: "prompt", Title: "u1-merged"},
	}, "merge")
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	// user 2 那条必须原样存在
	u2Items, _ := repo.List(2)
	if len(u2Items) != 1 || u2Items[0].Title != "u2-untouched" {
		t.Fatalf("user 2 was affected by user 1 merge: %+v", u2Items)
	}
	// user 1 得到一条新条目
	u1Items, _ := repo.List(1)
	if len(u1Items) != 1 || u1Items[0].Title != "u1-merged" {
		t.Fatalf("user 1 merge missing: %+v", u1Items)
	}

	// replace：user 1 的 replace 不该动 user 2
	_, err = repo.ImportBatch(1, []Item{
		{Type: "skill", Title: "u1-replaced"},
	}, "replace")
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	u2Items, _ = repo.List(2)
	if len(u2Items) != 1 || u2Items[0].Title != "u2-untouched" {
		t.Fatalf("user 2 was wiped by user 1 replace: %+v", u2Items)
	}
}

func TestImportBatchReplaceRejectsEmpty(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)

	// 预置一条，若 replace 未被守护则会被删光
	if _, err := repo.Create(1, &Item{Type: "bookmark", Title: "keep-me"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := repo.ImportBatch(1, []Item{}, "replace")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	// 确认数据没被误删
	items, _ := repo.List(1)
	if len(items) != 1 || items[0].Title != "keep-me" {
		t.Fatalf("guard didn't work, data lost: %+v", items)
	}
}

func TestItemFolderRoundtrip(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)

	// Create 带 folder/icon，返回值与 List/findByID 都应读回
	created, err := repo.Create(1, &Item{Type: "bookmark", Title: "Ex", Folder: "Infini-AI", Icon: "https://cdn.example.com/logo.png"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Folder != "Infini-AI" {
		t.Fatalf("created folder = %q", created.Folder)
	}
	if created.Icon != "https://cdn.example.com/logo.png" {
		t.Fatalf("created icon = %q", created.Icon)
	}
	items, _ := repo.List(1)
	if len(items) != 1 || items[0].Folder != "Infini-AI" || items[0].Icon != "https://cdn.example.com/logo.png" {
		t.Fatalf("list folder/icon mismatch: %+v", items)
	}

	// Create 的 folder 自动登记进 hub_folders（含条数）
	folders, err := repo.ListFolders(1, "bookmark")
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	if len(folders) != 1 || folders[0].Name != "Infini-AI" || folders[0].ItemCount != 1 {
		t.Fatalf("auto upsert mismatch: %+v", folders)
	}

	// Update folder patch：条目 folder 更新，且新名字也被登记（旧目录保留）
	f2 := " work "
	updated, err := repo.Update(1, created.ID, UpdatePatch{Folder: &f2})
	if err != nil {
		t.Fatalf("update folder: %v", err)
	}
	if updated.Folder != " work " {
		t.Fatalf("updated folder = %q", updated.Folder)
	}
	folders, _ = repo.ListFolders(1, "bookmark")
	names := map[string]bool{}
	for _, f := range folders {
		names[f.Name] = true
	}
	if !names["Infini-AI"] || !names[" work "] {
		t.Fatalf("expected both folders registered: %+v", folders)
	}
}

func TestRepositoryFoldersCRUD(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)

	// CreateFolder
	f1, err := repo.CreateFolder(1, &Folder{Type: "bookmark", Name: "Infini-AI"})
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if f1.ID == 0 {
		t.Fatalf("expected non-zero id")
	}
	// 重名 → ErrFolderExists；但同名不同 type 允许（命名空间按 type 隔离）
	if _, err := repo.CreateFolder(1, &Folder{Type: "bookmark", Name: "Infini-AI"}); err != ErrFolderExists {
		t.Fatalf("want ErrFolderExists, got %v", err)
	}
	if _, err := repo.CreateFolder(1, &Folder{Type: "prompt", Name: "Infini-AI"}); err != nil {
		t.Fatalf("same name in another type should be allowed: %v", err)
	}

	// 条目挂到文件夹后，itemCount 正确
	if _, err := repo.Create(1, &Item{Type: "bookmark", Title: "A", Folder: "Infini-AI"}); err != nil {
		t.Fatalf("seed item A: %v", err)
	}
	if _, err := repo.Create(1, &Item{Type: "bookmark", Title: "B", Folder: "Infini-AI"}); err != nil {
		t.Fatalf("seed item B: %v", err)
	}
	folders, _ := repo.ListFolders(1, "bookmark")
	if len(folders) != 1 || folders[0].ItemCount != 2 {
		t.Fatalf("itemCount mismatch: %+v", folders)
	}

	// RenameFolder 级联更新条目 folder
	renamed, err := repo.RenameFolder(1, f1.ID, "芯穹")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.Name != "芯穹" {
		t.Fatalf("renamed name = %q", renamed.Name)
	}
	items, _ := repo.List(1)
	for _, it := range items {
		if it.Folder != "芯穹" {
			t.Fatalf("cascade failed, item %d folder = %q", it.ID, it.Folder)
		}
	}

	// 重命名为已存在的名字 → ErrFolderExists
	f2, err := repo.CreateFolder(1, &Folder{Type: "bookmark", Name: "other"})
	if err != nil {
		t.Fatalf("create other: %v", err)
	}
	if _, err := repo.RenameFolder(1, f2.ID, "芯穹"); err != ErrFolderExists {
		t.Fatalf("want ErrFolderExists on rename, got %v", err)
	}
	// 重命名不存在的 → ErrNotFound
	if _, err := repo.RenameFolder(1, 999999, "x"); err != ErrNotFound {
		t.Fatalf("want ErrNotFound on rename, got %v", err)
	}

	// DeleteFolder：条目回落未分类、条目不删
	if err := repo.DeleteFolder(1, f1.ID); err != nil {
		t.Fatalf("delete folder: %v", err)
	}
	items, _ = repo.List(1)
	if len(items) != 2 {
		t.Fatalf("items should survive folder deletion, got %d", len(items))
	}
	for _, it := range items {
		if it.Folder != "" {
			t.Fatalf("item %d folder should be reset to uncategorized, got %q", it.ID, it.Folder)
		}
	}
	// 删除不存在的 → ErrNotFound
	if err := repo.DeleteFolder(1, 999999); err != ErrNotFound {
		t.Fatalf("want ErrNotFound on delete, got %v", err)
	}
}

func TestImportUpsertsFolders(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)

	// merge 导入带 folder 的条目 → hub_folders 自动登记
	n, err := repo.ImportBatch(1, []Item{
		{Type: "bookmark", Title: "I1", Folder: "Infini-AI"},
		{Type: "prompt", Title: "I2", Folder: "写作"},
		{Type: "bookmark", Title: "I3"}, // 未分类不产生 folder 记录
	}, "merge")
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if n != 3 {
		t.Fatalf("import n=%d, want 3", n)
	}
	bm, _ := repo.ListFolders(1, "bookmark")
	if len(bm) != 1 || bm[0].Name != "Infini-AI" || bm[0].ItemCount != 1 {
		t.Fatalf("bookmark folders: %+v", bm)
	}
	pm, _ := repo.ListFolders(1, "prompt")
	if len(pm) != 1 || pm[0].Name != "写作" {
		t.Fatalf("prompt folders: %+v", pm)
	}
}

func TestReorderItems(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)

	// 同文件夹 3 条 + 另一文件夹 1 条 + 另一用户 1 条
	a, _ := repo.Create(1, &Item{Type: "bookmark", Title: "A", Folder: "F"})
	b, _ := repo.Create(1, &Item{Type: "bookmark", Title: "B", Folder: "F"})
	c, _ := repo.Create(1, &Item{Type: "bookmark", Title: "C", Folder: "F"})
	other, _ := repo.Create(1, &Item{Type: "bookmark", Title: "X", Folder: "G"})
	u2, _ := repo.Create(2, &Item{Type: "bookmark", Title: "U2", Folder: "F"})

	// 初始 position 全 0
	items, _ := repo.List(1)
	for _, it := range items {
		if it.Position != 0 {
			t.Fatalf("initial position should be 0, got %+v", items)
		}
	}

	// 重排为 [C, A, B] → position 1,2,3；List 按手动序返回
	if err := repo.ReorderItems(1, "bookmark", "F", []int64{c.ID, a.ID, b.ID}); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	items, _ = repo.List(1)
	got := []string{}
	for _, it := range items {
		if it.Folder == "F" {
			got = append(got, it.Title)
		}
	}
	if len(got) != 3 || got[0] != "C" || got[1] != "A" || got[2] != "B" {
		t.Fatalf("manual order mismatch: %v", got)
	}
	pos := map[int64]int{}
	for _, it := range items {
		pos[it.ID] = it.Position
	}
	if pos[c.ID] != 1 || pos[a.ID] != 2 || pos[b.ID] != 3 {
		t.Fatalf("positions mismatch: %v", pos)
	}
	// 未列入的条目 position 不变
	if pos[other.ID] != 0 || pos[u2.ID] != 0 {
		t.Fatalf("untouched items position changed: %v", pos)
	}

	// 混入别的文件夹/别的用户的 id → ErrNotFound，且已有顺序不被破坏
	if err := repo.ReorderItems(1, "bookmark", "F", []int64{a.ID, other.ID}); err != ErrNotFound {
		t.Fatalf("want ErrNotFound for foreign id, got %v", err)
	}
	if err := repo.ReorderItems(1, "bookmark", "F", []int64{u2.ID}); err != ErrNotFound {
		t.Fatalf("want ErrNotFound for other user's id, got %v", err)
	}
	items, _ = repo.List(1)
	for _, it := range items {
		if it.ID == c.ID && it.Position != 1 {
			t.Fatalf("failed reorder should not disturb positions: %+v", items)
		}
	}

	// 空 ids → 报错
	if err := repo.ReorderItems(1, "bookmark", "F", []int64{}); err == nil {
		t.Fatalf("expected error for empty ids")
	}
}
