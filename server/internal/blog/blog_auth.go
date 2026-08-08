package blog

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// cookie 名：
//   - fluxblog_token：httpOnly，承载 JWT。SSR 私有页服务端读取后转发给 Go 私有 API。
//   - fluxblog_session：非 httpOnly，仅前端 JS 判断登录态（避免 JS 读 token 造成 XSS 泄露）。
const (
	blogTokenCookie   = "fluxblog_token"
	blogSessionCookie = "fluxblog_session"
	blogCookieMaxAge  = 7 * 24 * 60 * 60 // 7 天，与 refresh 宽限窗口一致
)

// requestToken 取请求中的 blog JWT：优先 Authorization: Bearer，回退 cookie。
// 保留 Bearer 支持，便于 SSR 私有页服务端转发、脚本调试。
func requestToken(c *gin.Context) string {
	if header := c.GetHeader("Authorization"); header != "" {
		if parts := strings.SplitN(header, " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}
	if v, err := c.Cookie(blogTokenCookie); err == nil && v != "" {
		return v
	}
	return ""
}

// isSecureRequest 判断是否 HTTPS（生产 nginx 终止 TLS，靠 X-Forwarded-Proto）。
// dev（http://localhost）返回 false，Secure cookie 才能写入浏览器。
func isSecureRequest(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

// SetSessionCookies 写入 token（httpOnly）与 session（JS 可读）两个 cookie。
// SameSite=Lax 缓解 CSRF；变更接口仅接受 application/json 进一步兜底。
// 导出供 admin SSO 端点复用，避免在 admin 包重写 cookie 逻辑造成漂移。
func SetSessionCookies(c *gin.Context, token string) {
	secure := isSecureRequest(c)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(blogTokenCookie, token, blogCookieMaxAge, "/", "", secure, true)
	c.SetCookie(blogSessionCookie, "1", blogCookieMaxAge, "/", "", secure, false)
}

// ClearSessionCookies 清除两个 cookie（注销）。
func ClearSessionCookies(c *gin.Context) {
	secure := isSecureRequest(c)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(blogTokenCookie, "", -1, "/", "", secure, true)
	c.SetCookie(blogSessionCookie, "", -1, "/", "", secure, false)
}

// AuthRequired 是 FluxBlog 独立鉴权中间件，与 finflow/admin 的
// middleware.AuthRequired 体系隔离：
//   - 用 blog JWT（iss=apppilot/aud=fluxblog）而非主 JWTSecret
//   - token 来源：Authorization: Bearer 或 fluxblog_token cookie
//   - 每次请求都查库校验账号 is_enabled 与 token_version
//   - 软删除/停用账号令牌立即失效（token_version 已递增）
//
// 失败一律 401（停用也按 401，避免泄露账号状态）。
func AuthRequired(repo *Repository, secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := requestToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims, err := ParseToken(token, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		u, err := repo.FindActiveByID(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		if !u.IsEnabled || u.TokenVersion != claims.TokenVersion {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("blogUserID", u.ID)
		c.Set("blogUsername", u.Username)
		c.Next()
	}
}
