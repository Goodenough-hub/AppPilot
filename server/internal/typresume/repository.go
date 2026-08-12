package typresume

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("resume not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// SELECT / RETURNING 列顺序（scanOne 依赖同顺序）
const resumeCols = `client_id, name, mode, content, active_file, files, created_at, updated_at`

// ListByUser 返回该用户的所有简历，按更新时间倒序。
func (r *Repository) ListByUser(userID int64) ([]Resume, error) {
	rows, err := r.db.Query(
		`SELECT `+resumeCols+` FROM resumes WHERE user_id = $1 ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Resume{}
	for rows.Next() {
		got, err := scanOne(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *got)
	}
	return out, rows.Err()
}

// UpsertByClientID 按 (user_id, client_id) upsert 一份简历。
// 客户端 created_at/updated_at 会被后端覆盖为 NOW()。
func (r *Repository) UpsertByClientID(userID int64, in Resume) (*Resume, error) {
	if in.ClientID == "" {
		return nil, errors.New("client_id required")
	}
	normalize(&in)
	filesJSON, err := json.Marshal(in.Files)
	if err != nil {
		return nil, fmt.Errorf("encode files: %w", err)
	}
	contentJSON := normalizeJSON(in.Content)
	row := r.db.QueryRow(
		`INSERT INTO resumes (user_id, client_id, name, mode, content, active_file, files, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		 ON CONFLICT (user_id, client_id) DO UPDATE
		   SET name = EXCLUDED.name,
		       mode = EXCLUDED.mode,
		       content = EXCLUDED.content,
		       active_file = EXCLUDED.active_file,
		       files = EXCLUDED.files,
		       updated_at = NOW()
		 RETURNING `+resumeCols,
		userID, in.ClientID, in.Name, in.Mode, contentJSON, in.ActiveFile, filesJSON,
	)
	return scanOne(row.Scan)
}

// DeleteByClientID 删除一份简历。找不到返回 ErrNotFound。
func (r *Repository) DeleteByClientID(userID int64, clientID string) error {
	res, err := r.db.Exec(
		`DELETE FROM resumes WHERE user_id = $1 AND client_id = $2`,
		userID, clientID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// BulkUpsert 首次登录迁移专用：一次性 upsert 多份简历。单事务。
func (r *Repository) BulkUpsert(userID int64, ins []Resume) ([]Resume, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO resumes (user_id, client_id, name, mode, content, active_file, files, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		 ON CONFLICT (user_id, client_id) DO UPDATE
		   SET name = EXCLUDED.name,
		       mode = EXCLUDED.mode,
		       content = EXCLUDED.content,
		       active_file = EXCLUDED.active_file,
		       files = EXCLUDED.files,
		       updated_at = NOW()
		 RETURNING ` + resumeCols,
	)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	out := make([]Resume, 0, len(ins))
	for _, in := range ins {
		if in.ClientID == "" {
			continue
		}
		normalize(&in)
		filesJSON, err := json.Marshal(in.Files)
		if err != nil {
			return nil, fmt.Errorf("encode files for %s: %w", in.ClientID, err)
		}
		contentJSON := normalizeJSON(in.Content)
		row := stmt.QueryRow(userID, in.ClientID, in.Name, in.Mode, contentJSON, in.ActiveFile, filesJSON)
		got, err := scanOne(row.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *got)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

// normalize 填缺失字段的默认值。
func normalize(in *Resume) {
	if in.Name == "" {
		in.Name = "未命名简历"
	}
	if in.Mode == "" {
		in.Mode = "typst"
	}
	if in.ActiveFile == "" {
		in.ActiveFile = "main.typ"
	}
	if in.Files == nil {
		in.Files = map[string]string{}
	}
}

// normalizeJSON 保证 JSONB 列非 null。
func normalizeJSON(b json.RawMessage) []byte {
	if len(b) == 0 || string(b) == "null" {
		return []byte(`{}`)
	}
	return []byte(b)
}

// scanOne 从单行读出 Resume。列顺序必须与 resumeCols 一致。
type scanFn func(dest ...any) error

func scanOne(scan scanFn) (*Resume, error) {
	var (
		r           Resume
		filesRaw    []byte
		contentRaw  []byte
	)
	if err := scan(&r.ClientID, &r.Name, &r.Mode, &contentRaw, &r.ActiveFile, &filesRaw, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(filesRaw, &r.Files); err != nil {
		return nil, fmt.Errorf("decode files: %w", err)
	}
	r.Content = json.RawMessage(contentRaw)
	return &r, nil
}
