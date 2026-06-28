package admin

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

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
		g.GET("/apps", h.listApps)
		g.GET("/users", h.listUsers)
		g.POST("/users", h.authH.CreateUser) // 复用 auth.Handler.CreateUser
		g.DELETE("/users/:id", h.deleteUser)
		g.GET("/users/:id/transactions", h.listUserTransactions)
		g.GET("/users/:id/categories", h.listUserCategories)
		g.GET("/users/:id/accounts", h.listUserAccounts)
		g.GET("/stats", h.stats)
	}
}

// listApps 返回所有接入应用的去重列表，从 users.app_scope 聚合，排除 "admin"。
func (h *Handler) listApps(c *gin.Context) {
	rows, err := h.db.Query(`SELECT DISTINCT unnest(app_scope) FROM users`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	apps := []string{}
	for rows.Next() {
		var app string
		if err := rows.Scan(&app); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if app == "admin" {
			continue
		}
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, apps)
}

func (h *Handler) listUsers(c *gin.Context) {
	app := c.Query("app")
	users, err := h.authRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	filtered := users[:0]
	for _, u := range users {
		if app != "" && !contains(u.AppScope, app) {
			continue
		}
		filtered = append(filtered, u)
	}

	// 聚合每个用户在当前应用下的交易数和最近活跃时间。
	// lastActiveAt 取 transactions.created_at 的最大值。
	type userStat struct {
		TxCount      int64  `json:"transactionCount"`
		LastActiveAt *string `json:"lastActiveAt"`
	}
	statsMap := map[int64]userStat{}
	if len(filtered) > 0 {
		ids := make([]any, len(filtered))
		for i, u := range filtered {
			ids[i] = u.ID
		}
		query := `SELECT user_id, COUNT(*), MAX(created_at)::text FROM transactions GROUP BY user_id`
		// transactions 表是 finflow 应用专属，无需再按 app 过滤；
		// 未来接入新应用时，每个应用一张业务表，此处需要按应用切换查询。
		rows, err := h.db.Query(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for rows.Next() {
			var uid int64
			var s userStat
			if err := rows.Scan(&uid, &s.TxCount, &s.LastActiveAt); err != nil {
				rows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			statsMap[uid] = s
		}
		rows.Close()
	}

	out := make([]gin.H, 0, len(filtered))
	for _, u := range filtered {
		s := statsMap[u.ID] // 零值也 OK
		out = append(out, gin.H{
			"id":        strconv.FormatInt(u.ID, 10),
			"username":  u.Username,
			"role":      u.Role,
			"appScope":  u.AppScope,
			"createdAt": u.CreatedAt,
			"updatedAt": u.UpdatedAt,
			"stats":     s,
		})
	}
	c.JSON(http.StatusOK, out)
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
	app := c.Query("app")
	users, err := h.authRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	totalTx := 0
	adminCount, userCount := 0, 0
	appUserIDs := []int64{}
	for _, u := range users {
		if app != "" && !contains(u.AppScope, app) {
			continue
		}
		if u.Role == "admin" {
			adminCount++
		} else {
			userCount++
		}
		appUserIDs = append(appUserIDs, u.ID)
	}

	// 该应用所有用户的交易数与本周活跃用户数。
	// transactions 表当前只服务于 finflow，未来接入新应用需按应用切换。
	activeThisWeek := 0
	if len(appUserIDs) > 0 {
		_ = h.db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&totalTx)
		// 本周活跃：本周内有过交易创建的用户数
		weekStart := nowWeekStart()
		err := h.db.QueryRow(
			`SELECT COUNT(DISTINCT user_id) FROM transactions WHERE created_at >= $1`,
			weekStart,
		).Scan(&activeThisWeek)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"totalUsers":       len(appUserIDs),
		"totalTransactions": totalTx,
		"admins":           adminCount,
		"regularUsers":     userCount,
		"activeThisWeek":   activeThisWeek,
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

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// nowWeekStart 返回本周一 00:00（本地时区）。
func nowWeekStart() time.Time {
	now := time.Now()
	days := int(now.Weekday()) - 1
	if days < 0 {
		days = 6
	}
	return time.Date(now.Year(), now.Month(), now.Day()-days, 0, 0, 0, 0, now.Location())
}

var _ = errors.New
