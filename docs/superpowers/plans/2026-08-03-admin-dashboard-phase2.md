# Admin Dashboard 增强 — Phase 2 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Phase 2：dashboards/widgets 表 + CRUD API + DataSource 抽象 + 应用看板页（含拖拽网格布局）

**Architecture:** 后端新增 dashboard 包（dashboards/widgets CRUD + DataSource 接口 + 已注册数据源查询），前端新增 DashboardDetailPage（react-grid-layout 拖拽看板 + 编辑模式 + 添加 Widget 抽屉），侧边栏新增"应用看板"导航

**Tech Stack:** Go + Gin + PostgreSQL (后端), React 19 + TypeScript + recharts + react-grid-layout (前端)

## Global Constraints

- 所有 dashboard/widget CRUD 和 datasource 查询必须 admin 鉴权
- 提交前 `npm run typecheck` + `npm test` 必须通过（admin 前端），`go test ./...` 通过（后端）
- 遵循现有代码风格：admin 前端用 dark space theme + glass-panel + recharts，后端用 Gin + database/sql
- 新增前端依赖：`react-grid-layout`（拖拽网格布局）
- Git 身份须为 `Goodenough`（不得出现 `wwxq`）

---

## 文件结构

```
后端 (AppPilot/server/)
  internal/dashboard/
    handler.go          — dashboards/widgets CRUD + datasource 查询端点（新增）
    repository.go       — dashboards/widgets 表操作（新增）
    datasource.go       — DataSource 接口 + 注册表 + 已注册数据源（新增）
  internal/db/
    migrations.go       — 新增 dashboards/dashboard_widgets DDL（修改）
  cmd/
    cmd.go              — 注册 dashboard 路由（修改）

前端 (AppPilot/admin/)
  src/api/
    dashboard.ts        — dashboards/widgets/datasources API 封装（新增）
  src/pages/
    DashboardDetailPage.tsx  — 应用看板页（新增）
  src/components/
    WidgetCard.tsx      — 单个 widget 渲染组件（新增）
    AddWidgetDrawer.tsx — 添加 Widget 抽屉（新增）
  src/App.tsx           — 新增 /admin/dashboards/:app 路由（修改）
  src/components/
    Layout.tsx          — 侧边栏新增"应用看板"导航（修改）
  src/styles.css        — react-grid-layout 样式覆盖（修改）
```

---

### Task 1: 数据库 — dashboards + dashboard_widgets 表 + Seed 默认看板

**Files:**
- Modify: `AppPilot/server/internal/db/migrations.go`
- Create: `AppPilot/server/internal/db/seed_dashboards.go`

**Interfaces:**
- Produces: `dashboards` 表（app UNIQUE）、`dashboard_widgets` 表（含 grid 位置列）
- `SeedDashboards(db)` 函数：为每个已知 app 创建默认 dashboard + widgets

**Steps:**

- [ ] **Step 1: 在 migrations.go 添加 DDL**

在 `Migrate` 函数中，`analytics_events` 表之后、`return MigrateBlog(db)` 之前插入：

```go
if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS dashboards (
    id          BIGSERIAL PRIMARY KEY,
    app         TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dashboard_widgets (
    id           BIGSERIAL PRIMARY KEY,
    dashboard_id BIGINT NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,
    title        TEXT NOT NULL,
    data_source  TEXT NOT NULL,
    config       JSONB DEFAULT '{}',
    grid_x       INT NOT NULL DEFAULT 0,
    grid_y       INT NOT NULL DEFAULT 0,
    grid_w       INT NOT NULL DEFAULT 4,
    grid_h       INT NOT NULL DEFAULT 3,
    sort_order   INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dw_dashboard ON dashboard_widgets(dashboard_id, sort_order);
`); err != nil {
    return err
}
```

- [ ] **Step 2: 创建 seed_dashboards.go**

```go
package db

import "database/sql"

