package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"apppilot-server/internal/auth"
)

func AuthRequired(jwtSecret string) gin.HandlerFunc {
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
		claims, err := auth.ParseToken(parts[1], jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("appScope", claims.AppScope)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// AuthOptional 解析 JWT（若存在），不强制要求。
// 解析成功时设置 userID/role/appScope/username，失败时静默继续（不 401）。
func AuthOptional(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.Next()
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Next()
			return
		}
		claims, err := auth.ParseToken(parts[1], jwtSecret)
		if err != nil {
			c.Next()
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("appScope", claims.AppScope)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin only"})
			return
		}
		c.Next()
	}
}

func AppScopeRequired(app string) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, _ := c.Get("appScope")
		scopes, _ := scope.([]string)
		for _, s := range scopes {
			if s == app || s == "admin" {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no access to this app"})
	}
}
