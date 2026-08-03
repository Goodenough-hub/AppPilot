package analytics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

type TrackRequest struct {
	App       string `json:"app" binding:"required"`
	EventType string `json:"eventType" binding:"required"`
	Path      string `json:"path" binding:"required"`
	Title     string `json:"title"`
	Referrer  string `json:"referrer"`
	SessionID string `json:"sessionId"`
}

// RegisterPublic 注册公开埋点端点（无需鉴权）。
func (h *Handler) RegisterPublic(rg *gin.RouterGroup) {
	rg.POST("/track", h.track)
}

// RegisterAdmin 注册 admin 鉴权的分析查询端点。
func (h *Handler) RegisterAdmin(rg *gin.RouterGroup, middlewares ...gin.HandlerFunc) {
	g := rg.Use(middlewares...)
	{
		g.GET("/analytics/pv", h.pvAggregate)
		g.GET("/analytics/top-pages", h.topPages)
		g.GET("/analytics/realtime", h.realtime)
	}
}

// knownApps 是允许上报埋点的应用白名单，与 admin.listApps 的聚合逻辑一致。
var knownApps = map[string]bool{"finflow": true, "fluxblog": true}

func (h *Handler) track(c *gin.Context) {
	var req TrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 仅允许已知应用上报，防止垃圾数据写入
	if !knownApps[req.App] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown app"})
		return
	}
	// 尝试从 JWT 获取 userID（若已登录），匿名亦可；
	// 公开端点可能没有 auth 中间件，但若中间件已解析则使用。
	var userID *int64
	if uid, exists := c.Get("userID"); exists {
		if id, ok := uid.(int64); ok {
			userID = &id
		}
	}
	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	referrer := req.Referrer
	if referrer == "" {
		referrer = c.GetHeader("Referer")
	}
	if err := h.repo.InsertEvent(req.App, req.EventType, req.Path, req.Title, referrer, ua, ip, req.SessionID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Status(http.StatusNoContent)
}

// pvAggregate 查询参数：app, start, end (RFC3339 日期), 默认近7天
func (h *Handler) pvAggregate(c *gin.Context) {
	app := c.Query("app")
	if app == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app required"})
		return
	}
	start, end, err := parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rows, err := h.repo.PVAggregate(app, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == nil {
		rows = []PVDailyRow{}
	}
	c.JSON(http.StatusOK, rows)
}

func (h *Handler) topPages(c *gin.Context) {
	app := c.Query("app")
	if app == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app required"})
		return
	}
	start, end, err := parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	rows, err := h.repo.TopPages(app, start, end, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == nil {
		rows = []TopPageRow{}
	}
	c.JSON(http.StatusOK, rows)
}

func (h *Handler) realtime(c *gin.Context) {
	app := c.Query("app")
	if app == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app required"})
		return
	}
	count, err := h.repo.RealtimeUsers(app, 5*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"online": count})
}

// parseDateRange 解析 start/end 查询参数，默认近7天。
func parseDateRange(c *gin.Context) (time.Time, time.Time, error) {
	now := time.Now()
	end := now
	if e := c.Query("end"); e != "" {
		t, err := time.Parse(time.RFC3339, e)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end = t
	}
	start := end.AddDate(0, 0, -7)
	if s := c.Query("start"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		start = t
	}
	return start, end, nil
}
