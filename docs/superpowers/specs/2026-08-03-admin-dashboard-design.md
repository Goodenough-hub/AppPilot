# Admin Dashboard 增强设计

**日期**: 2026-08-03
**状态**: 设计完成，待实现

## 目标

将 AppPilot admin 从单一概览页升级为多应用、可配置的 dashboard 系统，同时新增跨应用页面分析（PV/UV）。

## 架构

```
Admin SPA
├── 系统概览 /admin                    (现有增强)
├── 应用看板 /admin/dashboards/:app    (新增)
├── 页面分析 /admin/analytics          (新增)
├── 用户管理 /admin/users             (现有)
├── 用户详情 /admin/users/:id         (现有)
└── 博客账号 /admin/blog-users        (现有)
        │
/api/v1/admin/*  ───  AppPilot Server
        │
   PostgreSQL (新增 analytics_events, dashboards, dashboard_widgets)
```

## 数据库新增表

### analytics_events — 前端埋点事件

```sql
CREATE TABLE analytics_events (
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
CREATE INDEX idx_ae_app_time ON analytics_events(app, created_at);
CREATE INDEX idx_ae_path ON analytics_events(path);
CREATE INDEX idx_ae_user ON analytics_events(user_id);
```

### dashboards — 应用看板定义

```sql
CREATE TABLE dashboards (
    id          BIGSERIAL PRIMARY KEY,
    app         TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);
```

### dashboard_widgets — 看板上的卡片/图表

```sql
CREATE TABLE dashboard_widgets (
    id           BIGSERIAL PRIMARY KEY,
    dashboard_id BIGINT NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,           -- 'stat' | 'chart' | 'table'
    title        TEXT NOT NULL,
    data_source  TEXT NOT NULL,
    config       JSONB DEFAULT '{}',
    grid_x       INT DEFAULT 0,
    grid_y       INT DEFAULT 0,
    grid_w       INT DEFAULT 4,
    grid_h       INT DEFAULT 3,
    sort_order   INT DEFAULT 0,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);
```

## 后端 API

### 埋点收集

| 端点 | 用途 |
|---|---|
| `POST /api/v1/analytics/track` | 公开，前端埋点上报（fire-and-forget） |

### 分析查询（admin 鉴权）

| 端点 | 用途 |
|---|---|
| `GET /api/v1/admin/analytics/pv` | PV/UV 聚合（按 app/时间/路径） |
| `GET /api/v1/admin/analytics/realtime` | 实时在线（近5分钟） |

### Dashboard CRUD（admin 鉴权）

| 端点 | 用途 |
|---|---|
| `GET /api/v1/admin/dashboards` | 列出所有 dashboard |
| `GET /api/v1/admin/dashboards/:id` | 单个 dashboard 含 widgets |
| `PUT /api/v1/admin/dashboards/:id` | 更新 dashboard 布局 |
| `POST /api/v1/admin/dashboards/:id/widgets` | 添加 widget |
| `PUT /api/v1/admin/dashboards/:id/widgets/:wid` | 更新 widget |
| `DELETE /api/v1/admin/dashboards/:id/widgets/:wid` | 删除 widget |

### 数据源（admin 鉴权）

| 端点 | 用途 |
|---|---|
| `GET /api/v1/admin/datasources` | 列出可用数据源（app + key + description） |
| `POST /api/v1/admin/datasources/:key/query` | 查询某数据源（key 中用 `:` 替换为 `/`） |

## DataSource 抽象（Go 接口）

```go
type DataSource interface {
    Key() string
    Description() string
    Query(ctx context.Context, params map[string]any) ([]ChartData, error)
}
```

### 计划数据源

| Key | 描述 |
|---|---|
| `finflow:summary` | 交易/用户核心指标 |
| `finflow:daily_trend` | 交易日报趋势（收入/支出） |
| `finflow:category_breakdown` | 分类占比 |
| `finflow:account_balance` | 账户余额分布 |
| `fluxblog:summary` | 文章/作者核心指标 |
| `fluxblog:post_trend` | 发布趋势 |
| `fluxblog:author_activity` | 作者活跃度 |
| `analytics:pv` | 页面访问量 |
| `analytics:top_pages` | 热门页面排行 |

## 前端设计

### 路由变更

在 `App.tsx` 中新增：
```
/admin/dashboards/:app  → DashboardDetailPage
/admin/analytics        → AnalyticsPage
```

### 侧边导航（Layout.tsx）

```ts
{ to: '/admin', label: '概览', end: true, icon: LayoutDashboard }
{ to: '/admin/dashboards/finflow', label: '应用看板', end: false, icon: BarChart3 }
{ to: '/admin/analytics', label: '页面分析', end: false, icon: Activity }
{ to: '/admin/users', label: '用户管理', end: false, icon: Users }
{ to: '/admin/blog-users', label: '博客账号', end: false, icon: PenLine }
```

### 应用看板页 (`DashboardDetailPage`)

- 顶部：应用选择 tabs（FinFlow / FluxBlog）
- 12 列响应式网格（`react-grid-layout`）
- 右上角"编辑布局"按钮：切换编辑模式
- 编辑模式：widget 可拖拽排序、调整大小、删除
- "添加 Widget"抽屉：选数据源 → 选类型（stat/chart/table）→ 配标题 → 添加
- 非编辑模式：纯展示，数据从 `/dashboard/:id` + `/datasources/:key/query` 加载

### 页面分析页 (`AnalyticsPage`)

- 顶部：应用筛选 + 时间范围（今天/7天/30天/自定义）
- 概览行：PV / UV / 人均页面数
- 趋势图：PV/UV 日趋势（折线图，recharts）
- 热门页面排行表：路径 + PV + UV + 占比

### 前端埋点 SDK

在 `AppPilot/admin/src/api/` 下新增 `analytics.ts`：

```ts
// 各应用前端在路由切换时调用
analytics.track({
  app: 'finflow',
  event: 'pageview',
  path: '/transactions',
  title: '账单',
})
```

对于各应用前端（FinFlow PWA、FluxBlog 等）：在路由切换时调用 `POST /api/v1/analytics/track`，fire-and-forget，上报失败不影响主流程。

### 新增依赖

- `react-grid-layout` — 拖拽网格布局

## 分阶段实施

| 阶段 | 内容 |
|---|---|
| Phase 1 | analytics_events 表 + 埋点端点 + 前端埋点 SDK + 页面分析页 |
| Phase 2 | dashboards/widgets 表 + CRUD API + DataSource 抽象 + 应用看板页（含拖拽） |
| Phase 3 | 各应用 data source 实现 + 更多图表类型 |

## 不做的

- nginx log 解析：由外部日志管道处理，不在此 spec 范围内
- 拖拽编排的撤销/重做：v1 不做
- widget 配置面板（图表颜色、时间范围等）：先用 `config` JSONB 存默认值，后续迭代