var defaultWidgets = map[string][]struct {
    Type       string
    Title      string
    DataSource string
    GridX      int
    GridY      int
    GridW      int
    GridH      int
    SortOrder  int
}{
    "finflow": {
        {Type: "stat", Title: "总交易数", DataSource: "finflow:summary", GridX: 0, GridY: 0, GridW: 3, GridH: 2, SortOrder: 0},
        {Type: "stat", Title: "本月收入", DataSource: "finflow:summary", GridX: 3, GridY: 0, GridW: 3, GridH: 2, SortOrder: 1},
        {Type: "stat", Title: "本月支出", DataSource: "finflow:summary", GridX: 6, GridY: 0, GridW: 3, GridH: 2, SortOrder: 2},
        {Type: "stat", Title: "活跃用户", DataSource: "finflow:summary", GridX: 9, GridY: 0, GridW: 3, GridH: 2, SortOrder: 3},
        {Type: "chart", Title: "交易趋势", DataSource: "finflow:daily_trend", GridX: 0, GridY: 2, GridW: 8, GridH: 4, SortOrder: 4},
        {Type: "chart", Title: "分类占比", DataSource: "finflow:category_breakdown", GridX: 8, GridY: 2, GridW: 4, GridH: 4, SortOrder: 5},
    },
    "fluxblog": {
        {Type: "stat", Title: "文章总数", DataSource: "fluxblog:summary", GridX: 0, GridY: 0, GridW: 3, GridH: 2, SortOrder: 0},
        {Type: "stat", Title: "作者数", DataSource: "fluxblog:summary", GridX: 3, GridY: 0, GridW: 3, GridH: 2, SortOrder: 1},
        {Type: "stat", Title: "本月发布", DataSource: "fluxblog:summary", GridX: 6, GridY: 0, GridW: 3, GridH: 2, SortOrder: 2},
        {Type: "stat", Title: "公开文章", DataSource: "fluxblog:summary", GridX: 9, GridY: 0, GridW: 3, GridH: 2, SortOrder: 3},
        {Type: "chart", Title: "发布趋势", DataSource: "fluxblog:post_trend", GridX: 0, GridY: 2, GridW: 8, GridH: 4, SortOrder: 4},
        {Type: "chart", Title: "作者活跃度", DataSource: "fluxblog:author_activity", GridX: 8, GridY: 2, GridW: 4, GridH: 4, SortOrder: 5},
    },
}

