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
		"blog_audit_logs", "blog_publish_jobs", "blog_draft_versions",
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

// TestPublishLifecycle job 生命周期：queued→building→succeeded，草稿标记 published + published_version。
func TestPublishLifecycle(t *testing.T) {
	pg := testDSN(t)
	defer pg.Close()
	truncateBlog(t, pg)
	uid, repo := newBlogUser(t, pg)
	d, _ := repo.CreateDraft(uid, Draft{Slug: "pub", Title: "T", Markdown: "m"})

	h := NewHandler(repo, testSecret, nil, "")

	job, err := repo.CreateJob(d.ID, ActionPublish, d.Version)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if job.Status != JobQueued {
		t.Fatalf("status = %s, want queued", job.Status)
	}
	sha := "deadbeef"
	if err := repo.SetJobBuilding(job.ID, sha); err != nil {
		t.Fatalf("set building: %v", err)
	}
	job.CommitSha = &sha
	if err := h.applyJobSucceeded(job); err != nil {
		t.Fatalf("applyJobSucceeded: %v", err)
	}
	got, _ := repo.GetDraftByID(d.ID)
	if got.Status != StatusPublished {
		t.Fatalf("status = %s, want published", got.Status)
	}
	if got.PublishedVersion == nil || *got.PublishedVersion != d.Version {
		t.Fatalf("published_version = %v, want %d", got.PublishedVersion, d.Version)
	}
	if got.PublishedCommitSha == nil || *got.PublishedCommitSha != sha {
		t.Fatalf("published_commit_sha mismatch")
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

// TestRecoverInterrupted 启动恢复：building+sha job 被 RecoverInterrupted 完成。
func TestRecoverInterrupted(t *testing.T) {
	pg := testDSN(t)
	defer pg.Close()
	truncateBlog(t, pg)
	uid, repo := newBlogUser(t, pg)
	d, _ := repo.CreateDraft(uid, Draft{Slug: "rec", Title: "T", Markdown: "m"})
	h := NewHandler(repo, testSecret, nil, "")
	job, _ := repo.CreateJob(d.ID, ActionPublish, d.Version)
	repo.SetJobBuilding(job.ID, "cafe")
	h.RecoverInterrupted()
	got, _ := repo.GetDraftByID(d.ID)
	if got.Status != StatusPublished {
		t.Fatalf("after recovery status = %s, want published", got.Status)
	}
}

// 为测试引入的引用，避免 import 抖动。
var _ = time.Second
