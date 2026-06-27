package admin

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"apppilot-server/internal/auth"
	"apppilot-server/internal/finflow"
)

type Handler struct {
	db        *sql.DB
	authRepo  *auth.Repository
	authH     *auth.Handler
	finflow   *finflow.Handler
}

func NewHandler(db *sql.DB, authRepo *auth.Repository, jwtSecret string) *Handler {
	return &Handler{
		db:       db,
		authRepo: authRepo,
		authH:    auth.NewHandler(authRepo, jwtSecret),
		finflow:  finflow.NewHandler(db),
	}
}

func (h *Handler) Register(rg *gin.RouterGroup, middlewares ...gin.HandlerFunc) {
	g := rg.Use(middlewares...)
	{
		g.GET("/users", h.listUsers)
		g.POST("/users", h.authH.CreateUser) // 复用 auth.Handler.CreateUser
		g.DELETE("/users/:id", h.deleteUser)
		g.GET("/users/:id/transactions", h.listUserTransactions)
		g.GET("/users/:id/categories", h.listUserCategories)
		g.GET("/users/:id/accounts", h.listUserAccounts)
		g.GET("/stats", h.stats)
	}
}

func (h *Handler) listUsers(c *gin.Context) {
	users, err := h.authRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *Handler) deleteUser(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	// 防止删除最后一个管理员
	u, err := h.authRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if u.Role == "admin" {
		admins, _ := h.authRepo.List()
		adminCount := 0
		for _, a := range admins {
			if a.Role == "admin" {
				adminCount++
			}
		}
		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete the last admin"})
			return
		}
	}
	if err := h.authRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listUserTransactions(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	// 复用 finflow repository，但用 URL 中的 user_id 而非 JWT 中的
	repo := finflow.NewRepository(h.db)
	txs, err := repo.ListTransactions(id, finflow.ListTxFilter{Limit: 200})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, txs)
}

func (h *Handler) listUserCategories(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	repo := finflow.NewRepository(h.db)
	cats, err := repo.ListCategories(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cats)
}

func (h *Handler) listUserAccounts(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	repo := finflow.NewRepository(h.db)
	accs, err := repo.ListAccounts(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, accs)
}

func (h *Handler) stats(c *gin.Context) {
	users, err := h.authRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	totalTx := 0
	adminCount, userCount := 0, 0
	for _, u := range users {
		if u.Role == "admin" {
			adminCount++
		} else {
			userCount++
		}
	}
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&totalTx)
	c.JSON(http.StatusOK, gin.H{
		"totalUsers":       len(users),
		"totalTransactions": totalTx,
		"admins":           adminCount,
		"regularUsers":     userCount,
	})
}

func parseIDParam(c *gin.Context, key string) (int64, bool) {
	s := c.Param(key)
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

var _ = errors.New
