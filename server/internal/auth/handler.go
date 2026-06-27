package auth

import (
	"errors"
	"net/http"

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
	})
}

func (h *Handler) refresh(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(int64)
	u, err := h.repo.FindByID(userID)
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
	})
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
	token, exp, _ := GenerateToken(u, h.jwtSecret)
	c.JSON(http.StatusCreated, TokenResponse{
		Token:     token,
		ExpiresAt: exp,
		UserID:    encodeID(u.ID),
		Role:      u.Role,
		AppScope:  u.AppScope,
		Username:  u.Username,
	})
}
