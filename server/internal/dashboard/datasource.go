package dashboard

import (
	"context"
	"database/sql"
	"fmt"
)

// ChartData is a single point rendered by a dashboard widget. Label is the
// category/series name; Value is the numeric measure; Extra carries any
// auxiliary fields (e.g. a second series for trend charts).
type ChartData struct {
	Label string         `json:"label"`
	Value float64        `json:"value"`
	Extra map[string]any `json:"extra,omitempty"`
}

// DataSourceMeta is the registry listing entry for a data source: its key
// (used in widget.data_source) and a human description.
type DataSourceMeta struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

// DataSource is implemented by every built-in data source. Query receives the
// widget config params (may be nil/empty) and returns the chart series.
type DataSource interface {
	Key() string
	Description() string
	Query(ctx context.Context, params map[string]any) ([]ChartData, error)
}

// Registry holds the registered data sources keyed by their Key().
type Registry struct {
	sources map[string]DataSource
}

// NewRegistry constructs a Registry over the given DB and registers all
// built-in data sources (finflow + fluxblog).
func NewRegistry(db *sql.DB) *Registry {
	r := &Registry{sources: make(map[string]DataSource)}
	r.register(&finflowSummarySource{db: db})
	r.register(&finflowDailyTrendSource{db: db})
	r.register(&finflowCategoryBreakdownSource{db: db})
	r.register(&fluxblogSummarySource{db: db})
	r.register(&fluxblogPostTrendSource{db: db})
	r.register(&fluxblogAuthorActivitySource{db: db})
	return r
}

func (r *Registry) register(ds DataSource) {
	r.sources[ds.Key()] = ds
}

// List returns metadata for every registered data source, ordered by key.
func (r *Registry) List() []DataSourceMeta {
	out := make([]DataSourceMeta, 0, len(r.sources))
	// Collect keys first for deterministic ordering.
	keys := make([]string, 0, len(r.sources))
	for k := range r.sources {
		keys = append(keys, k)
	}
	// Simple insertion sort by key for stable output without sort import.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	for _, k := range keys {
		ds := r.sources[k]
		out = append(out, DataSourceMeta{Key: ds.Key(), Description: ds.Description()})
	}
	return out
}

// Query looks up the data source by key and delegates to its Query method.
// Returns an error if no source is registered for the key.
func (r *Registry) Query(key string, params map[string]any) ([]ChartData, error) {
	ds, ok := r.sources[key]
	if !ok {
		return nil, fmt.Errorf("dashboard: unknown data source %q", key)
	}
	return ds.Query(context.Background(), params)
}

// ---- finflow:summary ----

type finflowSummarySource struct {
	db *sql.DB
}

func (s *finflowSummarySource) Key() string         { return "finflow:summary" }
func (s *finflowSummarySource) Description() string {
	return "FinFlow 核心指标：总交易数、本月收入、本月支出、活跃用户数"
}

func (s *finflowSummarySource) Query(ctx context.Context, _ map[string]any) ([]ChartData, error) {
	// Single query: total tx count, current-month income, current-month
	// expense, and distinct active users (anyone with >=1 transaction).
	const q = `
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN type='income'  AND date >= date_trunc('month', NOW()) THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type='expense' AND date >= date_trunc('month', NOW()) THEN amount ELSE 0 END), 0),
			COUNT(DISTINCT user_id)
		FROM transactions`
	var total, activeUsers int64
	var monthIncome, monthExpense float64
	if err := s.db.QueryRowContext(ctx, q).Scan(&total, &monthIncome, &monthExpense, &activeUsers); err != nil {
		return nil, err
	}
	return []ChartData{
		{Label: "总交易数", Value: float64(total)},
		{Label: "本月收入", Value: monthIncome},
		{Label: "本月支出", Value: monthExpense},
		{Label: "活跃用户", Value: float64(activeUsers)},
	}, nil
}

// ---- finflow:daily_trend ----

type finflowDailyTrendSource struct {
	db *sql.DB
}

func (s *finflowDailyTrendSource) Key() string { return "finflow:daily_trend" }
func (s *finflowDailyTrendSource) Description() string {
	return "近 30 天每日收入/支出趋势"
}

func (s *finflowDailyTrendSource) Query(ctx context.Context, _ map[string]any) ([]ChartData, error) {
	const q = `
		SELECT to_char(date, 'YYYY-MM-DD') AS d,
			COALESCE(SUM(CASE WHEN type='income'  THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type='expense' THEN amount ELSE 0 END), 0)
		FROM transactions
		WHERE date >= (CURRENT_DATE - INTERVAL '29 days')
		GROUP BY date, d
		ORDER BY d`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChartData{}
	for rows.Next() {
		var label string
		var income, expense float64
		if err := rows.Scan(&label, &income, &expense); err != nil {
			return nil, err
		}
		out = append(out, ChartData{
			Label: label,
			Value: expense,
			Extra: map[string]any{"income": income, "expense": expense},
		})
	}
	return out, rows.Err()
}

