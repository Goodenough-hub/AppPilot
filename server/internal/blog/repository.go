package blog

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ==================== BlogUser ====================

func (r *Repository) FindByUsernameActive(username string) (*BlogUser, error) {
	u := &BlogUser{}
	err := r.db.QueryRow(
		`SELECT id, username, password_hash, is_enabled, token_version, deleted_at, created_at, updated_at
		 FROM blog_users WHERE username = $1 AND deleted_at IS NULL`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsEnabled, &u.TokenVersion, &u.DeletedAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

// FindActiveByID 供中间件使用：账号必须存在、未删除。is_enabled 与 token_version
// 由调用方（中间件）进一步校验。
func (r *Repository) FindActiveByID(id int64) (*BlogUser, error) {
	u := &BlogUser{}
	err := r.db.QueryRow(
		`SELECT id, username, password_hash, is_enabled, token_version, deleted_at, created_at, updated_at
		 FROM blog_users WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsEnabled, &u.TokenVersion, &u.DeletedAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

// FindByID 用于 admin 查看（含已停用但未删除的账号）。
func (r *Repository) FindByID(id int64) (*BlogUser, error) {
	u := &BlogUser{}
	err := r.db.QueryRow(
		`SELECT id, username, password_hash, is_enabled, token_version, deleted_at, created_at, updated_at
		 FROM blog_users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsEnabled, &u.TokenVersion, &u.DeletedAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *Repository) ListUsers() ([]BlogUser, error) {
	rows, err := r.db.Query(
		`SELECT id, username, is_enabled, token_version, deleted_at, created_at, updated_at
		 FROM blog_users ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlogUser{}
	for rows.Next() {
		var u BlogUser
		if err := rows.Scan(&u.ID, &u.Username, &u.IsEnabled, &u.TokenVersion, &u.DeletedAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repository) Create(username, password string) (*BlogUser, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash: %w", err)
	}
	u := &BlogUser{}
	err = r.db.QueryRow(
		`INSERT INTO blog_users (username, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, username, is_enabled, token_version, deleted_at, created_at, updated_at`,
		username, string(hash),
	).Scan(&u.ID, &u.Username, &u.IsEnabled, &u.TokenVersion, &u.DeletedAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return u, nil
}

// UpdateProfile 改用户名/启停。停用时递增 token_version 使现有令牌立即失效。
func (r *Repository) UpdateProfile(id int64, username *string, isEnabled *bool) (*BlogUser, error) {
	sets := []string{}
	args := []any{}
	n := 1
	if username != nil {
		sets = append(sets, fmt.Sprintf("username = $%d", n))
		args = append(args, *username)
		n++
	}
	if isEnabled != nil {
		sets = append(sets, fmt.Sprintf("is_enabled = $%d", n))
		args = append(args, *isEnabled)
		n++
		// 停用时令牌立即失效
		if !*isEnabled {
			sets = append(sets, "token_version = token_version + 1")
		}
	}
	if len(sets) == 0 {
		return r.FindByID(id)
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)
	u := &BlogUser{}
	err := r.db.QueryRow(
		fmt.Sprintf(
			`UPDATE blog_users SET %s WHERE id = $%d AND deleted_at IS NULL
			 RETURNING id, username, is_enabled, token_version, deleted_at, created_at, updated_at`,
			strings.Join(sets, ", "), n,
		),
		args...,
	).Scan(&u.ID, &u.Username, &u.IsEnabled, &u.TokenVersion, &u.DeletedAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return u, nil
}

// UpdatePassword 重置密码并递增 token_version（使旧令牌失效，强制重新登录）。
func (r *Repository) UpdatePassword(id int64, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}
	res, err := r.db.Exec(
		`UPDATE blog_users SET password_hash = $1, token_version = token_version + 1, updated_at = NOW()
		 WHERE id = $2 AND deleted_at IS NULL`,
		string(hash), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

// SoftDelete 软删除账号：递增 token_version 使现有令牌立即失效，草稿/版本/审计保留。
func (r *Repository) SoftDelete(id int64) error {
	res, err := r.db.Exec(
		`UPDATE blog_users SET deleted_at = NOW(), token_version = token_version + 1, is_enabled = FALSE, updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) VerifyPassword(u *BlogUser, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
}

// ==================== Draft ====================

const draftCols = "d.id, d.user_id, d.slug, d.title, d.description, d.tags, d.cover, d.markdown, d.status, d.visibility, d.version, d.published_commit_sha, d.published_version, d.published_at, d.scheduled_publish_at, d.created_at, d.updated_at, d.project_id, p.name AS project_name"

// draftRetCols 是 blog_drafts 表本身的列（无前缀/无 JOIN），用于 INSERT/UPDATE RETURNING
// 后在应用层补 project_id 再去查 project_name。
const draftRetCols = "id, user_id, slug, title, description, tags, cover, markdown, status, visibility, version, published_commit_sha, published_version, published_at, scheduled_publish_at, created_at, updated_at, project_id"

func scanDraft(sc func(...any) error) (*Draft, error) {
	d := &Draft{}
	err := sc(&d.ID, &d.UserID, &d.Slug, &d.Title, &d.Description, pq.Array(&d.Tags),
		&d.Cover, &d.Markdown, &d.Status, &d.Visibility, &d.Version, &d.PublishedCommitSha, &d.PublishedVersion, &d.PublishedAt, &d.ScheduledPublishAt, &d.CreatedAt, &d.UpdatedAt, &d.ProjectID, &d.ProjectName)
	if err != nil {
		return nil, err
	}
	// 派生：已发布但本地版本超过已发布版本 → 有未发布修改。
	if d.Status == StatusPublished && (d.PublishedVersion == nil || d.Version > *d.PublishedVersion) {
		d.HasUnpublishedChanges = true
	}
	return d, nil
}

// scanDraftRet 扫描仅 blog_drafts 表自身列（无 JOIN），再补 projectName。
func (r *Repository) scanDraftRet(sc func(...any) error) (*Draft, error) {
	d := &Draft{}
	err := sc(&d.ID, &d.UserID, &d.Slug, &d.Title, &d.Description, pq.Array(&d.Tags),
		&d.Cover, &d.Markdown, &d.Status, &d.Visibility, &d.Version, &d.PublishedCommitSha, &d.PublishedVersion, &d.PublishedAt, &d.ScheduledPublishAt, &d.CreatedAt, &d.UpdatedAt, &d.ProjectID)
	if err != nil {
		return nil, err
	}
	if d.Status == StatusPublished && (d.PublishedVersion == nil || d.Version > *d.PublishedVersion) {
		d.HasUnpublishedChanges = true
	}
	// 补 projectName
	if d.ProjectID != nil {
		_ = r.db.QueryRow(`SELECT name FROM blog_projects WHERE id = $1`, *d.ProjectID).Scan(&d.ProjectName)
	}
	return d, nil
}

func (r *Repository) ListDrafts(userID int64) ([]Draft, error) {
	rows, err := r.db.Query(
		`SELECT `+draftCols+` FROM blog_drafts d LEFT JOIN blog_projects p ON p.id = d.project_id WHERE d.user_id = $1 ORDER BY d.updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Draft{}
	for rows.Next() {
		d, err := scanDraft(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *Repository) GetDraft(userID, id int64) (*Draft, error) {
	d, err := scanDraft(func(dst ...any) error {
		return r.db.QueryRow(
			`SELECT `+draftCols+` FROM blog_drafts d LEFT JOIN blog_projects p ON p.id = d.project_id WHERE d.id = $1 AND d.user_id = $2`,
			id, userID,
		).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}
	return d, nil
}

// GetDraftByID 不带 user 过滤，供发布/撤回 job 处理使用（job 已通过接口鉴权）。
func (r *Repository) GetDraftByID(id int64) (*Draft, error) {
	d, err := scanDraft(func(dst ...any) error {
		return r.db.QueryRow(
			`SELECT `+draftCols+` FROM blog_drafts d LEFT JOIN blog_projects p ON p.id = d.project_id WHERE d.id = $1`,
			id,
		).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}
	return d, nil
}

func (r *Repository) CreateDraft(userID int64, d Draft) (*Draft, error) {
	if !ValidSlug(d.Slug) {
		return nil, fmt.Errorf("invalid slug")
	}
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}
	vis := d.Visibility
	if vis == "" {
		vis = VisibilityPrivate
	}
	out, err := r.scanDraftRet(func(dst ...any) error {
		return r.db.QueryRow(
			`INSERT INTO blog_drafts (user_id, slug, title, description, tags, cover, markdown, status, visibility, project_id)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,'draft',$8,$9)
			 RETURNING `+draftRetCols,
			userID, d.Slug, d.Title, d.Description, pq.Array(tags), d.Cover, d.Markdown, vis, d.ProjectID,
		).Scan(dst...)
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrConflict // slug 唯一冲突也用 409
		}
		return nil, err
	}
	// 新建草稿保存 v1 检查点。
	_ = r.insertCheckpoint(out)
	return out, nil
}

// UpdateDraft 乐观锁：必须提交 baseVersion；WHERE version = baseVersion。
// 成功后 version+1。版本快照按检查点策略（≥5min）异步创建，不再每次保存都写。
func (r *Repository) UpdateDraft(userID, id, baseVersion int64, req UpdateDraftRequest) (*Draft, int64, error) {
	sets := []string{"version = version + 1", "updated_at = NOW()"}
	args := []any{}
	n := 1
	add := func(col, expr string, val any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, n))
		args = append(args, val)
		n++
	}
	if req.Slug != nil {
		if !ValidSlug(*req.Slug) {
			return nil, 0, fmt.Errorf("invalid slug")
		}
		add("slug", "", *req.Slug)
	}
	if req.Title != nil {
		add("title", "", *req.Title)
	}
	if req.Description != nil {
		add("description", "", *req.Description)
	}
	if req.Tags != nil {
		tags := req.Tags
		if tags == nil {
			tags = []string{}
		}
		add("tags", "", pq.Array(tags))
	}
	if req.Cover != nil {
		add("cover", "", *req.Cover)
	}
	if req.Markdown != nil {
		add("markdown", "", *req.Markdown)
	}
	if req.Visibility != nil && (*req.Visibility == VisibilityPublic || *req.Visibility == VisibilityPrivate) {
		add("visibility", "", *req.Visibility)
	}
	if req.ProjectID != nil {
		add("project_id", "", *req.ProjectID)
	}
	args = append(args, id, userID, baseVersion) // $n, $n+1, $n+2

	d, err := r.scanDraftRet(func(dst ...any) error {
		return r.db.QueryRow(
			fmt.Sprintf(
				`UPDATE blog_drafts SET %s WHERE id = $%d AND user_id = $%d AND version = $%d
				 RETURNING `+draftRetCols,
				strings.Join(sets, ", "), n, n+1, n+2,
			),
			args...,
		).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			var serverVersion int64
			_ = r.db.QueryRow(`SELECT version FROM blog_drafts WHERE id = $1 AND user_id = $2`, id, userID).Scan(&serverVersion)
			if serverVersion == 0 {
				return nil, 0, ErrDraftNotFound
			}
			return nil, serverVersion, ErrConflict
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			var serverVersion int64
			_ = r.db.QueryRow(`SELECT version FROM blog_drafts WHERE id = $1 AND user_id = $2`, id, userID).Scan(&serverVersion)
			return nil, serverVersion, ErrConflict
		}
		return nil, 0, err
	}
	// 检查点策略：距上一个快照≥5min 才自动创建，避免每次保存都写快照。
	_ = r.maybeCheckpoint(d)
	return d, d.Version, nil
}

// checkpointInterval 自动检查点的最小间隔。
const checkpointInterval = 5 * time.Minute

// maybeCheckpoint 距上一个快照≥checkpointInterval 或无快照时创建一个。
func (r *Repository) maybeCheckpoint(d *Draft) error {
	var last sql.NullTime
	if err := r.db.QueryRow(
		`SELECT MAX(created_at) FROM blog_draft_versions WHERE draft_id = $1`, d.ID,
	).Scan(&last); err != nil {
		return err
	}
	if last.Valid && time.Since(last.Time) < checkpointInterval {
		return nil
	}
	return r.insertCheckpoint(d)
}

// CreateCheckpoint 为草稿当前内容显式创建检查点（手动保存版本、发布前、恢复后）。
func (r *Repository) CreateCheckpoint(d *Draft) error {
	return r.insertCheckpoint(d)
}

// insertCheckpoint 插入当前草稿状态的快照，(draft_id,version) 冲突则跳过，并裁剪到 10 条。
func (r *Repository) insertCheckpoint(d *Draft) error {
	if _, err := r.db.Exec(
		`INSERT INTO blog_draft_versions (draft_id, version, title, description, tags, cover, markdown)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (draft_id, version) DO NOTHING`,
		d.ID, d.Version, d.Title, d.Description, pq.Array(d.Tags), d.Cover, d.Markdown,
	); err != nil {
		return err
	}
	return trimVersionsTx(r.db, d.ID)
}

// trimVersions 仅保留单篇最近 10 个版本快照。
func trimVersions(tx *sql.Tx, draftID int64) error {
	_, err := tx.Exec(
		`DELETE FROM blog_draft_versions WHERE draft_id = $1 AND id NOT IN (
			SELECT id FROM blog_draft_versions WHERE draft_id = $1 ORDER BY version DESC LIMIT 10)`,
		draftID,
	)
	return err
}

func (r *Repository) DeleteDraft(userID, id int64) error {
	res, err := r.db.Exec(`DELETE FROM blog_drafts WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrDraftNotFound
	}
	return nil
}

// SetDraftStatus 更新发布状态（不触版本号）。
func (r *Repository) SetDraftStatus(id int64, status string) error {
	_, err := r.db.Exec(`UPDATE blog_drafts SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	return err
}

// PublishDraft 同步发布或定时发布。
//   - req.ScheduledPublishAt != nil：定时发布，只写 scheduled_publish_at，status 保持 draft
//   - req.ScheduledPublishAt == nil：立即发布，status=published、published_version=version、
//     published_at=COALESCE(已有, NOW())、scheduled_publish_at=NULL
//
// 可选更新 visibility/project_id/tags（nil 字段保持原值）。返回最新草稿。
func (r *Repository) PublishDraft(id int64, req PublishRequest) (*Draft, error) {
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	n := 1
	add := func(col string, val any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, n))
		args = append(args, val)
		n++
	}
	if req.Visibility != nil && (*req.Visibility == VisibilityPublic || *req.Visibility == VisibilityPrivate) {
		add("visibility", *req.Visibility)
	}
	if req.ProjectID != nil {
		add("project_id", *req.ProjectID)
	}
	if req.Tags != nil {
		tags := req.Tags
		if tags == nil {
			tags = []string{}
		}
		add("tags", pq.Array(tags))
	}
	if req.ScheduledPublishAt != nil {
		add("scheduled_publish_at", *req.ScheduledPublishAt)
	} else {
		sets = append(sets, "status = '"+StatusPublished+"'",
			"published_version = version",
			"published_at = COALESCE(published_at, NOW())",
			"scheduled_publish_at = NULL")
	}
	args = append(args, id)

	d, err := r.scanDraftRet(func(dst ...any) error {
		return r.db.QueryRow(
			fmt.Sprintf(
				`UPDATE blog_drafts SET %s WHERE id = $%d RETURNING `+draftRetCols,
				strings.Join(sets, ", "), n,
			),
			args...,
		).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}
	return d, nil
}

// PublishScheduledDrafts 提升所有到点的定时草稿为已发布。
// 由后台 scheduler 周期调用。返回受影响的草稿 ID 列表。
func (r *Repository) PublishScheduledDrafts() ([]int64, error) {
	rows, err := r.db.Query(
		`SELECT id FROM blog_drafts
		 WHERE status = $1 AND scheduled_publish_at IS NOT NULL AND scheduled_publish_at <= NOW()
		 ORDER BY scheduled_publish_at ASC
		 FOR UPDATE`,
		StatusDraft,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	published := make([]int64, 0, len(ids))
	for _, id := range ids {
		_, err := r.scanDraftRet(func(dst ...any) error {
			return r.db.QueryRow(
				`UPDATE blog_drafts
				 SET status = $1, published_version = version,
				     published_at = COALESCE(published_at, NOW()),
				     scheduled_publish_at = NULL, updated_at = NOW()
				 WHERE id = $2 AND status = $3
				 RETURNING `+draftRetCols,
				StatusPublished, id, StatusDraft,
			).Scan(dst...)
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return published, err
		}
		published = append(published, id)
	}
	return published, nil
}

// ListTags 列出当前用户所有草稿中去重后的标签，按字母序排序。
func (r *Repository) ListTags(userID int64) ([]string, error) {
	rows, err := r.db.Query(
		`SELECT DISTINCT unnest(tags) AS tag FROM blog_drafts
		 WHERE user_id = $1 ORDER BY tag`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	if tags == nil {
		tags = []string{}
	}
	return tags, rows.Err()
}

// UnpublishDraft 撤回：status=draft（保留 visibility、published_at、published_version）。
func (r *Repository) UnpublishDraft(id int64) (*Draft, error) {
	d, err := r.scanDraftRet(func(dst ...any) error {
		return r.db.QueryRow(
			`UPDATE blog_drafts SET status = $1, updated_at = NOW() WHERE id = $2 RETURNING `+draftRetCols,
			StatusDraft, id,
		).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}
	return d, nil
}

// ==================== 公开 / 私有读 ====================

// summaryCols 是列表场景的精简列：不含 markdown 正文。（含 project 信息）
const summaryCols = "d.id, d.slug, d.title, d.description, d.tags, d.cover, d.status, d.visibility, d.version, d.published_at, d.scheduled_publish_at, d.updated_at, d.project_id, p.name AS project_name"

func scanDraftSummary(sc func(...any) error) (*DraftSummary, error) {
	s := &DraftSummary{}
	err := sc(&s.ID, &s.Slug, &s.Title, &s.Description, pq.Array(&s.Tags),
		&s.Cover, &s.Status, &s.Visibility, &s.Version, &s.PublishedAt, &s.ScheduledPublishAt, &s.UpdatedAt, &s.ProjectID, &s.ProjectName)
	if err != nil {
		return nil, err
	}
	if s.Tags == nil {
		s.Tags = []string{}
	}
	return s, nil
}

// ListPublishedPublic 列出所有公开已发布文档，按更新时间倒序。可选 projectID 过滤。
func (r *Repository) ListPublishedPublic(projectID *int64) ([]DraftSummary, error) {
	where := `d.visibility = $1 AND d.status = $2`
	args := []any{VisibilityPublic, StatusPublished}
	if projectID != nil {
		where += ` AND d.project_id = $3`
		args = append(args, *projectID)
	}
	rows, err := r.db.Query(
		`SELECT `+summaryCols+` FROM blog_drafts d LEFT JOIN blog_projects p ON p.id = d.project_id
		 WHERE `+where+` ORDER BY COALESCE(d.updated_at, d.published_at) DESC`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DraftSummary{}
	for rows.Next() {
		s, err := scanDraftSummary(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

// GetPublishedPublicBySlug 取单篇公开已发布文档（含 markdown 正文）。
func (r *Repository) GetPublishedPublicBySlug(slug string) (*Draft, error) {
	d, err := scanDraft(func(dst ...any) error {
		return r.db.QueryRow(
			`SELECT `+draftCols+` FROM blog_drafts d LEFT JOIN blog_projects p ON p.id = d.project_id
			 WHERE d.slug = $1 AND d.visibility = $2 AND d.status = $3`,
			slug, VisibilityPublic, StatusPublished,
		).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}
	return d, nil
}

// SearchPublic 全文搜索公开已发布文档（首版 ILIKE）。返回 (结果, 总数)。可选 projectID 过滤。
func (r *Repository) SearchPublic(q string, projectID *int64, limit, offset int) ([]DraftSummary, int, error) {
	pattern := "%" + q + "%"
	where := `d.visibility = $1 AND d.status = $2 AND (d.title ILIKE $3 OR d.description ILIKE $3 OR d.markdown ILIKE $3)`
	args := []any{VisibilityPublic, StatusPublished, pattern}
	if projectID != nil {
		where += fmt.Sprintf(` AND d.project_id = $%d`, len(args)+1)
		args = append(args, *projectID)
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(
		`SELECT `+summaryCols+` FROM blog_drafts d LEFT JOIN blog_projects p ON p.id = d.project_id
		 WHERE `+where+` ORDER BY COALESCE(d.updated_at, d.published_at) DESC
		 LIMIT $`+fmt.Sprintf("%d", len(args)-1)+` OFFSET $`+fmt.Sprintf("%d", len(args)),
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []DraftSummary{}
	for rows.Next() {
		s, err := scanDraftSummary(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	countWhere := `d.visibility = $1 AND d.status = $2 AND (d.title ILIKE $3 OR d.description ILIKE $3 OR d.markdown ILIKE $3)`
	countArgs := []any{VisibilityPublic, StatusPublished, pattern}
	if projectID != nil {
		countWhere += fmt.Sprintf(` AND d.project_id = $%d`, len(countArgs)+1)
		countArgs = append(countArgs, *projectID)
	}
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM blog_drafts d WHERE `+countWhere,
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// ListPublishedPrivate 列出本人私有已发布文档。
func (r *Repository) ListPublishedPrivate(userID int64) ([]DraftSummary, error) {
	rows, err := r.db.Query(
		`SELECT `+summaryCols+` FROM blog_drafts d LEFT JOIN blog_projects p ON p.id = d.project_id
		 WHERE d.user_id = $1 AND d.visibility = $2 AND d.status = $3
		 ORDER BY COALESCE(d.updated_at, d.published_at) DESC`,
		userID, VisibilityPrivate, StatusPublished,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DraftSummary{}
	for rows.Next() {
		s, err := scanDraftSummary(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

// GetPublishedPrivateBySlug 取本人私有已发布文档（含 markdown 正文）。
func (r *Repository) GetPublishedPrivateBySlug(userID int64, slug string) (*Draft, error) {
	d, err := scanDraft(func(dst ...any) error {
		return r.db.QueryRow(
			`SELECT `+draftCols+` FROM blog_drafts d LEFT JOIN blog_projects p ON p.id = d.project_id
			 WHERE d.user_id = $1 AND d.slug = $2 AND d.visibility = $3 AND d.status = $4`,
			userID, slug, VisibilityPrivate, StatusPublished,
		).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}
	return d, nil
}

// DraftIsPublicPublished 判断草稿是否公开已发布（用于匿名读取其图片资产）。
func (r *Repository) DraftIsPublicPublished(draftID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM blog_drafts WHERE id = $1 AND visibility = $2 AND status = $3)`,
		draftID, VisibilityPublic, StatusPublished,
	).Scan(&exists)
	return exists, err
}

// ImportDraft 幂等导入：ON CONFLICT (slug) DO UPDATE。visibility 固定 public。
// published=true → status=published、published_version=version、published_at 用给定时间或 NOW()。
// 用于 import-blog CLI 把存量 Markdown 迁移入库。
func (r *Repository) ImportDraft(userID int64, d Draft, publishedAt *time.Time, published bool) (*Draft, error) {
	if !ValidSlug(d.Slug) {
		return nil, fmt.Errorf("invalid slug")
	}
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}
	status := StatusDraft
	var pubVersion *int64
	if published {
		status = StatusPublished
		v := d.Version
		if v == 0 {
			v = 1
		}
		pubVersion = &v
	}
	out, err := r.scanDraftRet(func(dst ...any) error {
		return r.db.QueryRow(
			`INSERT INTO blog_drafts (user_id, slug, title, description, tags, cover, markdown, status, visibility, version, published_version, published_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,COALESCE($12, NOW()))
			 ON CONFLICT (slug) DO UPDATE SET
			   title = EXCLUDED.title,
			   description = EXCLUDED.description,
			   tags = EXCLUDED.tags,
			   cover = EXCLUDED.cover,
			   markdown = EXCLUDED.markdown,
			   status = EXCLUDED.status,
			   visibility = EXCLUDED.visibility,
			   version = EXCLUDED.version,
			   published_version = EXCLUDED.published_version,
			   published_at = COALESCE(blog_drafts.published_at, EXCLUDED.published_at),
			   updated_at = NOW()
			 RETURNING `+draftRetCols,
			userID, d.Slug, d.Title, d.Description, pq.Array(tags), d.Cover, d.Markdown,
			status, VisibilityPublic, d.Version, pubVersion, publishedAt,
		).Scan(dst...)
	})
	if err != nil {
		return nil, err
	}
	// 导入后保存一个检查点，便于事后回滚。
	_ = r.insertCheckpoint(out)
	return out, nil
}

// ==================== DraftVersion ====================

const versionCols = "id, draft_id, version, title, description, tags, cover, markdown, created_at"

func (r *Repository) ListVersions(draftID int64) ([]DraftVersion, error) {
	rows, err := r.db.Query(
		`SELECT `+versionCols+` FROM blog_draft_versions WHERE draft_id = $1 ORDER BY version DESC LIMIT 100`,
		draftID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DraftVersion{}
	for rows.Next() {
		var v DraftVersion
		if err := rows.Scan(&v.ID, &v.DraftID, &v.Version, &v.Title, &v.Description, pq.Array(&v.Tags),
			&v.Cover, &v.Markdown, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetVersion 取指定版本快照（供恢复使用）。
func (r *Repository) GetVersion(draftID, version int64) (*DraftVersion, error) {
	var v DraftVersion
	err := r.db.QueryRow(
		`SELECT `+versionCols+` FROM blog_draft_versions WHERE draft_id = $1 AND version = $2`,
		draftID, version,
	).Scan(&v.ID, &v.DraftID, &v.Version, &v.Title, &v.Description, pq.Array(&v.Tags), &v.Cover, &v.Markdown, &v.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}
	return &v, nil
}

// RestoreVersion 把指定版本内容写回草稿为新版本（乐观推进，不含 baseVersion 冲突，
// 因为恢复是显式动作而非自动保存）。
func (r *Repository) RestoreVersion(userID, draftID, version int64) (*Draft, error) {
	v, err := r.GetVersion(draftID, version)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck
	// 恢复只是把历史内容写回草稿为新版本；不改变 status（线上文章仍存在，
	// 下次发布才更新线上）。若草稿当前为 published，恢复后内容与线上不一致是预期。
	d, err := r.scanDraftRet(func(dst ...any) error {
		return tx.QueryRow(
			`UPDATE blog_drafts
			   SET title = $1, description = $2, tags = $3, cover = $4, markdown = $5,
			       version = version + 1, updated_at = NOW()
			 WHERE id = $6 AND user_id = $7
			 RETURNING `+draftRetCols,
			v.Title, v.Description, pq.Array(v.Tags), v.Cover, v.Markdown, draftID, userID,
		).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}
	// 恢复也产生一个新版本快照
	if _, err := tx.Exec(
		`INSERT INTO blog_draft_versions (draft_id, version, title, description, tags, cover, markdown)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		d.ID, d.Version, d.Title, d.Description, pq.Array(d.Tags), d.Cover, d.Markdown,
	); err != nil {
		return nil, err
	}
	if err := trimVersions(tx, d.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return d, nil
}

// trimVersionsTx 与 trimVersions 同语义，但用 *sql.DB（供非事务场景备用）。
func trimVersionsTx(db *sql.DB, draftID int64) error {
	_, err := db.Exec(
		`DELETE FROM blog_draft_versions WHERE draft_id = $1 AND id NOT IN (
			SELECT id FROM blog_draft_versions WHERE draft_id = $1 ORDER BY version DESC LIMIT 10)`,
		draftID,
	)
	return err
}

// ==================== Asset ====================

const assetCols = "id, user_id, draft_id, sha256, filename, mime, size, staging_path, created_at"

func scanAsset(sc func(...any) error) (*Asset, error) {
	a := &Asset{}
	err := sc(&a.ID, &a.UserID, &a.DraftID, &a.SHA256, &a.Filename, &a.MIME, &a.Size,
		&a.StagingPath, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *Repository) CreateAsset(a Asset) (*Asset, error) {
	out, err := scanAsset(func(dst ...any) error {
		return r.db.QueryRow(
			`INSERT INTO blog_assets (user_id, draft_id, sha256, filename, mime, size, staging_path)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)
			 RETURNING `+assetCols,
			a.UserID, a.DraftID, a.SHA256, a.Filename, a.MIME, a.Size, a.StagingPath,
		).Scan(dst...)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) GetAsset(id int64) (*Asset, error) {
	a, err := scanAsset(func(dst ...any) error {
		return r.db.QueryRow(`SELECT `+assetCols+` FROM blog_assets WHERE id = $1`, id).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}
	return a, nil
}

// ==================== Project ====================

const projectCols = "id, user_id, name, intro, sort_order, created_at, updated_at"

func scanProject(sc func(...any) error) (*Project, error) {
	p := &Project{}
	err := sc(&p.ID, &p.UserID, &p.Name, &p.Intro, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *Repository) ListProjects(userID int64) ([]Project, error) {
	rows, err := r.db.Query(
		`SELECT `+projectCols+` FROM blog_projects WHERE user_id = $1 ORDER BY sort_order ASC, id ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Project{}
	for rows.Next() {
		p, err := scanProject(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// ListPublicProjectsWithCount 列出所有 project 及公开已发布文章数（无需鉴权）。
func (r *Repository) ListPublicProjectsWithCount() ([]ProjectPublic, error) {
	rows, err := r.db.Query(
		`SELECT p.id, p.name, p.intro, COUNT(d.id)
		 FROM blog_projects p
		 LEFT JOIN blog_drafts d ON d.project_id = p.id AND d.visibility = 'public' AND d.status = 'published'
		 GROUP BY p.id, p.name, p.intro, p.sort_order
		 ORDER BY p.sort_order ASC, p.id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProjectPublic{}
	for rows.Next() {
		var pp ProjectPublic
		if err := rows.Scan(&pp.ID, &pp.Name, &pp.Intro, &pp.PostCount); err != nil {
			return nil, err
		}
		out = append(out, pp)
	}
	return out, rows.Err()
}

func (r *Repository) CreateProject(userID int64, p Project) (*Project, error) {
	out, err := scanProject(func(dst ...any) error {
		return r.db.QueryRow(
			`INSERT INTO blog_projects (user_id, name, intro)
			 VALUES ($1,$2,$3)
			 RETURNING `+projectCols,
			userID, p.Name, p.Intro,
		).Scan(dst...)
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrConflict
		}
		return nil, err
	}
	return out, nil
}

func (r *Repository) GetProject(userID, id int64) (*Project, error) {
	p, err := scanProject(func(dst ...any) error {
		return r.db.QueryRow(
			`SELECT `+projectCols+` FROM blog_projects WHERE id = $1 AND user_id = $2`,
			id, userID,
		).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *Repository) UpdateProject(userID, id int64, req UpdateProjectRequest) (*Project, error) {
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	n := 1
	add := func(col, expr string, val any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, n))
		args = append(args, val)
		n++
	}
	if req.Name != nil {
		add("name", "", *req.Name)
	}
	if req.Intro != nil {
		add("intro", "", *req.Intro)
	}
	if req.SortOrder != nil {
		add("sort_order", "", *req.SortOrder)
	}
	args = append(args, id, userID)
	p, err := scanProject(func(dst ...any) error {
		return r.db.QueryRow(
			fmt.Sprintf(
				`UPDATE blog_projects SET %s WHERE id = $%d AND user_id = $%d RETURNING `+projectCols,
				strings.Join(sets, ", "), n, n+1,
			),
			args...,
		).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrConflict
		}
		return nil, err
	}
	return p, nil
}

func (r *Repository) DeleteProject(userID, id int64) error {
	res, err := r.db.Exec(`DELETE FROM blog_projects WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (r *Repository) ReorderProjects(userID int64, items []ReorderItem) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	for _, item := range items {
		if _, err := tx.Exec(
			`UPDATE blog_projects SET sort_order = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
			item.SortOrder, item.ID, userID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetDraftProject 切换文章归属（不含版本变更，不触发"未发布修改"）。
func (r *Repository) SetDraftProject(userID, id int64, projectID *int64) error {
	res, err := r.db.Exec(
		`UPDATE blog_drafts SET project_id = $1 WHERE id = $2 AND user_id = $3`,
		projectID, id, userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrDraftNotFound
	}
	return nil
}

// ==================== AuditLog ====================

func (r *Repository) InsertAudit(userID *int64, action, detail string) error {
	_, err := r.db.Exec(
		`INSERT INTO blog_audit_logs (user_id, action, detail) VALUES ($1,$2,$3)`,
		userID, action, detail,
	)
	return err
}
