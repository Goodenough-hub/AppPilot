package hub

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

var ErrNotFound = errors.New("item not found")

// ErrFolderExists 表示同 (user_id, type) 下已存在同名文件夹。
var ErrFolderExists = errors.New("folder already exists")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// execer 抽象 *sql.DB 与 *sql.Tx 的 Exec，供 upsertFolder 复用。
type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// upsertFolder 把 (userID, typ, name) 登记进 hub_folders（已存在则忽略）。
// 条目写入/导入时调用，保证"仅被条目引用"的文件夹名也在目录里落地。name 为空串时跳过。
func upsertFolder(ex execer, userID int64, typ, name string) error {
	if name == "" {
		return nil
	}
	_, err := ex.Exec(`
INSERT INTO hub_folders (user_id, type, name) VALUES ($1, $2, $3)
ON CONFLICT ON CONSTRAINT hub_folders_unique DO NOTHING`, userID, typ, name)
	return err
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

// UpdatePatch：所有字段可选（nil 表示不改）。
type UpdatePatch struct {
	Type     *string
	Title    *string
	URL      *string
	Content  *string
	Tags     *[]string
	Favorite *bool
	Folder   *string
}

// List 返回 user_id 作用域下的全部条目，
// 排序：favorite 优先，updated_at 降序。
func (r *Repository) List(userID int64) ([]Item, error) {
	rows, err := r.db.Query(`
SELECT id, user_id, type, title, url, content, tags, favorite, folder, created_at, updated_at
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
		if err := rows.Scan(&it.ID, &it.UserID, &it.Type, &it.Title, &it.URL, &it.Content, &tags, &it.Favorite, &it.Folder, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		it.Tags = []string(tags)
		out = append(out, it)
	}
	return out, rows.Err()
}

// Create 插入一条并回填 ID/时间戳。folder 非空时同步登记进 hub_folders。
func (r *Repository) Create(userID int64, it *Item) (*Item, error) {
	tags := it.Tags
	if tags == nil {
		tags = []string{}
	}
	row := r.db.QueryRow(`
INSERT INTO hub_items (user_id, type, title, url, content, tags, favorite, folder)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, created_at, updated_at`,
		userID, it.Type, it.Title, it.URL, it.Content, pq.StringArray(tags), it.Favorite, it.Folder)
	if err := row.Scan(&it.ID, &it.CreatedAt, &it.UpdatedAt); err != nil {
		return nil, err
	}
	if err := upsertFolder(r.db, userID, it.Type, it.Folder); err != nil {
		return nil, err
	}
	it.UserID = userID
	it.Tags = tags // 回填标准化后的空 slice 以保证返回值一致
	return it, nil
}

// Update 局部更新。至少一个字段被更新才走 UPDATE；否则直接返回原记录。
func (r *Repository) Update(userID, id int64, p UpdatePatch) (*Item, error) {
	sets := []string{}
	args := []any{}
	next := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, col+" = $"+strconv.Itoa(len(args)))
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
	if p.Folder != nil {
		next("folder", *p.Folder)
	}
	if len(sets) == 0 {
		return r.findByID(userID, id)
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, userID, id)
	q := "UPDATE hub_items SET " + strings.Join(sets, ", ") +
		" WHERE user_id = $" + strconv.Itoa(len(args)-1) + " AND id = $" + strconv.Itoa(len(args))
	res, err := r.db.Exec(q, args...)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	updated, err := r.findByID(userID, id)
	if err != nil {
		return nil, err
	}
	// folder 有变化时按更新后的 (type, folder) 登记目录
	if p.Folder != nil {
		if err := upsertFolder(r.db, userID, updated.Type, updated.Folder); err != nil {
			return nil, err
		}
	}
	return updated, nil
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
SELECT id, user_id, type, title, url, content, tags, favorite, folder, created_at, updated_at
FROM hub_items WHERE user_id = $1 AND id = $2`, userID, id)
	var it Item
	var tags pq.StringArray
	if err := row.Scan(&it.ID, &it.UserID, &it.Type, &it.Title, &it.URL, &it.Content, &tags, &it.Favorite, &it.Folder, &it.CreatedAt, &it.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	it.Tags = []string(tags)
	return &it, nil
}

// ExportAll 返回该 user 的全量条目（顺序等同 List）。
func (r *Repository) ExportAll(userID int64) ([]Item, error) {
	return r.List(userID)
}

// ImportBatch 按 mode 导入：
// - "merge"：按 ID 冲突则 UPDATE，无则 INSERT。id=0 视为新条目。
// - "replace"：事务内先 DELETE user scope 全部，再全量 INSERT（忽略传入 id）。
// 条目携带的非空 folder 会在同一事务内登记进 hub_folders；
// replace 只清条目、不清文件夹目录（不再被引用的空文件夹可由用户手动删除）。
// 返回受影响行数（merge 时是 update+insert 之和；replace 时是 insert 数）。
func (r *Repository) ImportBatch(userID int64, items []Item, mode string) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// folderUpserts 收集本事务内出现过的 (type, folder)，提交前统一登记
	type folderKey struct{ typ, name string }
	folderUpserts := map[folderKey]struct{}{}
	rememberFolder := func(typ, name string) {
		if name != "" {
			folderUpserts[folderKey{typ, name}] = struct{}{}
		}
	}
	commitFolders := func() error {
		for k := range folderUpserts {
			if err := upsertFolder(tx, userID, k.typ, k.name); err != nil {
				return err
			}
		}
		return nil
	}

	if mode == "replace" {
		if len(items) == 0 {
			return 0, errors.New("replace mode requires non-empty items")
		}
		if _, err := tx.Exec(`DELETE FROM hub_items WHERE user_id = $1`, userID); err != nil {
			return 0, err
		}
		for i := range items {
			it := &items[i]
			tags := it.Tags
			if tags == nil {
				tags = []string{}
			}
			if _, err := tx.Exec(`
INSERT INTO hub_items (user_id, type, title, url, content, tags, favorite, folder)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
				userID, it.Type, it.Title, it.URL, it.Content, pq.StringArray(tags), it.Favorite, it.Folder); err != nil {
				return 0, err
			}
			rememberFolder(it.Type, it.Folder)
		}
		if err := commitFolders(); err != nil {
			return 0, err
		}
		return len(items), tx.Commit()
	}

	// merge
	count := 0
	for i := range items {
		it := &items[i]
		tags := it.Tags
		if tags == nil {
			tags = []string{}
		}
		if it.ID != 0 {
			res, err := tx.Exec(`
UPDATE hub_items SET type=$1, title=$2, url=$3, content=$4, tags=$5, favorite=$6, folder=$7, updated_at=NOW()
WHERE user_id=$8 AND id=$9`,
				it.Type, it.Title, it.URL, it.Content, pq.StringArray(tags), it.Favorite, it.Folder, userID, it.ID)
			if err != nil {
				return 0, err
			}
			if n, _ := res.RowsAffected(); n > 0 {
				count++
				rememberFolder(it.Type, it.Folder)
				continue
			}
			// id 未命中（可能来自别的用户或已删除）→ 走 insert（新 id）
		}
		if _, err := tx.Exec(`
INSERT INTO hub_items (user_id, type, title, url, content, tags, favorite, folder)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			userID, it.Type, it.Title, it.URL, it.Content, pq.StringArray(tags), it.Favorite, it.Folder); err != nil {
			return 0, err
		}
		count++
		rememberFolder(it.Type, it.Folder)
	}
	if err := commitFolders(); err != nil {
		return 0, err
	}
	return count, tx.Commit()
}

// ListFolders 返回该 user 某 type 下的全部文件夹（含条数），按创建时间升序。
func (r *Repository) ListFolders(userID int64, typ string) ([]Folder, error) {
	rows, err := r.db.Query(`
SELECT f.id, f.user_id, f.type, f.name, f.created_at,
       (SELECT count(*) FROM hub_items i
         WHERE i.user_id = f.user_id AND i.type = f.type AND i.folder = f.name) AS item_count
FROM hub_folders f
WHERE f.user_id = $1 AND f.type = $2
ORDER BY f.created_at ASC, f.id ASC`, userID, typ)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Folder{}
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.UserID, &f.Type, &f.Name, &f.CreatedAt, &f.ItemCount); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// CreateFolder 新建文件夹并回填 ID/时间戳。重名返回 ErrFolderExists。
func (r *Repository) CreateFolder(userID int64, f *Folder) (*Folder, error) {
	row := r.db.QueryRow(`
INSERT INTO hub_folders (user_id, type, name) VALUES ($1, $2, $3)
RETURNING id, created_at`, userID, f.Type, f.Name)
	if err := row.Scan(&f.ID, &f.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrFolderExists
		}
		return nil, err
	}
	f.UserID = userID
	return f, nil
}

// RenameFolder 重命名文件夹，并级联更新同 user+type 下所有条目的 folder
// （级联只改 folder 列、不动条目的 updated_at，避免列表顺序被打乱）。
// 文件夹不存在返回 ErrNotFound；目标名已存在返回 ErrFolderExists。
func (r *Repository) RenameFolder(userID, id int64, newName string) (*Folder, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var typ, oldName string
	var createdAt time.Time
	err = tx.QueryRow(`
SELECT type, name, created_at FROM hub_folders
WHERE user_id = $1 AND id = $2 FOR UPDATE`, userID, id).Scan(&typ, &oldName, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE hub_folders SET name = $1 WHERE user_id = $2 AND id = $3`, newName, userID, id); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrFolderExists
		}
		return nil, err
	}
	if _, err := tx.Exec(`
UPDATE hub_items SET folder = $1
WHERE user_id = $2 AND type = $3 AND folder = $4`, newName, userID, typ, oldName); err != nil {
		return nil, err
	}
	f := &Folder{ID: id, UserID: userID, Type: typ, Name: newName, CreatedAt: createdAt}
	return f, tx.Commit()
}

// DeleteFolder 删除文件夹；其下条目 folder 置空回落未分类（条目不删）。
// 文件夹不存在返回 ErrNotFound。
func (r *Repository) DeleteFolder(userID, id int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var typ, name string
	err = tx.QueryRow(`
SELECT type, name FROM hub_folders
WHERE user_id = $1 AND id = $2 FOR UPDATE`, userID, id).Scan(&typ, &name)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}
	if _, err := tx.Exec(`DELETE FROM hub_folders WHERE user_id = $1 AND id = $2`, userID, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`
UPDATE hub_items SET folder = ''
WHERE user_id = $1 AND type = $2 AND folder = $3`, userID, typ, name); err != nil {
		return err
	}
	return tx.Commit()
}
