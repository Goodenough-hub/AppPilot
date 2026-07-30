package blog

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthRequired 是 FluxBlog 独立鉴权中间件，与 finflow/admin 的
// middleware.AuthRequired 体系隔离：
//   - 用 blog JWT（iss=apppilot/aud=fluxblog）而非主 JWTSecret
//   - 每次请求都查库校验账号 is_enabled 与 token_version
//   - 软删除/停用账号令牌立即失效（token_version 已递增）
//
// 失败一律 401（停用也按 401，避免泄露账号状态）。
func AuthRequired(repo *Repository, secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid auth header"})
			return
		}
		claims, err := ParseToken(parts[1], secret)
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

var _ = errors.New
