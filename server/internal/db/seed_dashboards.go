package db

import "database/sql"

// defaultWidgets 定义每个已知 app 的默认看板 widget 布局。
// 键为 app 名（与 dashboards.app UNIQUE 列一致），值为有序 widget 列表。
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

// SeedDashboards 为每个已知 app 创建默认 dashboard + widgets。
// 幂等：dashboard 用 ON CONFLICT(app) DO UPDATE upsert；
// 仅当 dashboard 无任何 widget 时才插入默认 widgets，避免覆盖用户自定义布局。
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
