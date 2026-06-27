package finflow

import (
	"database/sql"
	"strconv"

	"github.com/shopspring/decimal"
)

type Summary struct {
	Income  decimal.Decimal `json:"income"`
	Expense decimal.Decimal `json:"expense"`
	Net     decimal.Decimal `json:"net"`
}

type CategoryStat struct {
	CategoryID string          `json:"categoryId"`
	Type       string          `json:"type"`
	Total      decimal.Decimal `json:"total"`
	Count      int             `json:"count"`
}

type DailyStat struct {
	Date    string          `json:"date"`
	Income  decimal.Decimal `json:"income"`
	Expense decimal.Decimal `json:"expense"`
}

func (r *Repository) Summary(userID int64, startDate, endDate string) (Summary, error) {
	q := `SELECT
		COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN type='expense' THEN amount ELSE 0 END), 0)
		FROM transactions WHERE user_id = $1`
	args := []any{userID}
	if startDate != "" && endDate != "" {
		q += " AND date BETWEEN $2 AND $3"
		args = append(args, startDate, endDate)
	}
	var s Summary
	err := r.db.QueryRow(q, args...).Scan(&s.Income, &s.Expense)
	if err != nil {
		return Summary{}, err
	}
	s.Net = s.Income.Sub(s.Expense)
	return s, nil
}

func (r *Repository) CategoryBreakdown(userID int64, txType, startDate, endDate string) ([]CategoryStat, error) {
	q := `SELECT category_id, $2, SUM(amount), COUNT(*)
		FROM transactions WHERE user_id = $1 AND type = $2`
	args := []any{userID, txType}
	if startDate != "" && endDate != "" {
		q += " AND date BETWEEN $3 AND $4"
		args = append(args, startDate, endDate)
	}
	q += " GROUP BY category_id"
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CategoryStat
	for rows.Next() {
		var catID sql.NullInt64
		var stat CategoryStat
		if err := rows.Scan(&catID, &stat.Type, &stat.Total, &stat.Count); err != nil {
			return nil, err
		}
		if catID.Valid {
			stat.CategoryID = strconv.FormatInt(catID.Int64, 10)
		}
		out = append(out, stat)
	}
	return out, rows.Err()
}

func (r *Repository) DailyTrend(userID int64, startDate, endDate string) ([]DailyStat, error) {
	q := `SELECT date,
		COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN type='expense' THEN amount ELSE 0 END), 0)
		FROM transactions WHERE user_id = $1`
	args := []any{userID}
	if startDate != "" && endDate != "" {
		q += " AND date BETWEEN $2 AND $3"
		args = append(args, startDate, endDate)
	}
	q += " GROUP BY date ORDER BY date"
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DailyStat
	for rows.Next() {
		var d DailyStat
		if err := rows.Scan(&d.Date, &d.Income, &d.Expense); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
