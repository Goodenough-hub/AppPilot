package finflow

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ---- Transactions ----

type ListTxFilter struct {
	StartDate  *string
	EndDate    *string
	Type       *string
	CategoryID *int64
	AccountID  *int64
	Keyword    *string
	TripID     *int64
	Limit      int
	Offset     int
}

func (r *Repository) ListTransactions(userID int64, f ListTxFilter) ([]Transaction, error) {
	q := `SELECT id, amount, type, note, date, time, created_at, category_id, account_id, to_account_id, source_id, source_type, vendor, trip_id
	      FROM transactions WHERE user_id = $1`
	args := []any{userID}
	n := 2
	if f.StartDate != nil {
		q += fmt.Sprintf(" AND date >= $%d", n)
		args = append(args, *f.StartDate)
		n++
	}
	if f.EndDate != nil {
		q += fmt.Sprintf(" AND date <= $%d", n)
		args = append(args, *f.EndDate)
		n++
	}
	if f.Type != nil {
		q += fmt.Sprintf(" AND type = $%d", n)
		args = append(args, *f.Type)
		n++
	}
	if f.CategoryID != nil {
		q += fmt.Sprintf(" AND category_id = $%d", n)
		args = append(args, *f.CategoryID)
		n++
	}
	if f.AccountID != nil {
		q += fmt.Sprintf(" AND (account_id = $%d OR to_account_id = $%d)", n, n)
		args = append(args, *f.AccountID)
		n++
	}
	if f.TripID != nil {
		q += fmt.Sprintf(" AND trip_id = $%d", n)
		args = append(args, *f.TripID)
		n++
	}
	if f.Keyword != nil {
		q += fmt.Sprintf(" AND note ILIKE $%d", n)
		args = append(args, "%"+*f.Keyword+"%")
		n++
	}
	q += " ORDER BY date DESC, time DESC NULLS LAST, created_at DESC"
	if f.Limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", n)
		args = append(args, f.Limit)
		n++
	}
	if f.Offset > 0 {
		q += fmt.Sprintf(" OFFSET $%d", n)
		args = append(args, f.Offset)
	}
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTransactions(rows)
}

