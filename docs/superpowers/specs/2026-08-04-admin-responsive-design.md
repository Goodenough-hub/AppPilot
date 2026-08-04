# AppPilot Admin 设备适配设计

- 日期：2026-08-04
- 范围：`AppPilot/admin`（管理后台 SPA，Vite + React 19 + TS）
- 目标：完整响应式打磨——手机 / 平板 / 桌面各设断点，桌面端保持不变，重点解决 iPhone 上「显示很难看」的问题。

## 背景与问题

当前 admin 是**固定桌面布局**，几乎没有移动端媒体查询（`styles.css` 中仅有 `hover`/`pointer`/`reduced-motion`）：

- `.admin-shell` 是 flex 横向布局，**侧边栏固定 260px 且始终可见**。在 ~390px 的 iPhone 上只剩约 100px 给正文，基本不可用。
- `.stat-grid` 硬编码 `repeat(4, 1fr)`（手机上仍 4 列）；`.charts-row` 固定 `2fr 1fr`。
- 表格：通用 `.table-container` 有 `overflow-x:auto`，但 FluxBlog 的 `.admin-table` 无滚动容器、单元格内边距大。
- 仪表盘 `DashboardDetailPage` 用 `react-grid-layout` 锁死 12 列（`BREAKPOINTS={lg:0}`），手机上 widget 极小、拖拽编辑不可用。

## 方案

采用**混合方案（CSS 优先 + 极少 JS）**：布局、间距、网格、导航抽屉全部走 CSS 断点；只有真正需要按宽度切换 DOM 的地方（仪表盘单列只读）引入一个共享 `useMediaQuery` hook。理由：契合现有「单一 `styles.css` 设计系统」的风格，桌面端完全不受影响，移动端逻辑最少、可复用。

### 断点

| 名称 | 条件 | 说明 |
|---|---|---|
| 手机 | `max-width: 640px` | 单列、卡片化、抽屉导航 |
| 平板 | `max-width: 1024px` | 网格降列、抽屉导航 |
| 桌面 | `> 1024px` | 现状不变 |

所有改动都包裹在移动/平板断点内，**桌面端（>1024px）视觉与行为零变化**。

## 分节设计

### 1. 全局约定

- 新增 `src/hooks/useMediaQuery.ts`（基于 `window.matchMedia`），**仅**供仪表盘判断是否单列只读。
- 移动端表单输入字号 ≥16px（防 iOS 聚焦自动放大）；可点元素最小高度 44px。
- 检查 `index.html` 是否含 `<meta name="viewport" content="width=device-width, initial-scale=1">`，缺则补。

### 2. 应用外壳与导航（`components/Layout.tsx` + `styles.css`）

- `>1024`：保持现有 260px 固定侧边栏，逻辑不变。
- `≤1024`：侧边栏改为**左侧滑出抽屉**（`position:fixed` + `transform: translateX`）。新增：
  - 顶栏 `.admin-topbar`：左侧品牌 `AppPilot` + Logo，右侧汉堡按钮。
  - 抽屉背景遮罩 `.admin-drawer-backdrop`（点击关闭）。
  - `Layout` 增加 `drawerOpen` state；点导航项 / 点遮罩 / 路由变化 → 自动关闭；打开时锁 `body` 滚动。
  - 动画复用现有 `--ease-drawer` / `--dur-sheet` token，已被全局 `prefers-reduced-motion` 覆盖。

### 3. 网格与间距（`styles.css`）

- `.stat-grid`：`repeat(4, 1fr)` → `repeat(auto-fit, minmax(200px, 1fr))`，自动降为 2 列 / 1 列。
- `.charts-row`：`2fr 1fr` → `≤1024` 堆成单列。
- `.admin-shell` / `.admin-main` / `.page-container` 内边距在 `≤640` 收窄（如 24→12、32/40→16）。
- `.admin-page-header`：`≤640` 由横向 space-between 改为纵向堆叠；标题字号 28→22；`.stat-card-value` 34→28。

### 4. 表格卡片化（`styles.css` + 各表格 JSX）

- 新增 `.responsive-table` 类；`≤640` 时：`thead` 隐藏，每 `tr` 变卡片，`td` 左右分布，`td::before { content: attr(data-label) }` 显示列名，操作列独占一行。
- 为用到的表格加 `className="responsive-table"` 并给每个 `<td>` 补 `data-label="列名"`。涉及页面：`UsersPage`、`BlogUsersPage`、`UserDetailPage`、`AnalyticsPage`（若含表格）。

### 5. 仪表盘单列只读（`pages/DashboardDetailPage.tsx`）

- 用 `useMediaQuery('(max-width: 640px)')` 得 `isMobile`。
- `isMobile` 时**不渲染** `react-grid-layout`，改为单列依次堆叠 `WidgetCard`；隐藏「编辑 / 添加 Widget」按钮，显示提示「拖拽编辑请在大屏设备操作」。桌面端逻辑不变。

## 测试与验收

- 单测（vitest）：`useMediaQuery` hook；`Layout` 抽屉开关（点汉堡展开、点遮罩/导航项关闭）。卡片化为纯 CSS，不做单测。
- 提交前必过：`npm run typecheck`、`npm test`、`npm run build`。
- 手动回归：
  - 确认 `>1024` 桌面端视觉零变化。
  - iPhone 尺寸（~390px）下各页面可读、无横向溢出、导航（抽屉）可用、表格以卡片呈现、仪表盘单列只读。

## 非目标（YAGNI）

- 不重做设计系统 / 配色。
- 不为平板做区别于「手机+抽屉」的专属布局（平板复用移动端断点即可）。
- 不做手机端的仪表盘拖拽编辑。
