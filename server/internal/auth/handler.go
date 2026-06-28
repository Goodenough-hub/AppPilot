package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"apppilot-server/internal/db"
)

type Handler struct {
	repo      *Repository
	jwtSecret string
}

func NewHandler(repo *Repository, jwtSecret string) *Handler {
	return &Handler{repo: repo, jwtSecret: jwtSecret}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/login", h.login)
	rg.POST("/refresh", h.refresh)
	rg.PUT("/avatar", h.updateAvatar)
}

func (h *Handler) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.repo.FindByUsername(req.Username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.VerifyPassword(u, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	token, exp, err := GenerateToken(u, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, TokenResponse{
		Token:     token,
		ExpiresAt: exp,
		UserID:    encodeID(u.ID),
		Role:      u.Role,
		AppScope:  u.AppScope,
		Username:  u.Username,
		Avatar:    u.Avatar,
	})
}

func (h *Handler) refresh(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if header == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid auth header"})
		return
	}
	claims, err := ParseTokenForRefresh(parts[1], h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	u, err := h.repo.FindByID(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	token, exp, err := GenerateToken(u, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, TokenResponse{
		Token:     token,
		ExpiresAt: exp,
		UserID:    encodeID(u.ID),
		Role:      u.Role,
		AppScope:  u.AppScope,
		Username:  u.Username,
		Avatar:    u.Avatar,
	})
}

func (h *Handler) updateAvatar(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(int64)
	var req UpdateAvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 限制 avatar 长度（base64 图片约 100KB，预设头像 ID 几十字节）
	if len(req.Avatar) > 512*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar too large (max 512KB)"})
		return
	}
	if err := h.repo.UpdateAvatar(userID, req.Avatar); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"avatar": req.Avatar})
}

// CreateUser 由管理员通过 admin API 调用，不在 auth 路由组暴露
func (h *Handler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := NormalizeRole(req.Role)
	appScope := req.AppScope
	if appScope == nil {
		appScope = []string{"finflow"}
	}
	u, err := h.repo.Create(req.Username, req.Password, role, appScope)
	if err != nil {
		if errors.Is(err, ErrUserExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := db.SeedForUser(h.repo.db, u.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":        encodeID(u.ID),
		"username":  u.Username,
		"role":      u.Role,
		"appScope":  u.AppScope,
		"avatar":    u.Avatar,
		"createdAt": u.CreatedAt,
		"updatedAt": u.UpdatedAt,
	})
}