func (r *Repository) GetTransaction(userID, id int64) (*Transaction, error) {
	row := r.db.QueryRow(
		`SELECT id, amount, type, note, date, time, created_at, category_id, account_id, to_account_id, source_id, source_type, vendor, trip_id
		 FROM transactions WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	return scanTransaction(row.Scan)
}

func (r *Repository) CreateTransaction(userID int64, t Transaction) (*Transaction, error) {
	var dateStr string
	if t.Date != "" {
		dateStr = t.Date
	} else {
		return nil, fmt.Errorf("date is required")
	}
	catID, accID, toAcc, tripID := idPtrToNullInt(t.CategoryID), idPtrToNullInt(t.AccountID), idPtrToNullInt(t.ToAccount), idPtrToNullInt(t.TripID)
	row := r.db.QueryRow(
		`INSERT INTO transactions (user_id, amount, type, note, date, time, category_id, account_id, to_account_id, source_id, source_type, vendor, trip_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		 RETURNING id, amount, type, note, date, time, created_at, category_id, account_id, to_account_id, source_id, source_type, vendor, trip_id`,
		userID, t.Amount, t.Type, t.Note, dateStr, strPtrToNullTime(t.Time),
		catID, accID, toAcc, strPtrToNullString(t.SourceID), strPtrToNullString(t.SourceType), strPtrToNullString(t.Vendor), tripID,
	)
	return scanTransaction(row.Scan)
}

func (r *Repository) UpdateTransaction(userID, id int64, t Transaction) (*Transaction, error) {
	catID, accID, toAcc, tripID := idPtrToNullInt(t.CategoryID), idPtrToNullInt(t.AccountID), idPtrToNullInt(t.ToAccount), idPtrToNullInt(t.TripID)
	row := r.db.QueryRow(
		`UPDATE transactions SET amount=$3, type=$4, note=$5, date=$6, time=$7, category_id=$8, account_id=$9, to_account_id=$10, source_id=$11, source_type=$12, vendor=$13, trip_id=$14
		 WHERE id=$1 AND user_id=$2
		 RETURNING id, amount, type, note, date, time, created_at, category_id, account_id, to_account_id, source_id, source_type, vendor, trip_id`,
		id, userID, t.Amount, t.Type, t.Note, t.Date, strPtrToNullTime(t.Time),
		catID, accID, toAcc, strPtrToNullString(t.SourceID), strPtrToNullString(t.SourceType), strPtrToNullString(t.Vendor), tripID,
	)
	return scanTransaction(row.Scan)
}

func (r *Repository) DeleteTransaction(userID, id int64) error {
	res, err := r.db.Exec(`DELETE FROM transactions WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---- Categories ----

func (r *Repository) ListCategories(userID int64) ([]Category, error) {
	rows, err := r.db.Query(
		`SELECT id, name, type, icon, color_hex, sort_order, is_system, parent_id, scope, created_at
		 FROM categories WHERE user_id = $1 ORDER BY type, sort_order, id`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		var parentID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.Icon, &c.ColorHex, &c.SortOrder, &c.IsSystem, &parentID, &c.Scope, &c.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			s := strconv.FormatInt(parentID.Int64, 10)
			c.ParentID = &s
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCategory(userID int64, c Category) (*Category, error) {
	if c.Scope == "" {
		c.Scope = "normal"
	}
	row := r.db.QueryRow(
		`INSERT INTO categories (user_id, name, type, icon, color_hex, sort_order, is_system, parent_id, scope)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id, name, type, icon, color_hex, sort_order, is_system, parent_id, scope, created_at`,
		userID, c.Name, c.Type, c.Icon, c.ColorHex, c.SortOrder, c.IsSystem, idPtrToNullInt(c.ParentID), c.Scope,
	)
	var out Category
	var parentID sql.NullInt64
	err := row.Scan(&out.ID, &out.Name, &out.Type, &out.Icon, &out.ColorHex, &out.SortOrder, &out.IsSystem, &parentID, &out.Scope, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		s := strconv.FormatInt(parentID.Int64, 10)
		out.ParentID = &s
	}
	return &out, nil
}

func (r *Repository) UpdateCategory(userID, id int64, c Category) (*Category, error) {
	row := r.db.QueryRow(
		`UPDATE categories SET name=$3, icon=$4, color_hex=$5, sort_order=$6, parent_id=$7
		 WHERE id=$1 AND user_id=$2
		 RETURNING id, name, type, icon, color_hex, sort_order, is_system, parent_id, scope, created_at`,
		id, userID, c.Name, c.Icon, c.ColorHex, c.SortOrder, idPtrToNullInt(c.ParentID),
	)
	var out Category
	var parentID sql.NullInt64
	err := row.Scan(&out.ID, &out.Name, &out.Type, &out.Icon, &out.ColorHex, &out.SortOrder, &out.IsSystem, &parentID, &out.Scope, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		s := strconv.FormatInt(parentID.Int64, 10)
		out.ParentID = &s
	}
	return &out, nil
}

func (r *Repository) DeleteCategory(userID, id int64) error {
	res, err := r.db.Exec(`DELETE FROM categories WHERE id = $1 AND user_id = $2 AND is_system = FALSE`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---- Accounts ----

func (r *Repository) ListAccounts(userID int64) ([]Account, error) {
	rows, err := r.db.Query(
		`SELECT id, name, type, icon, color_hex, initial_balance, sort_order, is_system, parent_id, created_at
		 FROM accounts WHERE user_id = $1 ORDER BY sort_order, id`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Account{}
	for rows.Next() {
		var a Account
		var parentID sql.NullInt64
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Icon, &a.ColorHex, &a.InitialBalance, &a.SortOrder, &a.IsSystem, &parentID, &a.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			s := strconv.FormatInt(parentID.Int64, 10)
			a.ParentID = &s
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) CreateAccount(userID int64, a Account) (*Account, error) {
	row := r.db.QueryRow(
		`INSERT INTO accounts (user_id, name, type, icon, color_hex, initial_balance, sort_order, is_system, parent_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id, name, type, icon, color_hex, initial_balance, sort_order, is_system, parent_id, created_at`,
		userID, a.Name, a.Type, a.Icon, a.ColorHex, a.InitialBalance, a.SortOrder, a.IsSystem, idPtrToNullInt(a.ParentID),
	)
	var out Account
	var parentID sql.NullInt64
	if err := row.Scan(&out.ID, &out.Name, &out.Type, &out.Icon, &out.ColorHex, &out.InitialBalance, &out.SortOrder, &out.IsSystem, &parentID, &out.CreatedAt); err != nil {
		return nil, err
	}
	if parentID.Valid {
		s := strconv.FormatInt(parentID.Int64, 10)
		out.ParentID = &s
	}
	return &out, nil
}

func (r *Repository) UpdateAccount(userID, id int64, a Account) (*Account, error) {
	row := r.db.QueryRow(
		`UPDATE accounts SET name=$3, type=$4, icon=$5, color_hex=$6, initial_balance=$7, sort_order=$8, parent_id=$9
		 WHERE id=$1 AND user_id=$2
		 RETURNING id, name, type, icon, color_hex, initial_balance, sort_order, is_system, parent_id, created_at`,
		id, userID, a.Name, a.Type, a.Icon, a.ColorHex, a.InitialBalance, a.SortOrder, idPtrToNullInt(a.ParentID),
	)
	var out Account
	var parentID sql.NullInt64
	if err := row.Scan(&out.ID, &out.Name, &out.Type, &out.Icon, &out.ColorHex, &out.InitialBalance, &out.SortOrder, &out.IsSystem, &parentID, &out.CreatedAt); err != nil {
		return nil, err
	}
	if parentID.Valid {
		s := strconv.FormatInt(parentID.Int64, 10)
		out.ParentID = &s
	}
	return &out, nil
}

func (r *Repository) DeleteAccount(userID, id int64) error {
	var hasChildren bool
	err := r.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM accounts WHERE parent_id = $1 AND user_id = $2)`,
		id, userID,
	).Scan(&hasChildren)
	if err != nil {
		return err
	}
	if hasChildren {
		return fmt.Errorf("account has child accounts, delete children first")
	}
	res, err := r.db.Exec(`DELETE FROM accounts WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) ClearAccounts(userID int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM transactions WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM recurring_transactions WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM accounts WHERE user_id = $1`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// ---- Budgets ----

func (r *Repository) ListBudgets(userID int64, year, month int) ([]Budget, error) {
	rows, err := r.db.Query(
		`SELECT id, amount, month, year, category_id FROM budgets WHERE user_id = $1 AND year = $2 AND month = $3`,
		userID, year, month,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Budget{}
	for rows.Next() {
		var b Budget
		var catID int64
		if err := rows.Scan(&b.ID, &b.Amount, &b.Month, &b.Year, &catID); err != nil {
			return nil, err
		}
		b.CategoryID = strconv.FormatInt(catID, 10)
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertBudget(userID int64, b Budget) (*Budget, error) {
	catID, _ := strconv.ParseInt(b.CategoryID, 10, 64)
	row := r.db.QueryRow(
		`INSERT INTO budgets (user_id, amount, month, year, category_id)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (user_id, year, month, category_id) DO UPDATE SET amount = EXCLUDED.amount
		 RETURNING id, amount, month, year, category_id`,
		userID, b.Amount, b.Month, b.Year, catID,
	)
	var out Budget
	var cID int64
	if err := row.Scan(&out.ID, &out.Amount, &out.Month, &out.Year, &cID); err != nil {
		return nil, err
	}
	out.CategoryID = strconv.FormatInt(cID, 10)
	return &out, nil
}

func (r *Repository) DeleteBudget(userID, id int64) error {
	res, err := r.db.Exec(`DELETE FROM budgets WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---- Recurring ----

func (r *Repository) ListRecurring(userID int64) ([]Recurring, error) {
	rows, err := r.db.Query(
		`SELECT id, amount, type, note, category_id, account_id, to_account_id, frequency, interval, day_of_month, day_of_week, next_date, start_date, end_date, is_active, created_at
		 FROM recurring_transactions WHERE user_id = $1 ORDER BY next_date`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Recurring{}
	for rows.Next() {
		r, err := scanRecurring(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func (r *Repository) CreateRecurring(userID int64, rec Recurring) (*Recurring, error) {
	row := r.db.QueryRow(
		`INSERT INTO recurring_transactions (user_id, amount, type, note, category_id, account_id, to_account_id, frequency, interval, day_of_month, day_of_week, next_date, start_date, end_date, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		 RETURNING id, amount, type, note, category_id, account_id, to_account_id, frequency, interval, day_of_month, day_of_week, next_date, start_date, end_date, is_active, created_at`,
		userID, rec.Amount, rec.Type, rec.Note, idPtrToNullInt(rec.CategoryID), idPtrToNullInt(rec.AccountID), idPtrToNullInt(rec.ToAccount),
		rec.Frequency, rec.Interval, intPtrToNullInt(rec.DayOfMonth), intPtrToNullInt(rec.DayOfWeek),
		rec.NextDate, rec.StartDate, strPtrToNullString(rec.EndDate), rec.IsActive,
	)
	return scanRecurring(row.Scan)
}

func (r *Repository) UpdateRecurring(userID, id int64, rec Recurring) (*Recurring, error) {
	row := r.db.QueryRow(
		`UPDATE recurring_transactions SET amount=$3, type=$4, note=$5, category_id=$6, account_id=$7, to_account_id=$8, frequency=$9, interval=$10, day_of_month=$11, day_of_week=$12, next_date=$13, start_date=$14, end_date=$15, is_active=$16
		 WHERE id=$1 AND user_id=$2
		 RETURNING id, amount, type, note, category_id, account_id, to_account_id, frequency, interval, day_of_month, day_of_week, next_date, start_date, end_date, is_active, created_at`,
		id, userID, rec.Amount, rec.Type, rec.Note, idPtrToNullInt(rec.CategoryID), idPtrToNullInt(rec.AccountID), idPtrToNullInt(rec.ToAccount),
		rec.Frequency, rec.Interval, intPtrToNullInt(rec.DayOfMonth), intPtrToNullInt(rec.DayOfWeek),
		rec.NextDate, rec.StartDate, strPtrToNullString(rec.EndDate), rec.IsActive,
	)
	return scanRecurring(row.Scan)
}

func (r *Repository) DeleteRecurring(userID, id int64) error {
	res, err := r.db.Exec(`DELETE FROM recurring_transactions WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) ListDueRecurring(userID int64, today string) ([]Recurring, error) {
	rows, err := r.db.Query(
		`SELECT id, amount, type, note, category_id, account_id, to_account_id, frequency, interval, day_of_month, day_of_week, next_date, start_date, end_date, is_active, created_at
		 FROM recurring_transactions WHERE user_id = $1 AND is_active = TRUE AND next_date <= $2`,
		userID, today,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Recurring{}
	for rows.Next() {
		r, err := scanRecurring(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func (r *Repository) AdvanceRecurringNextDate(id int64, nextDate string) error {
	_, err := r.db.Exec(`UPDATE recurring_transactions SET next_date = $2 WHERE id = $1`, id, nextDate)
	return err
}

// ---- scan helpers ----

type scanFn func(dest ...any) error

func scanTransaction(scan scanFn) (*Transaction, error) {
	var t Transaction
	var dateVal, timeVal sql.NullTime
	var catID, accID, toAcc, tripID sql.NullInt64
	var sourceID, sourceType, vendor sql.NullString
	if err := scan(&t.ID, &t.Amount, &t.Type, &t.Note, &dateVal, &timeVal, &t.CreatedAt, &catID, &accID, &toAcc, &sourceID, &sourceType, &vendor, &tripID); err != nil {
		return nil, err
	}
	if dateVal.Valid {
		t.Date = dateVal.Time.Format("2006-01-02")
	}
	if timeVal.Valid {
		s := timeVal.Time.Format("15:04:05")
		t.Time = &s
	}
	if catID.Valid {
		s := strconv.FormatInt(catID.Int64, 10)
		t.CategoryID = &s
	}
	if accID.Valid {
		s := strconv.FormatInt(accID.Int64, 10)
		t.AccountID = &s
	}
	if toAcc.Valid {
		s := strconv.FormatInt(toAcc.Int64, 10)
		t.ToAccount = &s
	}
	if sourceID.Valid {
		s := sourceID.String
		t.SourceID = &s
	}
	if sourceType.Valid {
		s := sourceType.String
		t.SourceType = &s
	}
	if vendor.Valid {
		s := vendor.String
		t.Vendor = &s
	}
	if tripID.Valid {
		s := strconv.FormatInt(tripID.Int64, 10)
		t.TripID = &s
	}
	return &t, nil
}

func scanTransactions(rows *sql.Rows) ([]Transaction, error) {
	out := []Transaction{}
	for rows.Next() {
		t, err := scanTransaction(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func scanRecurring(scan scanFn) (*Recurring, error) {
	var r Recurring
	var catID, accID, toAcc sql.NullInt64
	var dayOfMonth, dayOfWeek sql.NullInt64
	var nextDate, startDate sql.NullTime
	var endDate sql.NullTime
	if err := scan(&r.ID, &r.Amount, &r.Type, &r.Note, &catID, &accID, &toAcc, &r.Frequency, &r.Interval, &dayOfMonth, &dayOfWeek, &nextDate, &startDate, &endDate, &r.IsActive, &r.CreatedAt); err != nil {
		return nil, err
	}
	if nextDate.Valid {
		r.NextDate = nextDate.Time.Format("2006-01-02")
	}
	if startDate.Valid {
		r.StartDate = startDate.Time.Format("2006-01-02")
	}
	if catID.Valid {
		s := strconv.FormatInt(catID.Int64, 10)
		r.CategoryID = &s
	}
	if accID.Valid {
		s := strconv.FormatInt(accID.Int64, 10)
		r.AccountID = &s
	}
	if toAcc.Valid {
		s := strconv.FormatInt(toAcc.Int64, 10)
		r.ToAccount = &s
	}
	if dayOfMonth.Valid {
		v := int(dayOfMonth.Int64)
		r.DayOfMonth = &v
	}
	if dayOfWeek.Valid {
		v := int(dayOfWeek.Int64)
		r.DayOfWeek = &v
	}
	if endDate.Valid {
		s := endDate.Time.Format("2006-01-02")
		r.EndDate = &s
	}
	return &r, nil
}

// ---- ptr helpers ----

func idPtrToNullInt(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	id, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		return nil
	}
	return id
}

func strPtrToNullString(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

func strPtrToNullTime(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

func intPtrToNullInt(i *int) any {
	if i == nil {
		return nil
	}
	return *i
}

// 防止 unused 警告（decimal 包将来用于金额计算）
var _ = decimal.Decimal{}
var _ = strings.TrimSpace

// ---- Trips ----

func (r *Repository) ListTrips(userID int64) ([]Trip, error) {
	rows, err := r.db.Query(
		`SELECT id, name, start_date, end_date, budget, note, created_at
		 FROM trips WHERE user_id = $1 ORDER BY created_at DESC, id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Trip{}
	for rows.Next() {
		t, err := scanTrip(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func scanTrip(scan scanFn) (*Trip, error) {
	var t Trip
	var start, end sql.NullTime
	if err := scan(&t.ID, &t.Name, &start, &end, &t.Budget, &t.Note, &t.CreatedAt); err != nil {
		return nil, err
	}
	if start.Valid {
		s := start.Time.Format("2006-01-02")
		t.StartDate = &s
	}
	if end.Valid {
		s := end.Time.Format("2006-01-02")
		t.EndDate = &s
	}
	return &t, nil
}

func (r *Repository) CreateTrip(userID int64, t Trip) (*Trip, error) {
	row := r.db.QueryRow(
		`INSERT INTO trips (user_id, name, start_date, end_date, budget, note)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 RETURNING id, name, start_date, end_date, budget, note, created_at`,
		userID, t.Name, strPtrToNullTime(t.StartDate), strPtrToNullTime(t.EndDate), t.Budget, t.Note,
	)
	return scanTrip(row.Scan)
}

func (r *Repository) UpdateTrip(userID, id int64, t Trip) (*Trip, error) {
	row := r.db.QueryRow(
		`UPDATE trips SET name=$3, start_date=$4, end_date=$5, budget=$6, note=$7
		 WHERE id=$1 AND user_id=$2
		 RETURNING id, name, start_date, end_date, budget, note, created_at`,
		id, userID, t.Name, strPtrToNullTime(t.StartDate), strPtrToNullTime(t.EndDate), t.Budget, t.Note,
	)
	return scanTrip(row.Scan)
}

func (r *Repository) DeleteTrip(userID, id int64) error {
	res, err := r.db.Exec(`DELETE FROM trips WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
