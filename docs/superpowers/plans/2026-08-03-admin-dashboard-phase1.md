# Admin Dashboard 增强 — Phase 1 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Phase 1：analytics_events 表 + 埋点端点 + 前端埋点 SDK + 页面分析页

**Architecture:** 后端新增 analytics 包（track 端点公开、查询端点 admin 鉴权），前端新增 AnalyticsPage 展示 PV/UV 趋势和热门页面，各应用前端在路由切换时调用 track 端点 fire-and-forget

**Tech Stack:** Go + Gin + PostgreSQL (后端), React 19 + TypeScript + recharts (前端), 无新增依赖

## Global Constraints

- 所有数据源查询必须 admin 鉴权（`/admin/*` 路径由 `AuthRequired` + `AdminRequired` 保护）
- 埋点上报端点 `/api/v1/analytics/track` 为公开端点（不限速）
- 前端埋点 fire-and-forget：上报失败不阻塞主流程
- 提交前 `npm run typecheck` + `npm test` 必须通过（admin 前端），`go test ./...` 通过（后端）
- 遵循现有代码风格：admin 前端用 dark space theme + glass-panel + recharts，后端用 Gin + database/sql

---

## 文件结构

```
后端 (AppPilot/server/)
  internal/analytics/
    handler.go          — track 端点 + 查询端点（新增）
    repository.go       — analytics_events CRUD + 聚合查询（新增）
  internal/db/
    migrations.go       — 新增 analytics_events DDL（修改）
  cmd/
    cmd.go              — 注册 analytics 路由（修改）

前端 (AppPilot/admin/)
  src/api/
    analytics.ts        — track 函数 + 查询 API 封装（新增）
  src/pages/
    AnalyticsPage.tsx   — 页面分析页（新增）
  src/App.tsx           — 新增 /admin/analytics 路由（修改）
  src/components/
    Layout.tsx          — 侧边栏新增"页面分析"导航项（修改）

FinFlow 前端 (FinFlow/web/)
  src/App.tsx           — 路由切换时上报埋点（修改）
```

---

### Task 1: 数据库 — analytics_events 表

**Files:**
- Modify: `AppPilot/server/internal/db/migrations.go`

**Interfaces:**
- Produces: `analytics_events` 表可供 `internal/analytics/repository.go` 查询

- [ ] **Step 1: 在 migrations.go 的 Migrate 函数末尾添加 analytics_events DDL**

在 `AppPilot/server/internal/db/migrations.go` 的 `Migrate` 函数中，在 `return MigrateBlog(db)` 之前插入：

```go
// 管理后台页面分析：前端埋点事件表
if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS analytics_events (
    id          BIGSERIAL PRIMARY KEY,
    app         TEXT NOT NULL,
    user_id     BIGINT,
    event_type  TEXT NOT NULL,
    path        TEXT NOT NULL,
    title       TEXT,
    referrer    TEXT,
    user_agent  TEXT,
    ip          TEXT,
    session_id  TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ae_app_time ON analytics_events(app, created_at);
CREATE INDEX IF NOT EXISTS idx_ae_path ON analytics_events(path);
CREATE INDEX IF NOT EXISTS idx_ae_event_type ON analytics_events(app, event_type, created_at);
`); err != nil {
    return err
}
```

- [ ] **Step 2: 运行 go test 验证迁移不报错**

```bash
cd AppPilot/server && go test ./internal/db/... -v -run TestMigrate
```

预计：已有的迁移测试通过（如果有的话），无新增错误。

- [ ] **Step 3: Commit**

```bash
git add AppPilot/server/internal/db/migrations.go
git commit -m "feat(admin): add analytics_events table for pageview tracking"
```

---

### Task 2: 后端 — analytics repository

**Files:**
- Create: `AppPilot/server/internal/analytics/repository.go`

**Interfaces:**
- Consumes: `*sql.DB` (from `db.NewPostgres`)
- Produces:
  - `func NewRepository(db *sql.DB) *Repository`
  - `func (r *Repository) InsertEvent(app, eventType, path, title, referrer, userAgent, ip, sessionID string, userID *int64) error`
  - `func (r *Repository) PVAggregate(app string, start, end time.Time) ([]PVDailyRow, error)` — 按日聚合 PV/UV
  - `func (r *Repository) TopPages(app string, start, end time.Time, limit int) ([]TopPageRow, error)` — 热门页面排行
  - `func (r *Repository) RealtimeUsers(app string, window time.Duration) (int, error)` — 实时在线

- [ ] **Step 1: 创建 repository.go**

```go
package analytics