func SeedDashboards(db *sql.DB) error {
    for app, widgets := range defaultWidgets {
        var dashboardID int64
        err := db.QueryRow(
            `INSERT INTO dashboards (app, title, description)
             VALUES ($1, $2, $3)
             ON CONFLICT (app) DO UPDATE SET title = EXCLUDED.title, updated_at = NOW()
             RETURNING id`,
            app, app+" 看板", app+" 应用数据概览",
        ).Scan(&dashboardID)
        if err != nil {
            return err
        }
        // 仅当 dashboard 无 widget 时才插入默认 widget（避免重复 seed）
        var count int
        if err := db.QueryRow(`SELECT COUNT(*) FROM dashboard_widgets WHERE dashboard_id = $1`, dashboardID).Scan(&count); err != nil {
            return err
        }
        if count > 0 {
            continue
        }
        for _, w := range widgets {
            _, err := db.Exec(
                `INSERT INTO dashboard_widgets (dashboard_id, type, title, data_source, grid_x, grid_y, grid_w, grid_h, sort_order)
                 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
                dashboardID, w.Type, w.Title, w.DataSource, w.GridX, w.GridY, w.GridW, w.GridH, w.SortOrder,
            )
            if err != nil {
                return err
            }
        }
    }
    return nil
}
```

- [ ] **Step 3: 在 Migrate 中调用 SeedDashboards**

在 `migrations.go` 的 `Migrate` 函数末尾，`return MigrateBlog(db)` 之后添加：

```go
if err := SeedDashboards(db); err != nil {
    return err
}
```

- [ ] **Step 4: 运行 go vet + test**

```bash
cd AppPilot/server && go vet ./... && go test ./...
```

- [ ] **Step 5: Commit**

```bash
git add server/internal/db/migrations.go server/internal/db/seed_dashboards.go
git commit -m "feat(admin): add dashboards and dashboard_widgets tables with seed data"
```

---

### Task 2: 后端 — dashboard repository

**Files:**
- Create: `AppPilot/server/internal/dashboard/repository.go`

**Interfaces:**
- Consumes: `*sql.DB`
- Produces:
  - `func NewRepository(db *sql.DB) *Repository`
  - `func (r *Repository) ListDashboards() ([]Dashboard, error)`
  - `func (r *Repository) GetDashboard(id int64) (*Dashboard, error)`
  - `func (r *Repository) GetDashboardByApp(app string) (*Dashboard, error)`
  - `func (r *Repository) UpdateDashboard(id int64, title, description *string) (*Dashboard, error)`
  - `func (r *Repository) ListWidgets(dashboardID int64) ([]Widget, error)`
  - `func (r *Repository) CreateWidget(dashboardID int64, w Widget) (*Widget, error)`
  - `func (r *Repository) UpdateWidget(id int64, w Widget) (*Widget, error)`
  - `func (r *Repository) DeleteWidget(id int64) error`
  - `func (r *Repository) UpdateWidgetLayout(id int64, gridX, gridY, gridW, gridH int) error`

Types:
```go
type Dashboard struct {
    ID          int64     `json:"id,string"`
    App         string    `json:"app"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"createdAt"`
    UpdatedAt   time.Time `json:"updatedAt"`
}

type Widget struct {
    ID          int64           `json:"id,string"`
    DashboardID int64           `json:"dashboardId,string"`
    Type        string          `json:"type"`
    Title       string          `json:"title"`
    DataSource  string          `json:"dataSource"`
    Config      json.RawMessage `json:"config"`
    GridX       int             `json:"gridX"`
    GridY       int             `json:"gridY"`
    GridW       int             `json:"gridW"`
    GridH       int             `json:"gridH"`
    SortOrder   int             `json:"sortOrder"`
    CreatedAt   time.Time       `json:"createdAt"`
    UpdatedAt   time.Time       `json:"updatedAt"`
}
```

- [ ] **Step 1: 创建 repository.go（完整实现）**

实现所有 CRUD 方法。关键点：
- `ListDashboards` 左连接 `dashboard_widgets` 计数，返回 `[]Dashboard`
- `GetDashboard` 返回单个 dashboard + 关联 widgets
- `UpdateDashboard` 更新 title/description，`updated_at = NOW()`
- `CreateWidget` / `UpdateWidget` / `DeleteWidget` 标准 CRUD
- `UpdateWidgetLayout` 仅更新 grid_x/grid_y/grid_w/grid_h

- [ ] **Step 2: 运行 go vet**

```bash
cd AppPilot/server && go vet ./internal/dashboard/...
```

- [ ] **Step 3: Commit**

```bash
git add server/internal/dashboard/repository.go
git commit -m "feat(admin): add dashboard repository with CRUD operations"
```

---

### Task 3: 后端 — DataSource 接口 + 已注册数据源

**Files:**
- Create: `AppPilot/server/internal/dashboard/datasource.go`

**Interfaces:**
- Consumes: `*sql.DB` (for finflow/fluxblog queries)
- Produces:
  - `type DataSource interface { Key() string; Description() string; Query(ctx context.Context, params map[string]any) ([]ChartData, error) }`
  - `type ChartData struct { Label string; Value float64; Extra map[string]any }`
  - `func NewRegistry(db *sql.DB) *Registry`
  - `func (r *Registry) List() []DataSourceMeta`
  - `func (r *Registry) Query(key string, params map[string]any) ([]ChartData, error)`

- [ ] **Step 1: 创建 datasource.go**

定义 DataSource 接口和 Registry。内置实现以下数据源（Phase 2 先做简单实现，Phase 3 丰富）：

**finflow:summary** — 返回 FinFlow 核心指标（总交易数、本月收入、本月支出、活跃用户数）
```go
// 查询 transactions 表：COUNT(*), SUM(income), SUM(expense), COUNT(DISTINCT user_id)
```

**finflow:daily_trend** — 近30天每日收入/支出趋势
```go
// SELECT date, SUM(CASE WHEN type='income' THEN amount ELSE 0 END), SUM(CASE WHEN type='expense' THEN amount ELSE 0 END) FROM transactions WHERE date >= $1 GROUP BY date ORDER BY date
```

**finflow:category_breakdown** — 本月支出分类占比（Top 10）
```go
// SELECT c.name, SUM(t.amount) FROM transactions t JOIN categories c ON c.id = t.category_id WHERE t.type='expense' AND t.date >= $1 GROUP BY c.name ORDER BY SUM DESC LIMIT 10
```

**fluxblog:summary** — FluxBlog 核心指标（文章总数、作者数、本月发布、公开文章数）
```go
// SELECT COUNT(*), COUNT(DISTINCT user_id), COUNT(*) FILTER (WHERE published_at >= $1), COUNT(*) FILTER (WHERE visibility='public') FROM blog_drafts
```

**fluxblog:post_trend** — 近12个月发布趋势
```go
// SELECT to_char(published_at,'YYYY-MM'), COUNT(*) FROM blog_drafts WHERE status='published' AND published_at >= $1 GROUP BY 1 ORDER BY 1
```

**fluxblog:author_activity** — 作者活跃度排行（Top 10）
```go
// SELECT bu.username, COUNT(*) FROM blog_drafts d JOIN blog_users bu ON bu.id = d.user_id WHERE d.status='published' GROUP BY bu.username ORDER BY COUNT DESC LIMIT 10
```

`ChartData` 结构：
```go
type ChartData struct {
    Label string         `json:"label"`
    Value float64        `json:"value"`
    Extra map[string]any `json:"extra,omitempty"`
}
```

- [ ] **Step 2: 运行 go vet**

```bash
cd AppPilot/server && go vet ./internal/dashboard/...
```

- [ ] **Step 3: Commit**

```bash
git add server/internal/dashboard/datasource.go
git commit -m "feat(admin): add DataSource interface and built-in data sources"
```

---

### Task 4: 后端 — dashboard handler + 路由注册

**Files:**
- Create: `AppPilot/server/internal/dashboard/handler.go`
- Modify: `AppPilot/server/cmd/cmd.go`

**Interfaces:**
- Consumes: `*Repository`, `*Registry` (from Tasks 2-3)
- Produces: HTTP handler with `RegisterAdmin(rg, middlewares...)` method

**Endpoints (all admin-authed):**

| Method | Path | Handler |
|---|---|---|
| GET | `/admin/dashboards` | `listDashboards` |
| GET | `/admin/dashboards/:id` | `getDashboard` |
| PUT | `/admin/dashboards/:id` | `updateDashboard` |
| POST | `/admin/dashboards/:id/widgets` | `createWidget` |
| PUT | `/admin/dashboards/:id/widgets/:wid` | `updateWidget` |
| DELETE | `/admin/dashboards/:id/widgets/:wid` | `deleteWidget` |
| GET | `/admin/datasources` | `listDataSources` |
| POST | `/admin/datasources/:key/query` | `queryDataSource` |

- [ ] **Step 1: 创建 handler.go**

关键实现：
- `listDashboards` — 调 `repo.ListDashboards()`，返回 JSON 数组
- `getDashboard` — 调 `repo.GetDashboard(id)` + `repo.ListWidgets(id)`，返回 `{dashboard, widgets}`
- `updateDashboard` — 绑定 `{title?, description?}`，调 `repo.UpdateDashboard`
- `createWidget` — 绑定 widget JSON，调 `repo.CreateWidget`
- `updateWidget` — 绑定 widget JSON，调 `repo.UpdateWidget`
- `deleteWidget` — 调 `repo.DeleteWidget`
- `listDataSources` — 调 `registry.List()`，返回数据源列表
- `queryDataSource` — 从 URL param `key` 取数据源 key（`:` 转 `/`），调 `registry.Query(key, params)`

- [ ] **Step 2: 在 cmd.go 注册路由**

在 `serve()` 函数中，`adminHandler.Register(...)` 之后添加：

```go
dashboardRepo := dashboard.NewRepository(pg)
dsRegistry := dashboard.NewRegistry(pg)
dashboardHandler := dashboard.NewHandler(dashboardRepo, dsRegistry)
dashboardHandler.RegisterAdmin(
    v1.Group("/admin"),
    middleware.AuthRequired(cfg.JWTSecret),
    middleware.AdminRequired(),
)
```

- [ ] **Step 3: 运行 go vet + test**

```bash
cd AppPilot/server && go vet ./... && go test ./...
```

- [ ] **Step 4: Commit**

```bash
git add server/internal/dashboard/handler.go server/cmd/cmd.go
git commit -m "feat(admin): add dashboard CRUD and datasource query endpoints"
```

---

### Task 5: 前端 — dashboard API 客户端

**Files:**
- Create: `AppPilot/admin/src/api/dashboard.ts`

**Interfaces:**
- Consumes: `apiClient` from `./client`
- Produces: TypeScript types + API functions for all dashboard endpoints

```ts
export interface Dashboard {
  id: string
  app: string
  title: string
  description: string
  createdAt: string
  updatedAt: string
}

export interface Widget {
  id: string
  dashboardId: string
  type: 'stat' | 'chart' | 'table'
  title: string
  dataSource: string
  config: Record<string, any>
  gridX: number
  gridY: number
  gridW: number
  gridH: number
  sortOrder: number
}

export interface DataSourceMeta {
  key: string
  description: string
}

export interface ChartData {
  label: string
  value: number
  extra?: Record<string, any>
}

export async function listDashboards(): Promise<Dashboard[]>
export async function getDashboard(id: string): Promise<{ dashboard: Dashboard; widgets: Widget[] }>
export async function updateDashboard(id: string, req: { title?: string; description?: string }): Promise<Dashboard>
export async function createWidget(dashboardId: string, req: Partial<Widget>): Promise<Widget>
export async function updateWidget(dashboardId: string, widgetId: string, req: Partial<Widget>): Promise<Widget>
export async function deleteWidget(dashboardId: string, widgetId: string): Promise<void>
export async function listDataSources(): Promise<DataSourceMeta[]>
export async function queryDataSource(key: string, params?: Record<string, any>): Promise<ChartData[]>
```

- [ ] **Step 1: 创建 dashboard.ts**

- [ ] **Step 2: 运行 typecheck**

```bash
cd AppPilot/admin && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add admin/src/api/dashboard.ts
git commit -m "feat(admin): add dashboard API client for dashboards, widgets, and datasources"
```

---

### Task 6: 前端 — 安装 react-grid-layout + 添加 CSS

**Files:**
- Modify: `AppPilot/admin/package.json` (npm install)
- Modify: `AppPilot/admin/src/styles.css` (react-grid-layout 样式覆盖)

- [ ] **Step 1: 安装 react-grid-layout**

```bash
cd AppPilot/admin && npm install react-grid-layout
```

- [ ] **Step 2: 添加 CSS 样式覆盖到 styles.css**

```css
/* react-grid-layout 暗色主题覆盖 */
.react-grid-layout {
  position: relative;
}
.react-grid-item {
  transition: all 200ms ease;
  transition-property: left, top, width, height;
}
.react-grid-item.cssTransforms {
  transition-property: transform, width, height;
}
.react-grid-item > .react-resizable-handle {
  opacity: 0;
  transition: opacity 0.2s;
}
.react-grid-item:hover > .react-resizable-handle {
  opacity: 1;
}
.react-grid-item > .react-resizable-handle::after {
  border-right-color: var(--primary);
  border-bottom-color: var(--primary);
}
.react-grid-placeholder {
  background: rgba(99, 102, 241, 0.15) !important;
  border: 2px dashed var(--primary) !important;
  border-radius: var(--radius-lg);
}
```

- [ ] **Step 3: Commit**

```bash
git add admin/package.json admin/package-lock.json admin/src/styles.css
git commit -m "feat(admin): add react-grid-layout dependency and dark theme styles"
```

---

### Task 7: 前端 — WidgetCard + AddWidgetDrawer 组件

**Files:**
- Create: `AppPilot/admin/src/components/WidgetCard.tsx`
- Create: `AppPilot/admin/src/components/AddWidgetDrawer.tsx`

- [ ] **Step 1: 创建 WidgetCard.tsx**

根据 widget 类型渲染不同内容：
- `stat` — 查询 dataSource 返回单值，用大号数字展示（类似 StatCard）
- `chart` — 查询 dataSource 返回 ChartData[]，用 recharts 渲染（AreaChart/BarChart/PieChart 根据 config.chartType 选择）
- `table` — 查询 dataSource 返回 ChartData[]，用表格展示

WidgetCard 接收 `widget: Widget` prop，内部自行 `useEffect` + `queryDataSource` 加载数据。

```tsx
interface WidgetCardProps {
  widget: Widget
  isEditing?: boolean
  onDelete?: () => void
}
```

- [ ] **Step 2: 创建 AddWidgetDrawer.tsx**

右侧滑出抽屉，包含：
- 数据源选择（从 `listDataSources()` 获取列表，搜索过滤）
- Widget 类型选择（stat/chart/table）
- 标题输入
- 添加按钮

```tsx
interface AddWidgetDrawerProps {
  open: boolean
  onClose: () => void
  onAdd: (widget: { type: string; title: string; dataSource: string; gridW: number; gridH: number }) => void
}
```

- [ ] **Step 3: 运行 typecheck + test**

```bash
cd AppPilot/admin && npm run typecheck && npm test
```

- [ ] **Step 4: Commit**

```bash
git add admin/src/components/WidgetCard.tsx admin/src/components/AddWidgetDrawer.tsx
git commit -m "feat(admin): add WidgetCard and AddWidgetDrawer components"
```

---

### Task 8: 前端 — DashboardDetailPage + 路由 + 导航

**Files:**
- Create: `AppPilot/admin/src/pages/DashboardDetailPage.tsx`
- Modify: `AppPilot/admin/src/App.tsx`
- Modify: `AppPilot/admin/src/components/Layout.tsx`

- [ ] **Step 1: 创建 DashboardDetailPage.tsx**

核心结构：
```
┌──────────────────────────────────────────────┐
│ 应用选择 tabs: [FinFlow] [FluxBlog]          │
│ 标题 + 编辑布局按钮                            │
├──────────────────────────────────────────────┤
│ 12 列 react-grid-layout                       │
│ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐         │
│ │ Stat │ │ Stat │ │ Stat │ │ Stat │         │
│ └──────┘ └──────┘ └──────┘ └──────┘         │
│ ┌───────────────┐ ┌──────────┐              │
│ │   趋势图       │ │ 分类饼图  │              │
│ └───────────────┘ └──────────┘              │
├──────────────────────────────────────────────┤
│ 编辑模式：添加 Widget 按钮 + 删除按钮         │
└──────────────────────────────────────────────┘
```

关键逻辑：
- URL param `:app` → 调 `getDashboardByApp(app)` 获取 dashboard + widgets
- 非编辑模式：`react-grid-layout` 的 `isDraggable={false}` `isResizable={false}`
- 编辑模式：`isDraggable={true}` `isResizable={true}`，每个 widget 右上角显示删除按钮
- 布局变更时调 `updateWidgetLayout` 保存到后端
- "添加 Widget" 按钮 → 打开 AddWidgetDrawer
- 删除 widget 时调 `deleteWidget` + 重新加载

```tsx
// 关键状态
const [dashboard, setDashboard] = useState<Dashboard | null>(null)
const [widgets, setWidgets] = useState<Widget[]>([])
const [isEditing, setIsEditing] = useState(false)
const [drawerOpen, setDrawerOpen] = useState(false)
const [app, setApp] = useState(params.app) // from useParams
```

- [ ] **Step 2: 在 App.tsx 添加路由**

```tsx
import DashboardDetailPage from './pages/DashboardDetailPage'

// 在 Layout 内部：
<Route path="/admin/dashboards/:app" element={<DashboardDetailPage />} />
```

- [ ] **Step 3: 在 Layout.tsx 添加导航项**

在 `navItems` 数组中，概览之后添加：
```tsx
{ to: '/admin/dashboards/finflow', label: '应用看板', end: false, icon: BarChart3 },
```

顶部 import 添加 `BarChart3`：
```tsx
import { LayoutDashboard, Users, PenLine, Activity, BarChart3, LogOut, type LucideIcon } from 'lucide-react'
```

- [ ] **Step 4: 运行 typecheck + test**

```bash
cd AppPilot/admin && npm run typecheck && npm test
```

- [ ] **Step 5: Commit**

```bash
git add admin/src/pages/DashboardDetailPage.tsx admin/src/App.tsx admin/src/components/Layout.tsx
git commit -m "feat(admin): add DashboardDetailPage with drag-and-drop grid layout"
```

---

### Task 9: 端到端验证

- [ ] **Step 1: 启动后端，用 curl 测试 dashboard CRUD**

```bash
# 列出 dashboards
curl -s http://localhost:8080/api/v1/admin/dashboards -H "Authorization: Bearer $TOKEN" | jq

# 获取单个 dashboard
curl -s http://localhost:8080/api/v1/admin/dashboards/1 -H "Authorization: Bearer $TOKEN" | jq

# 列出数据源
curl -s http://localhost:8080/api/v1/admin/datasources -H "Authorization: Bearer $TOKEN" | jq

# 查询数据源
curl -s -X POST http://localhost:8080/api/v1/admin/datasources/finflow:summary/query \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{}' | jq
```

- [ ] **Step 2: 启动 admin 前端验证看板页面**

打开 `http://localhost:5076/admin/dashboards/finflow`，确认能看到默认看板（4 个 stat 卡片 + 2 个图表），点击"编辑布局"可拖拽调整。

- [ ] **Step 3: Commit（如有微调）**