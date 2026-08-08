package admin

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"apppilot-server/internal/auth"
	"apppilot-server/internal/blog"
	"apppilot-server/internal/finflow"
)

type Handler struct {
	db            *sql.DB
	authRepo      *auth.Repository
	authH         *auth.Handler
	finflow       *finflow.Handler
	blogRepo      *blog.Repository
	blogJwtSecret string
}

func NewHandler(db *sql.DB, authRepo *auth.Repository, jwtSecret string, blogRepo *blog.Repository, blogJwtSecret string) *Handler {
	return &Handler{
		db:            db,
		authRepo:      authRepo,
		authH:         auth.NewHandler(authRepo, jwtSecret),
		finflow:       finflow.NewHandler(db),
		blogRepo:      blogRepo,
		blogJwtSecret: blogJwtSecret,
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

		// FluxBlog 账号管理已移除：SSO 端点 /admin/blog/session 会按需自动建 stub
		// 并下发 cookie，admin 无需也不应直接管理 blog_users。
		// 若需吊销某个 admin 的博客访问，改 SQL：UPDATE blog_users SET
		// is_enabled=false, token_version=token_version+1 WHERE username=...

		// admin SSO 进入 FluxBlog 写作后台：以当前 admin 用户名为 key
		// 查找或自动创建 blog_users stub（随机密码，admin 永不需要知道），
		// 签发 blog JWT 并下发 fluxblog_token/fluxblog_session cookie。
		// 调用方随后跳转 /blog/studio/，cookie 已就绪，无需二次登录。
		g.POST("/blog/session", h.startBlogSession)
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
		TxCount      int64   `json:"transactionCount"`
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
		"totalUsers":        len(appUserIDs),
		"totalTransactions": totalTx,
		"admins":            adminCount,
		"regularUsers":      userCount,
		"activeThisWeek":    activeThisWeek,
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

// ==================== FluxBlog 账号管理 ====================
// 独立于 finflow users：软删除、停用即时失效（token_version 递增）。

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// ==================== FluxBlog SSO ====================
// admin 用户进入博客写作后台的"免登录"入口：以 admin 用户名为 key
// 查找或自动创建 blog_users stub，签发 blog JWT 并下发 cookie。
// 隔离原则不破坏——admin 仍不能直接调 /blog/* 写操作（仍需 blog JWT），
// 此端点只是把 blog JWT 经 cookie 下发给浏览器，让 /blog/studio/ 走通。

// startBlogSession 把当前 admin 用户映射到 blog_users：
//   - 已存在且启用 → 直接签发
//   - 已存在但停用 → 403（admin 可在 /admin/blog-users 重新启用）
//   - 不存在 → 自动创建 stub（随机密码，admin 永不需要）再签发
//
// 写入的 fluxblog_token/fluxblog_session cookie 与正常 blog login 完全等价。
func (h *Handler) startBlogSession(c *gin.Context) {
	username, _ := c.Get("username")
	uname, _ := username.(string)
	if uname == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing admin identity"})
		return
	}

	u, err := h.blogRepo.FindByUsernameActive(uname)
	switch {
	case err == nil:
		// 命中已存在 stub
	case errors.Is(err, blog.ErrUserNotFound):
		// 自动创建 stub：随机 32 字节 hex 作密码，admin 永不需要
		pw, err := randomHex(32)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		u, err = h.blogRepo.Create(uname, pw)
		if err != nil {
			if errors.Is(err, blog.ErrUserExists) {
				// 并发：同 admin 用户名已被另一请求创建 → 重查
				u, err = h.blogRepo.FindByUsernameActive(uname)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			_ = h.blogRepo.InsertAudit(nil, "sso_create", u.Username)
		}
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !u.IsEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "blog account disabled"})
		return
	}

	token, _, err := blog.GenerateToken(u, h.blogJwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	blog.SetSessionCookies(c, token)
	_ = h.blogRepo.InsertAudit(&u.ID, "sso_session", "")

	c.JSON(http.StatusOK, gin.H{
		"redirect": "/blog/studio/",
		"username": u.Username,
	})
}

// randomHex 返回 n 字节的十六进制字符串（n=32 时为 64 字符）。
// 用于为自动创建的 blog_users stub 生成永不复用的随机密码。
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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
