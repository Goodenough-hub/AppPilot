package admin

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"apppilot-server/internal/auth"
	"apppilot-server/internal/blog"
	"apppilot-server/internal/db"
	"apppilot-server/internal/middleware"
	_ "github.com/lib/pq"
)

const testBlogSecret = "test-blog-jwt-secret-must-be-at-least-32-chars-long"

// testPG 返回测试用 PG 连接串（来自 APPLOT_TEST_DSN）。未设置则跳过集成测试。
func testPG(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("APPPLOT_TEST_DSN")
	if dsn == "" {
		t.Skip("APPPLOT_TEST_DSN not set; skipping admin SSO integration tests")
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

// truncateAll 清空 users 与 blog 表族，保证用例隔离。
func truncateAll(t *testing.T, pg *sql.DB) {
	t.Helper()
	for _, tbl := range []string{
		"blog_audit_logs", "blog_draft_versions",
		"blog_assets", "blog_drafts", "blog_projects", "blog_users",
		"transactions", "categories", "accounts", "budgets", "recurring_transactions",
		"users",
	} {
		if _, err := pg.Exec("TRUNCATE TABLE " + tbl + " RESTART IDENTITY CASCADE"); err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}
}

// newAdminRouter 创建带完整中间件链的测试 router：admin JWT 鉴权 + AdminRequired。
// 调用方用 adminToken(c) 设上下文身份，或走真实登录拿 JWT。
func newAdminRouter(t *testing.T, pg *sql.DB, jwtSecret string) (*gin.Engine, *auth.Repository, *blog.Repository) {
	t.Helper()
	if err := db.Migrate(pg); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.MigrateBlog(pg); err != nil {
		t.Fatalf("migrate blog: %v", err)
	}
	authRepo := auth.NewRepository(pg)
	blogRepo := blog.NewRepository(pg)
	adminH := NewHandler(pg, authRepo, jwtSecret, blogRepo, testBlogSecret)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	v1 := r.Group("/api/v1")
	g := v1.Group("/admin", middleware.AuthRequired(jwtSecret), middleware.AdminRequired())
	adminH.Register(g)
	return r, authRepo, blogRepo
}

// newAdminUser 创建一名 admin 角色用户并签发 JWT，返回 Authorization header 值。
func newAdminUser(t *testing.T, repo *auth.Repository, jwtSecret, username string) string {
	t.Helper()
	u, err := repo.Create(username, "pw123456", "admin", []string{"admin"})
	if err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	token, _, err := auth.GenerateToken(u, jwtSecret)
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}
	return "Bearer " + token
}

// TestStartBlogSession_AutoCreateStub 首次调用：blog_users stub 不存在，
// 端点应自动创建（随机密码不可见）+ 下发两个 cookie + 返回 redirect。
func TestStartBlogSession_AutoCreateStub(t *testing.T) {
	pg := testPG(t)
	defer pg.Close()
	truncateAll(t, pg)

	jwtSecret := "test-admin-secret-must-be-32-chars-or-more"
	r, authRepo, blogRepo := newAdminRouter(t, pg, jwtSecret)
	authHeader := newAdminUser(t, authRepo, jwtSecret, "alice")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/blog/session", nil)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Redirect string `json:"redirect"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Redirect != "/blog/studio/" {
		t.Errorf("redirect = %q, want /blog/studio/", body.Redirect)
	}
	if body.Username != "alice" {
		t.Errorf("username = %q, want alice", body.Username)
	}

	// cookie 检查
	var tokenFound, sessionFound bool
	for _, ck := range w.Result().Cookies() {
		switch ck.Name {
		case "fluxblog_token":
			tokenFound = true
			if !ck.Secure {
				t.Errorf("fluxblog_token should be Secure under https")
			}
			if !ck.HttpOnly {
				t.Errorf("fluxblog_token must be HttpOnly")
			}
		case "fluxblog_session":
			sessionFound = true
		}
	}
	if !tokenFound || !sessionFound {
		t.Errorf("cookies missing: token=%v session=%v", tokenFound, sessionFound)
	}

	// stub 确实写入了 blog_users
	bu, err := blogRepo.FindByUsernameActive("alice")
	if err != nil {
		t.Fatalf("blog_user stub not created: %v", err)
	}
	if !bu.IsEnabled {
		t.Errorf("stub should be enabled by default")
	}
	// 随机密码不应等于用户名、不应为空
	if len(bu.PasswordHash) == 0 {
		t.Errorf("password hash should be set")
	}
}

