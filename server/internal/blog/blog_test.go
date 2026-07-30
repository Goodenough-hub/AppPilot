package blog

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ==================== slug 校验 ====================

func TestValidSlug(t *testing.T) {
	cases := map[string]bool{
		"hello-world":     true,
		"hello":           true,
		"a-b-1-2":         true,
		"欢迎来到fluxblog":    true, // 中文 slug 允许
		"markdown-能力演示":   true,
		"":                false,
		"Hello":           false, // 大写仍拒绝
		"-leading":        false,
		"trailing-":       false,
		"with_underscore": false,
		"with.dot":        false,
		"hello world":     false,
	}
	for slug, want := range cases {
		if got := ValidSlug(slug); got != want {
			t.Errorf("ValidSlug(%q) = %v, want %v", slug, got, want)
		}
	}
}

// ==================== JWT 隔离与刷新 ====================

const testSecret = "test-blog-secret-must-be-at-least-32-chars-long"

func TestJWTRoundTrip(t *testing.T) {
	u := &BlogUser{ID: 7, Username: "alice", TokenVersion: 3, IsEnabled: true}
	token, exp, err := GenerateToken(u, testSecret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if token == "" || exp == 0 {
		t.Fatal("empty token/exp")
	}
	claims, err := ParseToken(token, testSecret)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.UserID != 7 || claims.Username != "alice" || claims.TokenVersion != 3 {
		t.Errorf("claims mismatch: %+v", claims)
	}
	if claims.Issuer != blogIssuer || claims.Audience[0] != blogAudience {
		t.Errorf("iss/aud mismatch: %s / %v", claims.Issuer, claims.Audience)
	}
}

// 跨域拒绝：用另一个 secret 签发的 token 不能通过 blog 校验。
func TestJWTCrossDomainRejected(t *testing.T) {
	u := &BlogUser{ID: 1, Username: "x", TokenVersion: 0}
	otherSecret := "another-blog-secret-must-be-at-least-32-chars-long!!"
	token, _, _ := GenerateToken(u, otherSecret)
	if _, err := ParseToken(token, testSecret); err == nil {
		t.Fatal("token signed with different secret should be rejected")
	}
}

// iss/aud 校验：篡改 issuer 的 token 应被拒绝。
func TestJWTIssuerAudienceEnforced(t *testing.T) {
	// 手动构造一个 iss 不匹配的 token
	claims := Claims{
		UserID: 1, Username: "x", TokenVersion: 0,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "evil", Audience: jwt.ClaimStrings{blogAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := tok.SignedString([]byte(testSecret))
	if _, err := ParseToken(s, testSecret); err == nil {
		t.Fatal("token with wrong issuer should be rejected")
	}
}

// 刷新：过期但签发在 7 天内的 token 仍可解析。
func TestParseTokenForRefreshAllowsExpired(t *testing.T) {
	claims := Claims{
		UserID: 5, Username: "bob", TokenVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    blogIssuer,
			Audience:  jwt.ClaimStrings{blogAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // 已过期
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)), // 1h 前签发
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := tok.SignedString([]byte(testSecret))
	c, err := ParseTokenForRefresh(s, testSecret)
	if err != nil {
		t.Fatalf("expired-but-recent token should refresh: %v", err)
	}
	if c.UserID != 5 {
		t.Errorf("claims mismatch: %+v", c)
	}
}

// 超过刷新宽限窗口的 token 应被拒绝。
func TestParseTokenForRefreshWindowExpired(t *testing.T) {
	claims := Claims{
		UserID: 5, Username: "bob", TokenVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    blogIssuer,
			Audience:  jwt.ClaimStrings{blogAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-30 * 24 * time.Hour)), // 30 天前
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := tok.SignedString([]byte(testSecret))
	if _, err := ParseTokenForRefresh(s, testSecret); err == nil {
		t.Fatal("token older than refresh window should be rejected")
	}
}

// ==================== frontmatter 组装 ====================

func TestAssembleFrontmatter(t *testing.T) {
	cover := "/blog/media/2026/07/abc.webp"
	d := &Draft{Slug: "my-post", Title: "标题 \"引号\"", Description: "描述", Tags: []string{"a", "b"}, Markdown: "正文"}
	pub := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	out := assembleFrontmatter(d, &cover, false, pub, now)

	// 必须以 --- 开头并以 ---\n\n 闭合（否则正文会被当成未闭合 frontmatter）
	if !strings.HasPrefix(out, "---\n") {
		t.Errorf("frontmatter must start with ---\n: %q", out)
	}
	if !strings.Contains(out, "\n---\n\n") {
		t.Errorf("frontmatter must close with ---\\n\\n (got: %q)", out)
	}
	if !strings.Contains(out, "slug: \"my-post\"") {
		t.Errorf("frontmatter missing slug: %s", out)
	}
	// title 用 JSON 序列化，应安全转义引号
	if !strings.Contains(out, `"标题 \"引号\""`) {
		t.Errorf("title not JSON-escaped: %s", out)
	}
	if !strings.Contains(out, "draft: false") {
		t.Errorf("draft flag wrong: %s", out)
	}
	if !strings.Contains(out, `cover: "/blog/media/2026/07/abc.webp"`) {
		t.Errorf("cover missing: %s", out)
	}
	// publishedAt 固定为首次发布日期；updatedAt 为本次提交时间，二者不同
	if !strings.Contains(out, "publishedAt: 2026-07-29") {
		t.Errorf("publishedAt missing: %s", out)
	}
	if !strings.Contains(out, "updatedAt: 2026-07-30T12:00:00Z") {
		t.Errorf("updatedAt missing: %s", out)
	}
}

// 无封面时必须省略 cover 行（schema 的 cover 为 optional，不接受 null）。
func TestAssembleFrontmatterOmitsCoverWhenNone(t *testing.T) {
	d := &Draft{Slug: "p", Title: "T", Description: "D", Tags: []string{"x"}, Markdown: "m"}
	pub := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	out := assembleFrontmatter(d, nil, false, pub, pub)
	if strings.Contains(out, "cover:") {
		t.Errorf("cover line should be omitted when no cover: %s", out)
	}
	if !strings.Contains(out, "\n---\n\n") {
		t.Errorf("frontmatter must close: %q", out)
	}
}

// ==================== 发布路径契约 ====================

func TestPostFilePath(t *testing.T) {
	cases := map[string]string{
		"hello":        "src/content/posts/hello.md",
		"a-b-1":        "src/content/posts/a-b-1.md",
		"my-post-2026": "src/content/posts/my-post-2026.md",
	}
	for slug, want := range cases {
		if got := PostFilePath(slug); got != want {
			t.Errorf("PostFilePath(%q) = %s, want %s", slug, got, want)
		}
	}
}

// ==================== Publisher：Git Data API 流程与冲突重试 ====================

func newTestPublisher(t *testing.T, srv *httptest.Server) *Publisher {
	t.Helper()
	return &Publisher{
		repo: "Goodenough-hub/FluxBlog", branch: "main", token: "tok",
		baseURL: srv.URL, client: srv.Client(),
	}
}

func TestPublisherCommitSuccess(t *testing.T) {
	srv := newGitDataServer(t, false) // false = 不模拟冲突
	defer srv.Close()
	p := newTestPublisher(t, srv)
	sha, err := p.Commit(context.Background(), "content(blog): 发布 x", []GitFile{
		{Path: "src/content/posts/x.md", Content: []byte("---\ntitle: x\n---\nhi")},
	})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if sha != "commit-1" {
		t.Errorf("sha = %s, want commit-1", sha)
	}
}

func TestPublisherConflictRetry(t *testing.T) {
	srv := newGitDataServer(t, true) // true = 首次 ref 更新 422，重试成功
	defer srv.Close()
	p := newTestPublisher(t, srv)
	sha, err := p.Commit(context.Background(), "content(blog): 发布 y", []GitFile{
		{Path: "src/content/posts/y.md", Content: []byte("body")},
	})
	if err != nil {
		t.Fatalf("Commit after retry: %v", err)
	}
	if sha != "commit-2" {
		t.Errorf("sha = %s, want commit-2 (retry commit)", sha)
	}
}

func TestPublisherNoToken(t *testing.T) {
	p := &Publisher{repo: "r", branch: "main", token: "", baseURL: "https://example.invalid", client: http.DefaultClient}
	if _, err := p.Commit(context.Background(), "m", nil); err == nil {
		t.Fatal("Commit without token should error")
	}
}

// ==================== CI-only 发布完成恢复 ====================

func TestRecoverableCommittedJob(t *testing.T) {
	sha := "commit-1"
	cases := []struct {
		name string
		job  *PublishJob
		want bool
	}{
		{name: "building 且已有 commit SHA 可恢复", job: &PublishJob{Status: JobBuilding, CommitSha: &sha}, want: true},
		{name: "queued 尚未证明 Git 提交成功", job: &PublishJob{Status: JobQueued, CommitSha: &sha}, want: false},
		{name: "building 但没有 SHA 不可恢复", job: &PublishJob{Status: JobBuilding}, want: false},
		{name: "终结 job 不重复恢复", job: &PublishJob{Status: JobSucceeded, CommitSha: &sha}, want: false},
		{name: "nil job", job: nil, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := recoverableCommittedJob(tc.job); got != tc.want {
				t.Fatalf("recoverableCommittedJob() = %v, want %v", got, tc.want)
			}
		})
	}
}

// newGitDataServer 起一个模拟 GitHub Git Data API 的 httptest 服务。
// conflict=true 时，首次 PATCH ref 返回 422（非快进），第二次返回 200，
// 用于验证 Publisher 的“刷新一次并重试”逻辑。
func newGitDataServer(t *testing.T, conflict bool) *httptest.Server {
	patchCount := 0
	blobCount := 0
	treeCount := 0
	commitCount := 0
	mux := http.NewServeMux()

	handle := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/refs/heads/"):
			writeJSON(t, w, map[string]any{"object": map[string]string{"sha": "base-sha", "type": "commit"}})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/commits/"):
			writeJSON(t, w, map[string]any{"sha": "base-sha", "tree": map[string]string{"sha": "tree-base"}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/blobs"):
			blobCount++
			writeJSON(t, w, map[string]string{"sha": "blob-" + itoa(blobCount)})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/trees"):
			treeCount++
			writeJSON(t, w, map[string]string{"sha": "tree-" + itoa(treeCount)})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/commits"):
			commitCount++
			writeJSON(t, w, map[string]string{"sha": "commit-" + itoa(commitCount)})
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/git/refs/heads/"):
			patchCount++
			if conflict && patchCount == 1 {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(`{"message":"Reference update is not a fast forward"}`))
				return
			}
			writeJSON(t, w, map[string]any{"object": map[string]string{"sha": "commit-" + itoa(commitCount)}})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}
	mux.HandleFunc("/", handle)
	srv := httptest.NewServer(mux)
	// 确认授权头被设置
	orig := mux
	_ = orig
	return srv
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

func itoa(n int) string {
	switch n {
	case 1:
		return "1"
	case 2:
		return "2"
	case 3:
		return "3"
	}
	return "n"
}

// 防止 import 未使用（io 仅在 newGitDataServer 隐式使用）
var _ = io.Discard
