package blog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestSetSessionCookies_WritesBothCookies 验证 token 与 session cookie 同时写入，
// token httpOnly、session JS 可读；同时 Secure 标志按 X-Forwarded-Proto 翻转。
func TestSetSessionCookies_WritesBothCookies(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("X-Forwarded-Proto", "https")

	SetSessionCookies(c, "the-token")

	var tokenFound, sessionFound bool
	var tokenSecure, sessionSecure bool
	for _, ck := range w.Result().Cookies() {
		switch ck.Name {
		case blogTokenCookie:
			tokenFound = true
			tokenSecure = ck.Secure
			if ck.Value != "the-token" {
				t.Errorf("token cookie value = %q, want the-token", ck.Value)
			}
			if !ck.HttpOnly {
				t.Errorf("token cookie must be HttpOnly")
			}
		case blogSessionCookie:
			sessionFound = true
			sessionSecure = ck.Secure
			if ck.HttpOnly {
				t.Errorf("session cookie must NOT be HttpOnly (前端 JS 判登录态用)")
			}
		}
	}
	if !tokenFound {
		t.Errorf("fluxblog_token cookie not written")
	}
	if !sessionFound {
		t.Errorf("fluxblog_session cookie not written")
	}
	if !tokenSecure || !sessionSecure {
		t.Errorf("Secure flag should be true under HTTPS (X-Forwarded-Proto=https): token=%v session=%v", tokenSecure, sessionSecure)
	}
}

// TestSetSessionCookies_HttpNotSecure dev 环境（http://localhost）不应置 Secure，
// 否则浏览器拒绝写入。
func TestSetSessionCookies_HttpNotSecure(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil) // 无 X-Forwarded-Proto

	SetSessionCookies(c, "tok")

	for _, ck := range w.Result().Cookies() {
		if ck.Secure {
			t.Errorf("cookie %q should NOT be Secure in dev http", ck.Name)
		}
	}
}

// TestClearSessionCookies_Invalidates 清除时 MaxAge<0 浏览器立即丢弃。
func TestClearSessionCookies_Invalidates(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	ClearSessionCookies(c)

	for _, ck := range w.Result().Cookies() {
		if ck.MaxAge >= 0 {
			t.Errorf("cookie %q MaxAge = %d, want < 0", ck.Name, ck.MaxAge)
		}
		if ck.Value != "" {
			t.Errorf("cookie %q value = %q, want empty", ck.Name, ck.Value)
		}
	}
}
