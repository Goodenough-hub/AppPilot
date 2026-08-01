package blog

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ==================== visibility 解析 ====================

func TestParseVisibility(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantPub string // 期望 *Visibility 指向的值；"-" 表示期望 nil
	}{
		{"create public", `{"visibility":"public"}`, "public"},
		{"create private", `{"visibility":"private"}`, "private"},
		{"create omit", `{"slug":"x","title":"y"}`, "-"},
		{"create empty", `{"visibility":""}`, ""}, // 显式空串 → 非 nil 指针指向 ""
	}
	for _, tc := range cases {
		var req CreateDraftRequest
		if err := json.Unmarshal([]byte(tc.body), &req); err != nil {
			t.Fatalf("%s: unmarshal: %v", tc.name, err)
		}
		switch tc.wantPub {
		case "-":
			if req.Visibility != nil {
				t.Errorf("%s: visibility = %v, want nil", tc.name, *req.Visibility)
			}
		default:
			if req.Visibility == nil || *req.Visibility != tc.wantPub {
				got := "<nil>"
				if req.Visibility != nil {
					got = *req.Visibility
				}
				t.Errorf("%s: visibility = %s, want %s", tc.name, got, tc.wantPub)
			}
		}
	}

	// PublishRequest 可缺省 visibility（保持原可见性）。
	var pr PublishRequest
	if err := json.Unmarshal([]byte(`{}`), &pr); err != nil {
		t.Fatalf("publish empty unmarshal: %v", err)
	}
	if pr.Visibility != nil {
		t.Errorf("empty publish visibility = %v, want nil", *pr.Visibility)
	}
	if err := json.Unmarshal([]byte(`{"visibility":"public"}`), &pr); err != nil {
		t.Fatalf("publish unmarshal: %v", err)
	}
	if pr.Visibility == nil || *pr.Visibility != "public" {
		t.Fatalf("publish visibility not parsed")
	}
}

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
