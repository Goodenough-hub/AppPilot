package dashboard

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Repository wraps a *sql.DB for dashboard persistence.
type Repository struct {
	db *sql.DB
}

// NewRepository constructs a dashboard Repository over the given connection.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Dashboard mirrors a row in the dashboards table. WidgetCount is populated
// via a LEFT JOIN COUNT over dashboard_widgets (see ListDashboards).
type Dashboard struct {
	ID          int64     `json:"id,string"`
	App         string    `json:"app"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	WidgetCount int       `json:"widgetCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Widget mirrors a row in the dashboard_widgets table. Config is a JSONB
// column scanned into json.RawMessage so callers can unmarshal lazily.
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

const dashboardCols = `d.id, d.app, d.title, d.description, d.created_at, d.updated_at`

const widgetCols = `id, dashboard_id, type, title, data_source, config, grid_x, grid_y, grid_w, grid_h, sort_order, created_at, updated_at`

// ---- Dashboards ----

// ListDashboards returns every dashboard ordered by app, each with its widget
// count computed via a LEFT JOIN COUNT over dashboard_widgets.
func (r *Repository) ListDashboards() ([]Dashboard, error) {
	rows, err := r.db.Query(
		`SELECT ` + dashboardCols + `, COUNT(w.id) AS widget_count
		 FROM dashboards d
		 LEFT JOIN dashboard_widgets w ON w.dashboard_id = d.id
		 GROUP BY d.id
		 ORDER BY d.app`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Dashboard{}
	for rows.Next() {
		var d Dashboard
		if err := rows.Scan(&d.ID, &d.App, &d.Title, &d.Description, &d.CreatedAt, &d.UpdatedAt, &d.WidgetCount); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDashboard returns a single dashboard by id (no widgets).
func (r *Repository) GetDashboard(id int64) (*Dashboard, error) {
	var d Dashboard
	err := r.db.QueryRow(
		`SELECT `+dashboardCols+`
		 FROM dashboards d
		 WHERE d.id = $1`,
		id,
	).Scan(&d.ID, &d.App, &d.Title, &d.Description, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// GetDashboardByApp returns the dashboard configured for the given app slug.
func (r *Repository) GetDashboardByApp(app string) (*Dashboard, error) {
	var d Dashboard
	err := r.db.QueryRow(
		`SELECT `+dashboardCols+`
		 FROM dashboards d
		 WHERE d.app = $1`,
		app,
	).Scan(&d.ID, &d.App, &d.Title, &d.Description, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// UpdateDashboard patches title and/or description on the dashboard with the
// given id. nil pointers mean "leave unchanged". Returns the updated row.
func (r *Repository) UpdateDashboard(id int64, title, description *string) (*Dashboard, error) {
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	n := 1
	if title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", n))
		args = append(args, *title)
		n++
	}
	if description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", n))
		args = append(args, *description)
		n++
	}
	args = append(args, id) // $n
	q := `UPDATE dashboards SET ` + strings.Join(sets, ", ") +
		fmt.Sprintf(` WHERE id = $%d`, n) + ` RETURNING ` + dashboardCols
	var d Dashboard
	err := r.db.QueryRow(q, args...).Scan(
		&d.ID, &d.App, &d.Title, &d.Description, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ---- Widgets ----

// ListWidgets returns all widgets for a dashboard ordered by sort_order.
func (r *Repository) ListWidgets(dashboardID int64) ([]Widget, error) {
	rows, err := r.db.Query(
		`SELECT `+widgetCols+`
		 FROM dashboard_widgets
		 WHERE dashboard_id = $1
		 ORDER BY sort_order, id`,
		dashboardID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Widget{}
	for rows.Next() {
		w, err := scanWidget(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

// CreateWidget inserts a widget under the given dashboard and returns the row
// as the database materialized it (including generated id/timestamps).
func (r *Repository) CreateWidget(dashboardID int64, w Widget) (*Widget, error) {
	row := r.db.QueryRow(
		`INSERT INTO dashboard_widgets (dashboard_id, type, title, data_source, config, grid_x, grid_y, grid_w, grid_h, sort_order)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING `+widgetCols,
		dashboardID, w.Type, w.Title, w.DataSource, nullableJSON(w.Config),
		w.GridX, w.GridY, w.GridW, w.GridH, w.SortOrder,
	)
	return scanWidget(row.Scan)
}

// UpdateWidget overwrites every mutable field of the widget with id.
func (r *Repository) UpdateWidget(id int64, w Widget) (*Widget, error) {
	row := r.db.QueryRow(
		`UPDATE dashboard_widgets SET
		   type = $2, title = $3, data_source = $4, config = $5,
		   grid_x = $6, grid_y = $7, grid_w = $8, grid_h = $9,
		   sort_order = $10, updated_at = NOW()
		 WHERE id = $1
		 RETURNING `+widgetCols,
		id, w.Type, w.Title, w.DataSource, nullableJSON(w.Config),
		w.GridX, w.GridY, w.GridW, w.GridH, w.SortOrder,
	)
	return scanWidget(row.Scan)
}

// DeleteWidget removes a widget by id. Returns sql.ErrNoRows if not found.
func (r *Repository) DeleteWidget(id int64) error {
	res, err := r.db.Exec(`DELETE FROM dashboard_widgets WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdateWidgetLayout patches only the grid placement columns of a widget,
// leaving type/title/config untouched.
func (r *Repository) UpdateWidgetLayout(id int64, gridX, gridY, gridW, gridH int) error {
	res, err := r.db.Exec(
		`UPDATE dashboard_widgets
		 SET grid_x = $2, grid_y = $3, grid_w = $4, grid_h = $5, updated_at = NOW()
		 WHERE id = $1`,
		id, gridX, gridY, gridW, gridH,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---- scan helpers ----

type scanFn func(dest ...any) error

func scanWidget(scan scanFn) (*Widget, error) {
	var w Widget
	var config sql.NullString
	if err := scan(
		&w.ID, &w.DashboardID, &w.Type, &w.Title, &w.DataSource, &config,
		&w.GridX, &w.GridY, &w.GridW, &w.GridH, &w.SortOrder,
		&w.CreatedAt, &w.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if config.Valid {
		w.Config = json.RawMessage(config.String)
	} else {
		w.Config = json.RawMessage(`{}`)
	}
	return &w, nil
}

// nullableJSON normalizes a json.RawMessage so an empty value is stored as '{}'.
func nullableJSON(b json.RawMessage) any {
	if len(b) == 0 || string(b) == "null" {
		return []byte(`{}`)
	}
	return []byte(b)
}