import (
	"database/sql"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type PVDailyRow struct {
	Date string `json:"date"`
	PV   int64  `json:"pv"`
	UV   int64  `json:"uv"`
}

type TopPageRow struct {
	Path string `json:"path"`
	PV   int64  `json:"pv"`
	UV   int64  `json:"uv"`
}

// InsertEvent 写入一条埋点事件。userID 可为 nil（匿名访问）。
func (r *Repository) InsertEvent(app, eventType, path, title, referrer, userAgent, ip, sessionID string, userID *int64) error {
	_, err := r.db.Exec(
		`INSERT INTO analytics_events (app, user_id, event_type, path, title, referrer, user_agent, ip, session_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		app, userID, eventType, path, title, referrer, userAgent, ip, sessionID,
	)
	return err
}

// PVAggregate 按日聚合指定时间范围内的 PV 和 UV。
// UV 按 session_id 去重（无 session 时按 ip 兜底）。
func (r *Repository) PVAggregate(app string, start, end time.Time) ([]PVDailyRow, error) {
	rows, err := r.db.Query(
		`SELECT to_char(created_at, 'YYYY-MM-DD') AS date,
		        COUNT(*) AS pv,
		        COUNT(DISTINCT COALESCE(session_id, ip)) AS uv
		   FROM analytics_events
		  WHERE app = $1 AND event_type = 'pageview'
		    AND created_at >= $2 AND created_at < $3
		  GROUP BY date ORDER BY date`,
		app, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PVDailyRow{}
	for rows.Next() {
		var r PVDailyRow
		if err := rows.Scan(&r.Date, &r.PV, &r.UV); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// TopPages 返回指定时间范围内热门页面前 N 名。
func (r *Repository) TopPages(app string, start, end time.Time, limit int) ([]TopPageRow, error) {
	rows, err := r.db.Query(
		`SELECT path, COUNT(*) AS pv, COUNT(DISTINCT COALESCE(session_id, ip)) AS uv
		   FROM analytics_events
		  WHERE app = $1 AND event_type = 'pageview'
		    AND created_at >= $2 AND created_at < $3
		  GROUP BY path ORDER BY pv DESC LIMIT $4`,
		app, start, end, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TopPageRow{}
	for rows.Next() {
		var r TopPageRow
		if err := rows.Scan(&r.Path, &r.PV, &r.UV); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RealtimeUsers 返回近 window 时间内有活动的独立用户数。
func (r *Repository) RealtimeUsers(app string, window time.Duration) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(DISTINCT COALESCE(session_id, ip))
		   FROM analytics_events
		  WHERE app = $1 AND event_type = 'pageview'
		    AND created_at >= NOW() - $2::interval`,
		app, window.String(),
	).Scan(&count)
	return count, err
}
```

- [ ] **Step 2: 运行 go vet 确认无语法错误**

```bash
cd AppPilot/server && go vet ./internal/analytics/...
```

- [ ] **Step 3: Commit**

```bash
git add AppPilot/server/internal/analytics/repository.go
git commit -m "feat(admin): add analytics repository for event insert and aggregation"
```

---

### Task 3: 后端 — analytics handler + 路由注册

**Files:**
- Create: `AppPilot/server/internal/analytics/handler.go`
- Modify: `AppPilot/server/cmd/cmd.go`

**Interfaces:**
- Consumes: `*Repository` (from Task 2), `*gin.RouterGroup` (from `cmd.go`)
- Produces:
  - `func NewHandler(repo *Repository) *Handler`
  - `func (h *Handler) RegisterPublic(rg *gin.RouterGroup)` — 公开 track 端点
  - `func (h *Handler) RegisterAdmin(rg *gin.RouterGroup, middlewares ...gin.HandlerFunc)` — admin 鉴权查询端点

- [ ] **Step 1: 创建 handler.go**

```go
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

func (h *Handler) track(c *gin.Context) {
	var req TrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 尝试从 JWT 获取 userID（若已登录），匿名亦可
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
```

- [ ] **Step 2: 在 cmd.go 注册路由**

在 `AppPilot/server/cmd/cmd.go` 的 `serve` 函数中，在 `import` 块添加 `"apppilot-server/internal/analytics"`，然后在 `adminHandler := admin.NewHandler(...)` 之前插入：

```go
// 埋点：公开 track 端点 + admin 鉴权的分析查询
analyticsRepo := analytics.NewRepository(pg)
analyticsHandler := analytics.NewHandler(analyticsRepo)
analyticsHandler.RegisterPublic(v1.Group("/analytics"))
analyticsHandler.RegisterAdmin(
    v1.Group("/admin"),
    middleware.AuthRequired(cfg.JWTSecret),
    middleware.AdminRequired(),
)
```

同时需要更新 import 块，添加 `"apppilot-server/internal/analytics"`。

- [ ] **Step 3: 运行 go vet 和测试**

```bash
cd AppPilot/server && go vet ./... && go test ./...
```

- [ ] **Step 4: Commit**

```bash
git add AppPilot/server/internal/analytics/handler.go AppPilot/server/cmd/cmd.go
git commit -m "feat(admin): add analytics track endpoint and admin query endpoints"
```

---

### Task 4: 前端 — analytics API 封装

**Files:**
- Create: `AppPilot/admin/src/api/analytics.ts`

**Interfaces:**
- Consumes: `apiClient` from `./client`
- Produces:
  - `track(req: TrackRequest): Promise<void>` — fire-and-forget 埋点上报
  - `getPV(params): Promise<PVDailyRow[]>` — PV/UV 聚合
  - `getTopPages(params): Promise<TopPageRow[]>` — 热门页面
  - `getRealtime(params): Promise<number>` — 实时在线

- [ ] **Step 1: 创建 analytics.ts**

```ts
import { apiClient } from './client'

export interface TrackRequest {
  app: string
  eventType: string
  path: string
  title?: string
  referrer?: string
  sessionId?: string
}

export interface PVDailyRow {
  date: string
  pv: number
  uv: number
}

export interface TopPageRow {
  path: string
  pv: number
  uv: number
}

export interface PVAggregateParams {
  app: string
  start?: string  // RFC3339
  end?: string    // RFC3339
  limit?: number
}

function buildParams(params: PVAggregateParams): Record<string, string> {
  const p: Record<string, string> = { app: params.app }
  if (params.start) p.start = params.start
  if (params.end) p.end = params.end
  if (params.limit) p.limit = String(params.limit)
  return p
}

/** 埋点上报（fire-and-forget，失败不抛异常） */
export function track(req: TrackRequest): void {
  apiClient.post('/analytics/track', req).catch(() => {
    // fire-and-forget: 上报失败不影响主流程
  })
}

/** 获取 PV/UV 日聚合数据 */
export async function getPV(params: PVAggregateParams): Promise<PVDailyRow[]> {
  const { data } = await apiClient.get<PVDailyRow[]>('/admin/analytics/pv', {
    params: buildParams(params),
  })
  return data
}

/** 获取热门页面排行 */
export async function getTopPages(params: PVAggregateParams): Promise<TopPageRow[]> {
  const { data } = await apiClient.get<TopPageRow[]>('/admin/analytics/top-pages', {
    params: buildParams(params),
  })
  return data
}

/** 获取实时在线用户数（近5分钟） */
export async function getRealtime(app: string): Promise<number> {
  const { data } = await apiClient.get<{ online: number }>('/admin/analytics/realtime', {
    params: { app },
  })
  return data.online
}

/** 生成会话 ID（浏览器 session 级别，同标签页共享） */
export function getSessionId(): string {
  const key = 'apppilot_analytics_sid'
  let sid = sessionStorage.getItem(key)
  if (!sid) {
    sid = crypto.randomUUID()
    sessionStorage.setItem(key, sid)
  }
  return sid
}
```

- [ ] **Step 2: 运行 typecheck 确认无类型错误**

```bash
cd AppPilot/admin && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add AppPilot/admin/src/api/analytics.ts
git commit -m "feat(admin): add analytics API client for track and query endpoints"
```

---

### Task 5: 前端 — AnalyticsPage 页面分析页

**Files:**
- Create: `AppPilot/admin/src/pages/AnalyticsPage.tsx`
- Modify: `AppPilot/admin/src/App.tsx`
- Modify: `AppPilot/admin/src/components/Layout.tsx`

**Interfaces:**
- Consumes: `getPV`, `getTopPages`, `getRealtime` from `../api/analytics`, `listApps` from `../api/admin`
- Produces: `<AnalyticsPage />` 组件，路由挂载在 `/admin/analytics`

- [ ] **Step 1: 创建 AnalyticsPage.tsx**

```tsx
import { useEffect, useState, useCallback } from 'react'
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, BarChart, Bar, Cell } from 'recharts'
import { listApps } from '../api/admin'
import { getPV, getTopPages, getRealtime, type PVDailyRow, type TopPageRow } from '../api/analytics'
import StatCard from '../components/ui/StatCard'

const AXIS = '#71717A'
const GRID = 'rgba(255,255,255,0.06)'
const BLUE = '#818CF8'
const PURPLE = '#A855F7'
const EMERALD = '#10B981'
const AMBER = '#F59E0B'

const tooltipStyle = {
  background: 'rgba(9, 9, 11, 0.95)',
  border: '1px solid rgba(255,255,255,0.1)',
  borderRadius: 12,
  fontSize: 13,
  color: '#fff',
}

const RANGES: { label: string; days: number }[] = [
  { label: '今天', days: 0 },
  { label: '7 天', days: 7 },
  { label: '30 天', days: 30 },
]

export default function AnalyticsPage() {
  const [apps, setApps] = useState<string[]>([])
  const [app, setApp] = useState<string>('')
  const [days, setDays] = useState(7)
  const [pvData, setPvData] = useState<PVDailyRow[]>([])
  const [topPages, setTopPages] = useState<TopPageRow[]>([])
  const [online, setOnline] = useState(0)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  // 加载应用列表
  useEffect(() => {
    listApps()
      .then(list => {
        setApps(list)
        if (list.length > 0 && !app) setApp(list[0])
      })
      .catch(err => setError(err.response?.data?.error || '加载应用列表失败'))
  }, [])

  // 加载数据
  const load = useCallback(async () => {
    if (!app) return
    setLoading(true)
    setError('')
    try {
      const end = new Date().toISOString()
      const start = new Date(Date.now() - days * 86400000).toISOString()
      const [pv, top, rt] = await Promise.all([
        getPV({ app, start, end }),
        getTopPages({ app, start, end, limit: 20 }),
        getRealtime(app),
      ])
      setPvData(pv)
      setTopPages(top)
      setOnline(rt)
    } catch (err: any) {
      setError(err.response?.data?.error || '加载失败')
    } finally {
      setLoading(false)
    }
  }, [app, days])

  useEffect(() => { load() }, [load])

  const totalPV = pvData.reduce((s, d) => s + d.pv, 0)
  const totalUV = pvData.reduce((s, d) => s + d.uv, 0)
  const avgPV = totalUV > 0 ? (totalPV / totalUV).toFixed(1) : '0'

  return (
    <div className="animate-fade-in-up">
      <header className="admin-page-header">
        <div>
          <h1>页面分析</h1>
          <div className="subtitle">跨应用页面浏览统计</div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ color: 'var(--text-secondary)', fontSize: 13, fontWeight: 500 }}>应用</span>
          <select value={app} onChange={e => setApp(e.target.value)} style={{ width: 160 }}>
            {apps.map(a => <option key={a} value={a}>{a}</option>)}
          </select>
        </div>
      </header>

      {error && (
        <div style={{
          color: '#FCA5A5', background: 'var(--danger-bg)', padding: '12px 16px',
          borderRadius: 12, marginBottom: 24, fontSize: 14,
          border: '1px solid rgba(239, 68, 68, 0.2)'
        }}>{error}</div>
      )}

      {/* 概览卡片 */}
      <div className="stat-grid" style={{ gridTemplateColumns: 'repeat(4, 1fr)' }}>
        <StatCard label="PV" value={totalPV} gradient="linear-gradient(135deg, #818CF8, #6366F1)" glow="#818CF8" gradientValue className="animate-fade-in-up stagger-1" />
        <StatCard label="UV" value={totalUV} gradient="linear-gradient(135deg, #A855F7, #7C3AED)" glow="#A855F7" gradientValue className="animate-fade-in-up stagger-2" />
        <StatCard label="人均页面数" value={avgPV} gradient="linear-gradient(135deg, #10B981, #059669)" glow="#10B981" gradientValue className="animate-fade-in-up stagger-3" />
        <StatCard label="实时在线" value={online} gradient="linear-gradient(135deg, #F59E0B, #D97706)" glow="#F59E0B" gradientValue className="animate-fade-in-up stagger-4" />
      </div>

      {/* 时间范围 */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 24 }}>
        {RANGES.map(r => (
          <button
            key={r.days}
            onClick={() => setDays(r.days)}
            className={days === r.days ? 'primary' : ''}
            style={{ padding: '8px 16px', fontSize: 13 }}
          >
            {r.label}
          </button>
        ))}
      </div>

      {/* PV/UV 趋势图 */}
      <div className="charts-row" style={{ marginBottom: 32 }}>
        <div className="glass-panel chart-card" style={{ gridColumn: '1 / -1' }}>
          <h2>PV/UV 趋势</h2>
          <div className="subtitle">页面浏览量 & 独立访客</div>
          {loading ? (
            <div style={{ height: 260, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-tertiary)' }}>加载中…</div>
          ) : (
            <ResponsiveContainer width="100%" height={260}>
              <AreaChart data={pvData} margin={{ top: 8, right: 8, left: -16, bottom: 0 }}>
                <defs>
                  <linearGradient id="pvFill" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={BLUE} stopOpacity={0.3} />
                    <stop offset="100%" stopColor={BLUE} stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="uvFill" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={PURPLE} stopOpacity={0.25} />
                    <stop offset="100%" stopColor={PURPLE} stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis dataKey="date" stroke={AXIS} fontSize={12} tickLine={false} axisLine={{ stroke: GRID }} />
                <YAxis stroke={AXIS} fontSize={12} tickLine={false} axisLine={false} allowDecimals={false} width={36} />
                <Tooltip contentStyle={tooltipStyle} cursor={{ stroke: GRID }} />
                <Area type="monotone" dataKey="pv" name="PV" stroke={BLUE} strokeWidth={2} fill="url(#pvFill)" animationDuration={280} />
                <Area type="monotone" dataKey="uv" name="UV" stroke={PURPLE} strokeWidth={2} fill="url(#uvFill)" animationDuration={280} />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>

      {/* 热门页面排行 */}
      <div className="glass-panel animate-fade-in-up stagger-4" style={{ overflow: 'hidden' }}>
        <div style={{ padding: '24px 32px' }}>
          <h2 style={{ fontSize: 20, margin: 0 }}>热门页面</h2>
        </div>
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>#</th>
                <th>页面路径</th>
                <th style={{ textAlign: 'right' }}>PV</th>
                <th style={{ textAlign: 'right' }}>UV</th>
              </tr>
            </thead>
            <tbody>
              {topPages.map((p, i) => (
                <tr key={p.path}>
                  <td style={{ color: 'var(--text-tertiary)', fontSize: 13, fontWeight: 600 }}>{i + 1}</td>
                  <td style={{ fontWeight: 500 }}>{p.path}</td>
                  <td style={{ textAlign: 'right', fontFamily: 'Outfit, sans-serif' }}>{p.pv}</td>
                  <td style={{ textAlign: 'right', fontFamily: 'Outfit, sans-serif' }}>{p.uv}</td>
                </tr>
              ))}
              {topPages.length === 0 && (
                <tr>
                  <td colSpan={4} style={{ textAlign: 'center', padding: 48, color: 'var(--text-tertiary)', fontSize: 14 }}>
                    暂无数据
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: 在 App.tsx 添加路由**

在 `AppPilot/admin/src/App.tsx` 顶部添加 import：
```tsx
import AnalyticsPage from './pages/AnalyticsPage'
```

在 `<Route element={<Layout />}>` 内部添加：
```tsx
<Route path="/admin/analytics" element={<AnalyticsPage />} />
```

- [ ] **Step 3: 在 Layout.tsx 添加导航项**

在 `AppPilot/admin/src/components/Layout.tsx` 的 `navItems` 数组中，在概览之后添加：
```tsx
{ to: '/admin/analytics', label: '页面分析', end: false, icon: Activity },
```

需要在顶部 import 中添加 `Activity`：
```tsx
import { LayoutDashboard, Users, PenLine, Activity, LogOut, type LucideIcon } from 'lucide-react'
```

- [ ] **Step 4: 运行 typecheck 和测试**

```bash
cd AppPilot/admin && npm run typecheck && npm test
```

- [ ] **Step 5: Commit**

```bash
git add AppPilot/admin/src/pages/AnalyticsPage.tsx AppPilot/admin/src/App.tsx AppPilot/admin/src/components/Layout.tsx
git commit -m "feat(admin): add AnalyticsPage with PV/UV trends and top pages"
```

---

### Task 6: 前端埋点集成 — FinFlow PWA

**Files:**
- Modify: `FinFlow/web/src/App.tsx`

**Interfaces:**
- Consumes: `track` from admin 的 analytics API（通过共享的 `/api/v1/analytics/track` 端点）
- FinFlow 前端需要创建自己的埋点工具函数，直接 POST 到 `/api/v1/analytics/track`

- [ ] **Step 1: 在 FinFlow/web/src 下创建埋点工具**

在 `FinFlow/web/src/api/` 下创建 `track.ts`（FinFlow 自己的 axios 实例）：

```ts
import { apiClient } from './client'

const SESSION_KEY = 'finflow_analytics_sid'

function getSessionId(): string {
  let sid = sessionStorage.getItem(SESSION_KEY)
  if (!sid) {
    sid = crypto.randomUUID()
    sessionStorage.setItem(SESSION_KEY, sid)
  }
  return sid
}

export function trackPageview(path: string, title?: string): void {
  apiClient.post('/analytics/track', {
    app: 'finflow',
    eventType: 'pageview',
    path,
    title: title || document.title,
    sessionId: getSessionId(),
  }).catch(() => {
    // fire-and-forget
  })
}
```

- [ ] **Step 2: 在 FinFlow 的 App.tsx 中监听路由变化**

在 `FinFlow/web/src/App.tsx` 中，在 `useLocation` 的 useEffect 中调用 `trackPageview`。

先查看当前 App.tsx 的路由结构：

```tsx
import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { trackPageview } from './api/track'

// 在 App 组件内：
const location = useLocation()

useEffect(() => {
  trackPageview(location.pathname)
}, [location])
```

- [ ] **Step 3: 运行 typecheck 和测试**

```bash
cd FinFlow/web && npm run typecheck && npm test
```

- [ ] **Step 4: Commit**

```bash
git add FinFlow/web/src/api/track.ts FinFlow/web/src/App.tsx
git commit -m "feat(analytics): add pageview tracking to FinFlow PWA"
```

---

### Task 7: 前端埋点集成 — FluxBlog（如有前端路由）

**Files:**
- 检查 FluxBlog 前端位置，如有则同样集成埋点

**注意：** FluxBlog 如果目前没有独立前端 SPA（只是通过 API 服务），则跳过此任务。

- [ ] **Step 1: 确认 FluxBlog 前端状态**

```bash
# 检查是否有 FluxBlog 前端代码
ls -la "$WORKSPACE_ROOT/FluxBlog/" 2>/dev/null || echo "FluxBlog 前端目录不存在"
```

如果 FluxBlog 前端存在且使用 React Router，则参照 Task 6 的模式集成埋点。如果不存在，此任务标记为跳过。

- [ ] **Step 2: 运行 typecheck 和测试（如适用）**

- [ ] **Step 3: Commit（如适用）**

---

### Task 8: 端到端验证

- [ ] **Step 1: 启动后端（本地 PG）**

```bash
cd AppPilot/server && APPPLOT_DSN="postgres://..." APPPLOT_JWT_SECRET="test" go run . serve
```

- [ ] **Step 2: 用 curl 测试埋点端点**

```bash
curl -X POST http://localhost:8080/api/v1/analytics/track \
  -H 'Content-Type: application/json' \
  -d '{"app":"finflow","eventType":"pageview","path":"/transactions","title":"账单","sessionId":"test-session-1"}'
# 预期: 204 No Content
```

- [ ] **Step 3: 用 curl 测试查询端点（需 admin JWT）**

```bash
# 先登录获取 token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"..."}' | jq -r '.token')

# 查询 PV 聚合
curl -s http://localhost:8080/api/v1/admin/analytics/pv?app=finflow \
  -H "Authorization: Bearer $TOKEN" | jq
# 预期: [{"date":"2026-08-03","pv":1,"uv":1}]

# 查询热门页面
curl -s http://localhost:8080/api/v1/admin/analytics/top-pages?app=finflow \
  -H "Authorization: Bearer $TOKEN" | jq
# 预期: [{"path":"/transactions","pv":1,"uv":1}]

# 查询实时在线
curl -s http://localhost:8080/api/v1/admin/analytics/realtime?app=finflow \
  -H "Authorization: Bearer $TOKEN" | jq
# 预期: {"online":1}
```

- [ ] **Step 4: 启动 admin 前端验证页面**

```bash
cd AppPilot/admin && npm run dev
```

打开 `http://localhost:5076/admin/analytics`，选择应用后确认能看到 PV/UV 趋势图和热门页面表。

- [ ] **Step 5: Commit（如有微调）**