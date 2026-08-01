package blog

import (
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"apppilot-server/internal/db"
	_ "github.com/lib/pq"
)

// testDSN 返回测试用 PG 连接串（来自 APPLOT_TEST_DSN）。未设置则跳过集成测试。
func testDSN(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("APPPLOT_TEST_DSN")
	if dsn == "" {
		t.Skip("APPPLOT_TEST_DSN not set; skipping PG integration tests")
	}
	pg, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open pg: %v", err)
	}
	if err := pg.Ping(); err != nil {
		t.Fatalf("ping pg: %v", err)
	}
	return pg
}

// truncateBlog 清空 blog 表族，保证用例隔离。
func truncateBlog(t *testing.T, pg *sql.DB) {
	t.Helper()
	for _, tbl := range []string{
		"blog_audit_logs", "blog_draft_versions",
		"blog_assets", "blog_drafts", "blog_users",
	} {
		if _, err := pg.Exec("TRUNCATE TABLE " + tbl + " RESTART IDENTITY CASCADE"); err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}
}

func newBlogUser(t *testing.T, pg *sql.DB) (int64, *Repository) {
	t.Helper()
	repo := NewRepository(pg)
	u, err := repo.Create("blogtester", "secret123")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u.ID, repo
}

// TestMigrationIdempotent MigrateBlog 连续执行两次不报错（幂等 + 增量列/索引）。
func TestMigrationIdempotent(t *testing.T) {
	pg := testDSN(t)
	defer pg.Close()
	if err := db.MigrateBlog(pg); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := db.MigrateBlog(pg); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

// TestDraftVersionAndCheckpoint 草稿版本递增 + 检查点策略（5min 内不自动写、手动写）。
func TestDraftVersionAndCheckpoint(t *testing.T) {
	pg := testDSN(t)
	defer pg.Close()
	truncateBlog(t, pg)
	uid, repo := newBlogUser(t, pg)

	d, err := repo.CreateDraft(uid, Draft{Slug: "p1", Title: "T", Markdown: "v0"})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if d.Version != 1 {
		t.Fatalf("version = %d, want 1", d.Version)
	}
	// v1 检查点应在创建时写入
	vs, _ := repo.ListVersions(d.ID)
	if len(vs) != 1 || vs[0].Version != 1 {
		t.Fatalf("expected 1 checkpoint at v1, got %d", len(vs))
	}

	// 保存（baseVersion=1）→ version 2；5min 内不应自动新增检查点
	d2, _, err := repo.UpdateDraft(uid, d.ID, 1, UpdateDraftRequest{Markdown: strPtr("v1")})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if d2.Version != 2 {
		t.Fatalf("version = %d, want 2", d2.Version)
	}
	vs, _ = repo.ListVersions(d.ID)
	if len(vs) != 1 {
		t.Fatalf("5min 内不应自动新增检查点，got %d", len(vs))
	}

	// 手动检查点 → v2
	if err := repo.CreateCheckpoint(d2); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	vs, _ = repo.ListVersions(d.ID)
	if len(vs) != 2 || vs[0].Version != 2 {
		t.Fatalf("expected v2 checkpoint, got %+v", vs)
	}
}

// TestUpdateDraftConflict 过期 baseVersion 返回 409 + 服务端版本。
func TestUpdateDraftConflict(t *testing.T) {
	pg := testDSN(t)
	defer pg.Close()
	truncateBlog(t, pg)
	uid, repo := newBlogUser(t, pg)
	d, _ := repo.CreateDraft(uid, Draft{Slug: "c", Title: "T", Markdown: "a"})
	_, _, _ = repo.UpdateDraft(uid, d.ID, 1, UpdateDraftRequest{Markdown: strPtr("b")}) // version 2
	// 用过期 baseVersion=1 再保存 → 冲突
	_, serverVersion, err := repo.UpdateDraft(uid, d.ID, 1, UpdateDraftRequest{Markdown: strPtr("c")})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	if serverVersion != 2 {
		t.Fatalf("serverVersion = %d, want 2", serverVersion)
	}
}

// TestRestoreVersion 恢复版本写入新版本快照。
func TestRestoreVersion(t *testing.T) {
	pg := testDSN(t)
	defer pg.Close()
	truncateBlog(t, pg)
	uid, repo := newBlogUser(t, pg)
	d, _ := repo.CreateDraft(uid, Draft{Slug: "r", Title: "T", Markdown: "a"})
	_, _, _ = repo.UpdateDraft(uid, d.ID, 1, UpdateDraftRequest{Markdown: strPtr("b")}) // v2 + manual? no
	repo.CreateCheckpoint(&Draft{ID: d.ID, UserID: uid, Version: 2, Title: "T", Markdown: "b"})
	restored, err := repo.RestoreVersion(uid, d.ID, 2)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.Version != 3 {
		t.Fatalf("restored version = %d, want 3", restored.Version)
	}
	if restored.Markdown != "b" {
		t.Fatalf("restored markdown = %s, want b", restored.Markdown)
	}
}

// 为测试引入的引用，避免 import 抖动。
var _ = time.Second

// publishForTest 把草稿直接置为已发布（保留其现有 visibility），供读接口测试。
func publishForTest(t *testing.T, repo *Repository, id int64) {
	t.Helper()
	d, err := repo.GetDraftByID(id)
	if err != nil {
		t.Fatalf("get draft: %v", err)
	}
	if _, err := repo.PublishDraft(id, d.Visibility); err != nil {
		t.Fatalf("publish: %v", err)
	}
}

// TestPublishSyncFlip 发布/撤回为 DB 内同步翻转：published_version/published_at 写入，
// 撤回回 draft，再发布保留 visibility。
func TestPublishSyncFlip(t *testing.T) {
	pg := testDSN(t)
	defer pg.Close()
	truncateBlog(t, pg)
	uid, repo := newBlogUser(t, pg)
	d, _ := repo.CreateDraft(uid, Draft{Slug: "flip", Title: "T", Markdown: "m", Visibility: VisibilityPublic})

	pub, err := repo.PublishDraft(d.ID, VisibilityPublic)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if pub.Status != StatusPublished {
		t.Fatalf("status = %s, want published", pub.Status)
	}
	if pub.PublishedVersion == nil || *pub.PublishedVersion != d.Version {
		t.Fatalf("published_version = %v, want %d", pub.PublishedVersion, d.Version)
	}
	if pub.PublishedAt == nil {
		t.Fatal("published_at should be set on first publish")
	}
	firstAt := *pub.PublishedAt

	// 撤回 → draft，visibility 保留。
	unpub, err := repo.UnpublishDraft(d.ID)
	if err != nil {
		t.Fatalf("unpublish: %v", err)
	}
	if unpub.Status != StatusDraft {
		t.Fatalf("status = %s, want draft", unpub.Status)
	}
	if unpub.Visibility != VisibilityPublic {
		t.Fatalf("visibility = %s, want public (preserved)", unpub.Visibility)
	}

	// 再发布：published_at 保持首次值不变。
	pub2, _ := repo.PublishDraft(d.ID, VisibilityPrivate)
	if pub2.Visibility != VisibilityPrivate {
		t.Fatalf("re-publish visibility = %s, want private", pub2.Visibility)
	}
	if pub2.PublishedAt == nil || !pub2.PublishedAt.UTC().Equal(firstAt.UTC()) {
		t.Fatalf("published_at = %v, want %v (preserved)", pub2.PublishedAt, firstAt)
	}
}

// TestVisibilityFilter 公开/私有可见性过滤：ListPublishedPublic 只回 public+published，
// ListPublishedPrivate 只回 private+published。
func TestVisibilityFilter(t *testing.T) {
	pg := testDSN(t)
	defer pg.Close()
	truncateBlog(t, pg)
	uid, repo := newBlogUser(t, pg)

	pub, _ := repo.CreateDraft(uid, Draft{Slug: "pub", Title: "公开", Markdown: "m", Visibility: VisibilityPublic})
	priv, _ := repo.CreateDraft(uid, Draft{Slug: "priv", Title: "私有", Markdown: "m"}) // 默认 private
	publishForTest(t, repo, pub.ID)
	publishForTest(t, repo, priv.ID)

	got, err := repo.ListPublishedPublic()
	if err != nil {
		t.Fatalf("list public: %v", err)
	}
	if len(got) != 1 || got[0].Slug != "pub" {
		t.Fatalf("ListPublishedPublic = %+v, want only pub", got)
	}
	privs, err := repo.ListPublishedPrivate(uid)
	if err != nil {
		t.Fatalf("list private: %v", err)
	}
	if len(privs) != 1 || privs[0].Slug != "priv" {
		t.Fatalf("ListPublishedPrivate = %+v, want only priv", privs)
	}
}

// TestPublicPrivateRead 单篇按 slug 读取的可见性隔离。
func TestPublicPrivateRead(t *testing.T) {
	pg := testDSN(t)
	defer pg.Close()
	truncateBlog(t, pg)
	uid, repo := newBlogUser(t, pg)

	pub, _ := repo.CreateDraft(uid, Draft{Slug: "pub", Title: "T", Markdown: "正文", Visibility: VisibilityPublic})
	priv, _ := repo.CreateDraft(uid, Draft{Slug: "priv", Title: "T", Markdown: "私密正文"})
	publishForTest(t, repo, pub.ID)
	publishForTest(t, repo, priv.ID)

	// 公开 slug 匿名可读；私有 slug 公开接口 404。
	if d, err := repo.GetPublishedPublicBySlug("pub"); err != nil || d.Markdown != "正文" {
		t.Fatalf("public read pub: err=%v d=%+v", err, d)
	}
	if _, err := repo.GetPublishedPublicBySlug("priv"); !errors.Is(err, ErrDraftNotFound) {
		t.Fatalf("public read priv: err=%v, want ErrDraftNotFound", err)
	}
	// 私有 slug 属主可读；公开 slug 私有接口 404。
	if d, err := repo.GetPublishedPrivateBySlug(uid, "priv"); err != nil || d.Markdown != "私密正文" {
		t.Fatalf("private read priv: err=%v d=%+v", err, d)
	}
	if _, err := repo.GetPublishedPrivateBySlug(uid, "pub"); !errors.Is(err, ErrDraftNotFound) {
		t.Fatalf("private read pub: err=%v, want ErrDraftNotFound", err)
	}
}

// TestDraftIsPublicPublished 公开图片免鉴权判定的依据：仅 public+published 为 true。
func TestDraftIsPublicPublished(t *testing.T) {
	pg := testDSN(t)
	defer pg.Close()
	truncateBlog(t, pg)
	uid, repo := newBlogUser(t, pg)

	pub, _ := repo.CreateDraft(uid, Draft{Slug: "pub", Title: "T", Markdown: "m", Visibility: VisibilityPublic})
	priv, _ := repo.CreateDraft(uid, Draft{Slug: "priv", Title: "T", Markdown: "m"})

	// 未发布 → false
	if ok, _ := repo.DraftIsPublicPublished(pub.ID); ok {
		t.Fatalf("draft (not published) should not be public-published")
	}
	publishForTest(t, repo, pub.ID)
	publishForTest(t, repo, priv.ID)
	if ok, _ := repo.DraftIsPublicPublished(pub.ID); !ok {
		t.Fatalf("public+published should be true")
	}
	if ok, _ := repo.DraftIsPublicPublished(priv.ID); ok {
		t.Fatalf("private+published should be false")
	}
}

// TestImportDraftIdempotent 重复导入同 slug 不产生新行，字段被更新，published_at 保留首次值。
func TestImportDraftIdempotent(t *testing.T) {
	pg := testDSN(t)
	defer pg.Close()
	truncateBlog(t, pg)
	uid, repo := newBlogUser(t, pg)

	firstAt := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	d1 := Draft{Slug: "imp", Title: "v1", Description: "d", Markdown: "a", Version: 1}
	if _, err := repo.ImportDraft(uid, d1, &firstAt, true); err != nil {
		t.Fatalf("import 1: %v", err)
	}
	d2 := Draft{Slug: "imp", Title: "v2", Description: "d2", Markdown: "b", Version: 1}
	if _, err := repo.ImportDraft(uid, d2, &firstAt, true); err != nil {
		t.Fatalf("import 2: %v", err)
	}

	// 仅一行
	var n int
	if err := pg.QueryRow(`SELECT COUNT(*) FROM blog_drafts WHERE slug = 'imp'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("rows = %d, want 1 (idempotent)", n)
	}
	got, err := repo.GetPublishedPublicBySlug("imp")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Title != "v2" || got.Markdown != "b" {
		t.Fatalf("updated fields wrong: title=%s markdown=%s", got.Title, got.Markdown)
	}
	if got.Status != StatusPublished || got.Visibility != VisibilityPublic {
		t.Fatalf("status=%s visibility=%s, want published/public", got.Status, got.Visibility)
	}
	if got.PublishedAt == nil || !got.PublishedAt.UTC().Equal(firstAt) {
		t.Fatalf("published_at = %v, want %v (preserved)", got.PublishedAt, firstAt)
	}
}

// TestSearchPublic ILIKE 命中公开已发布，私有不命中。
func TestSearchPublic(t *testing.T) {
	pg := testDSN(t)
	defer pg.Close()
	truncateBlog(t, pg)
	uid, repo := newBlogUser(t, pg)

	pub, _ := repo.CreateDraft(uid, Draft{Slug: "s-pub", Title: "搜索关键词", Markdown: "正文含关键词", Visibility: VisibilityPublic})
	priv, _ := repo.CreateDraft(uid, Draft{Slug: "s-priv", Title: "搜索关键词私密", Markdown: "x"})
	publishForTest(t, repo, pub.ID)
	publishForTest(t, repo, priv.ID)

	items, total, err := repo.SearchPublic("关键词", 10, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Slug != "s-pub" {
		t.Fatalf("search = total=%d items=%+v, want only s-pub", total, items)
	}
}
