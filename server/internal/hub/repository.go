package hub

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

var ErrNotFound = errors.New("item not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// UpdatePatch：所有字段可选（nil 表示不改）。
type UpdatePatch struct {
	Type     *string
	Title    *string
	URL      *string
	Content  *string
	Tags     *[]string
	Favorite *bool
}

// List 返回 user_id 作用域下的全部条目，
// 排序：favorite 优先，updated_at 降序。
func (r *Repository) List(userID int64) ([]Item, error) {
	rows, err := r.db.Query(`
SELECT id, user_id, type, title, url, content, tags, favorite, created_at, updated_at
FROM hub_items
WHERE user_id = $1
ORDER BY favorite DESC, updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		var it Item
		var tags pq.StringArray
		if err := rows.Scan(&it.ID, &it.UserID, &it.Type, &it.Title, &it.URL, &it.Content, &tags, &it.Favorite, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		it.Tags = []string(tags)
		out = append(out, it)
	}
	return out, rows.Err()
}

// Create 插入一条并回填 ID/时间戳。
func (r *Repository) Create(userID int64, it *Item) (*Item, error) {
	if it.Tags == nil {
		it.Tags = []string{}
	}
	row := r.db.QueryRow(`
INSERT INTO hub_items (user_id, type, title, url, content, tags, favorite)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, created_at, updated_at`,
		userID, it.Type, it.Title, it.URL, it.Content, pq.StringArray(it.Tags), it.Favorite)
	if err := row.Scan(&it.ID, &it.CreatedAt, &it.UpdatedAt); err != nil {
		return nil, err
	}
	it.UserID = userID
	return it, nil
}

// Update 局部更新。至少一个字段被更新才走 UPDATE；否则直接返回原记录。
func (r *Repository) Update(userID, id int64, p UpdatePatch) (*Item, error) {
	sets := []string{}
	args := []any{}
	next := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, col+" = $"+itoa(len(args)))
	}
	if p.Type != nil {
		next("type", *p.Type)
	}
	if p.Title != nil {
		next("title", *p.Title)
	}
	if p.URL != nil {
		next("url", *p.URL)
	}
	if p.Content != nil {
		next("content", *p.Content)
	}
	if p.Tags != nil {
		next("tags", pq.StringArray(*p.Tags))
	}
	if p.Favorite != nil {
		next("favorite", *p.Favorite)
	}
	if len(sets) == 0 {
		return r.findByID(userID, id)
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, userID, id)
	q := "UPDATE hub_items SET " + joinComma(sets) +
		" WHERE user_id = $" + itoa(len(args)-1) + " AND id = $" + itoa(len(args))
	res, err := r.db.Exec(q, args...)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.findByID(userID, id)
}

// Delete 按 (user_id, id) 硬删。未匹配返回 ErrNotFound。
func (r *Repository) Delete(userID, id int64) error {
	res, err := r.db.Exec(`DELETE FROM hub_items WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) findByID(userID, id int64) (*Item, error) {
	row := r.db.QueryRow(`
SELECT id, user_id, type, title, url, content, tags, favorite, created_at, updated_at
FROM hub_items WHERE user_id = $1 AND id = $2`, userID, id)
	var it Item
	var tags pq.StringArray
	if err := row.Scan(&it.ID, &it.UserID, &it.Type, &it.Title, &it.URL, &it.Content, &tags, &it.Favorite, &it.CreatedAt, &it.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	it.Tags = []string(tags)
	return &it, nil
}

// 内部小工具，避免引 strconv 与 strings 两个包。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func joinComma(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += ", " + p
	}
	return out
}