// TestStartBlogSession_ReuseExisting 第二次调用应复用同 ID 的 stub，
// 不重复创建（密码哈希也不应被改动）。
func TestStartBlogSession_ReuseExisting(t *testing.T) {
	pg := testPG(t)
	defer pg.Close()
	truncateAll(t, pg)

	jwtSecret := "test-admin-secret-must-be-32-chars-or-more"
	r, authRepo, blogRepo := newAdminRouter(t, pg, jwtSecret)
	authHeader := newAdminUser(t, authRepo, jwtSecret, "bob")

	doReq := func() string {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/blog/session", nil)
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("X-Forwarded-Proto", "https")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		bu, _ := blogRepo.FindByUsernameActive("bob")
		return bu.PasswordHash
	}

	firstHash := doReq()
	secondHash := doReq()

	bu, err := blogRepo.FindByUsernameActive("bob")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if bu.Username != "bob" {
		t.Errorf("username = %q, want bob", bu.Username)
	}
	if firstHash != secondHash {
		t.Errorf("password hash changed between calls: SSO should not regenerate stub password")
	}
}

// TestStartBlogSession_DisabledStub 当 stub 已存在但被停用时返回 403，
// 提示 admin 在 blog-users 页重新启用。
func TestStartBlogSession_DisabledStub(t *testing.T) {
	pg := testPG(t)
	defer pg.Close()
	truncateAll(t, pg)

	jwtSecret := "test-admin-secret-must-be-32-chars-or-more"
	r, authRepo, blogRepo := newAdminRouter(t, pg, jwtSecret)
	authHeader := newAdminUser(t, authRepo, jwtSecret, "carol")

	// 首次创建 stub
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/blog/session", nil)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first call status = %d", w.Code)
	}

	// 停用 stub
	bu, _ := blogRepo.FindByUsernameActive("carol")
	falseVal := false
	if _, err := blogRepo.UpdateProfile(bu.ID, nil, &falseVal); err != nil {
		t.Fatalf("disable stub: %v", err)
	}

	// 再次调用应 403
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/blog/session", nil)
	req2.Header.Set("Authorization", authHeader)
	req2.Header.Set("X-Forwarded-Proto", "https")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403; body=%s", w2.Code, w2.Body.String())
	}
}

// TestStartBlogSession_NoAuth 未带 admin JWT 应 401。
func TestStartBlogSession_NoAuth(t *testing.T) {
	pg := testPG(t)
	defer pg.Close()
	truncateAll(t, pg)

	jwtSecret := "test-admin-secret-must-be-32-chars-or-more"
	r, _, _ := newAdminRouter(t, pg, jwtSecret)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/blog/session", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// TestStartBlogSession_BlogJwtIsolated 签发的 blog JWT 应使用 BlogJWTSecret
// 而非 admin JWT secret——否则 ParseToken 会失败（不同 secret）。
func TestStartBlogSession_BlogJwtIsolated(t *testing.T) {
	pg := testPG(t)
	defer pg.Close()
	truncateAll(t, pg)

	jwtSecret := "test-admin-secret-must-be-32-chars-or-more"
	r, authRepo, _ := newAdminRouter(t, pg, jwtSecret)
	authHeader := newAdminUser(t, authRepo, jwtSecret, "dave")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/blog/session", nil)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	// 取出 fluxblog_token cookie 内容，用 BlogJWTSecret 能解析，
	// 用 admin secret 应解析失败。
	var tokenStr string
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "fluxblog_token" {
			tokenStr = ck.Value
		}
	}
	if tokenStr == "" {
		t.Fatalf("no fluxblog_token cookie")
	}
	if _, err := blog.ParseToken(tokenStr, testBlogSecret); err != nil {
		t.Errorf("ParseToken with BlogJWTSecret failed: %v", err)
	}
	if _, err := blog.ParseToken(tokenStr, jwtSecret); err == nil {
		t.Errorf("ParseToken with admin secret should fail (隔离), got nil err")
	}
}