// ---- finflow:category_breakdown ----

type finflowCategoryBreakdownSource struct {
	db *sql.DB
}

func (s *finflowCategoryBreakdownSource) Key() string { return "finflow:category_breakdown" }
func (s *finflowCategoryBreakdownSource) Description() string {
	return "本月支出分类占比（Top 10）"
}

func (s *finflowCategoryBreakdownSource) Query(ctx context.Context, _ map[string]any) ([]ChartData, error) {
	const q = `
		SELECT c.name, COALESCE(SUM(t.amount), 0) AS total
		FROM transactions t
		JOIN categories c ON c.id = t.category_id
		WHERE t.type = 'expense'
		  AND t.date >= date_trunc('month', NOW())::date
		GROUP BY c.name
		ORDER BY total DESC
		LIMIT 10`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChartData{}
	for rows.Next() {
		var name string
		var total float64
		if err := rows.Scan(&name, &total); err != nil {
			return nil, err
		}
		out = append(out, ChartData{Label: name, Value: total})
	}
	return out, rows.Err()
}

// ---- fluxblog:summary ----

type fluxblogSummarySource struct {
	db *sql.DB
}

func (s *fluxblogSummarySource) Key() string         { return "fluxblog:summary" }
func (s *fluxblogSummarySource) Description() string {
	return "FluxBlog 核心指标：文章总数、作者数、本月发布、公开文章数"
}

func (s *fluxblogSummarySource) Query(ctx context.Context, _ map[string]any) ([]ChartData, error) {
	const q = `
		SELECT
			COUNT(*),
			COUNT(DISTINCT user_id),
			COALESCE(COUNT(*) FILTER (WHERE published_at >= date_trunc('month', NOW())), 0),
			COALESCE(COUNT(*) FILTER (WHERE visibility = 'public'), 0)
		FROM blog_drafts`
	var total, authors, monthPublished, publicPosts int64
	if err := s.db.QueryRowContext(ctx, q).Scan(&total, &authors, &monthPublished, &publicPosts); err != nil {
		return nil, err
	}
	return []ChartData{
		{Label: "文章总数", Value: float64(total)},
		{Label: "作者数", Value: float64(authors)},
		{Label: "本月发布", Value: float64(monthPublished)},
		{Label: "公开文章", Value: float64(publicPosts)},
	}, nil
}

// ---- fluxblog:post_trend ----

type fluxblogPostTrendSource struct {
	db *sql.DB
}

func (s *fluxblogPostTrendSource) Key() string         { return "fluxblog:post_trend" }
func (s *fluxblogPostTrendSource) Description() string {
	return "近 12 个月发布趋势"
}

func (s *fluxblogPostTrendSource) Query(ctx context.Context, _ map[string]any) ([]ChartData, error) {
	const q = `
		SELECT to_char(published_at, 'YYYY-MM') AS ym, COUNT(*) AS cnt
		FROM blog_drafts
		WHERE status = 'published'
		  AND published_at >= date_trunc('month', NOW()) - INTERVAL '11 months'
		GROUP BY ym
		ORDER BY ym`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChartData{}
	for rows.Next() {
		var ym string
		var cnt int64
		if err := rows.Scan(&ym, &cnt); err != nil {
			return nil, err
		}
		out = append(out, ChartData{Label: ym, Value: float64(cnt)})
	}
	return out, rows.Err()
}

// ---- fluxblog:author_activity ----

type fluxblogAuthorActivitySource struct {
	db *sql.DB
}

func (s *fluxblogAuthorActivitySource) Key() string         { return "fluxblog:author_activity" }
func (s *fluxblogAuthorActivitySource) Description() string {
	return "作者活跃度排行（Top 10）"
}

func (s *fluxblogAuthorActivitySource) Query(ctx context.Context, _ map[string]any) ([]ChartData, error) {
	const q = `
		SELECT bu.username, COUNT(*) AS cnt
		FROM blog_drafts d
		JOIN blog_users bu ON bu.id = d.user_id
		WHERE d.status = 'published'
		GROUP BY bu.username
		ORDER BY cnt DESC
		LIMIT 10`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChartData{}
	for rows.Next() {
		var username string
		var cnt int64
		if err := rows.Scan(&username, &cnt); err != nil {
			return nil, err
		}
		out = append(out, ChartData{Label: username, Value: float64(cnt)})
	}
	return out, rows.Err()
}
