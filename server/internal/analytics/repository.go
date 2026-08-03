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
	since := time.Now().Add(-window)
	err := r.db.QueryRow(
		`SELECT COUNT(DISTINCT COALESCE(session_id, ip))
		   FROM analytics_events
		  WHERE app = $1 AND event_type = 'pageview'
		    AND created_at >= $2`,
		app, since,
	).Scan(&count)
	return count, err
}
