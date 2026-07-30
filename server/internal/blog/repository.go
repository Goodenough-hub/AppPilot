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

const draftCols = "id, user_id, slug, title, description, tags, cover, markdown, status, version, published_commit_sha, published_version, published_at, created_at, updated_at"

func scanDraft(sc func(...any) error) (*Draft, error) {
	d := &Draft{}
	err := sc(&d.ID, &d.UserID, &d.Slug, &d.Title, &d.Description, pq.Array(&d.Tags),
		&d.Cover, &d.Markdown, &d.Status, &d.Version, &d.PublishedCommitSha, &d.PublishedVersion, &d.PublishedAt, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	// 派生：已发布但本地版本超过已发布版本 → 有未发布修改。
	if d.Status == StatusPublished && (d.PublishedVersion == nil || d.Version > *d.PublishedVersion) {
		d.HasUnpublishedChanges = true
	}
	return d, nil
}

func (r *Repository) ListDrafts(userID int64) ([]Draft, error) {
	rows, err := r.db.Query(
		`SELECT `+draftCols+` FROM blog_drafts WHERE user_id = $1 ORDER BY updated_at DESC`,
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
			`SELECT `+draftCols+` FROM blog_drafts WHERE id = $1 AND user_id = $2`,
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
			`SELECT `+draftCols+` FROM blog_drafts WHERE id = $1`,
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
	out, err := scanDraft(func(dst ...any) error {
		return r.db.QueryRow(
			`INSERT INTO blog_drafts (user_id, slug, title, description, tags, cover, markdown, status)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,'draft')
			 RETURNING `+draftCols,
			userID, d.Slug, d.Title, d.Description, pq.Array(tags), d.Cover, d.Markdown,
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
	args = append(args, id, userID, baseVersion) // $n, $n+1, $n+2

	d, err := scanDraft(func(dst ...any) error {
		return r.db.QueryRow(
			fmt.Sprintf(
				`UPDATE blog_drafts SET %s WHERE id = $%d AND user_id = $%d AND version = $%d
				 RETURNING `+draftCols,
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

// insertCheckpoint 插入当前草稿状态的快照，(draft_id,version) 冲突则跳过，并裁剪到 100 条。
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

// trimVersions 仅保留单篇最近 100 个版本快照。
func trimVersions(tx *sql.Tx, draftID int64) error {
	_, err := tx.Exec(
		`DELETE FROM blog_draft_versions WHERE draft_id = $1 AND id NOT IN (
			SELECT id FROM blog_draft_versions WHERE draft_id = $1 ORDER BY version DESC LIMIT 100)`,
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

// SetPublished 标记草稿已发布：status=published、记录 commit SHA。
func (r *Repository) SetPublished(id int64, commitSha string) error {
	_, err := r.db.Exec(
		`UPDATE blog_drafts SET status = $1, published_commit_sha = $2, updated_at = NOW() WHERE id = $3`,
		StatusPublished, commitSha, id,
	)
	return err
}

// EnsurePublishedAt 首次发布时固定 published_at（仅当仍为 NULL 时写入）。
// 后续更新只改 updatedAt，publishedAt 保持首次发布时间不变。
func (r *Repository) EnsurePublishedAt(id int64) error {
	_, err := r.db.Exec(
		`UPDATE blog_drafts SET published_at = NOW() WHERE id = $1 AND published_at IS NULL`,
		id,
	)
	return err
}

// FindBuildingJobByCommit 按 Git commit SHA 查找未终结的 job。
// GitHub Actions 无法获知 AppPilot 内部 jobId，回调只能凭 commitSha 定位 job。
func (r *Repository) FindBuildingJobByCommit(commitSha string) (*PublishJob, error) {
	if commitSha == "" {
		return nil, nil
	}
	j, err := scanJob(func(dst ...any) error {
		return r.db.QueryRow(
			`SELECT `+jobCols+` FROM blog_publish_jobs WHERE commit_sha = $1 AND status = 'building'
			 ORDER BY created_at DESC LIMIT 1`,
			commitSha,
		).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return j, nil
}

// FindJobByCommitAnySha 按 commitSha 查找任意状态的 job（幂等：已终结 job 不重复处理）。
func (r *Repository) FindJobByCommitAnySha(commitSha string) (*PublishJob, error) {
	if commitSha == "" {
		return nil, nil
	}
	j, err := scanJob(func(dst ...any) error {
		return r.db.QueryRow(
			`SELECT `+jobCols+` FROM blog_publish_jobs WHERE commit_sha = $1
			 ORDER BY created_at DESC LIMIT 1`,
			commitSha,
		).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return j, nil
}

// MarkDraftAssetsPublished 已由 PromoteDraftAssetsPublished 取代（回调提升暂存路径）。
// 保留旧名以兼容调用方，内部转调 PromoteDraftAssetsPublished。
func (r *Repository) MarkDraftAssetsPublished(draftID int64, _ string) error {
	return r.PromoteDraftAssetsPublished(draftID)
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
	d, err := scanDraft(func(dst ...any) error {
		return tx.QueryRow(
			`UPDATE blog_drafts
			   SET title = $1, description = $2, tags = $3, cover = $4, markdown = $5,
			       version = version + 1, updated_at = NOW()
			 WHERE id = $6 AND user_id = $7
			 RETURNING `+draftCols,
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
			SELECT id FROM blog_draft_versions WHERE draft_id = $1 ORDER BY version DESC LIMIT 100)`,
		draftID,
	)
	return err
}

// ==================== Asset ====================

const assetCols = "id, user_id, draft_id, sha256, filename, mime, size, staging_path, publish_path, published_path, created_at"

func scanAsset(sc func(...any) error) (*Asset, error) {
	a := &Asset{}
	err := sc(&a.ID, &a.UserID, &a.DraftID, &a.SHA256, &a.Filename, &a.MIME, &a.Size,
		&a.StagingPath, &a.PublishPath, &a.PublishedPath, &a.CreatedAt)
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

// ListDraftAssets 返回某草稿下所有图片（发布时一并提交）。
func (r *Repository) ListDraftAssets(draftID int64) ([]Asset, error) {
	rows, err := r.db.Query(`SELECT `+assetCols+` FROM blog_assets WHERE draft_id = $1 ORDER BY id`, draftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Asset{}
	for rows.Next() {
		a, err := scanAsset(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// SetAssetPublishPath 暂存计划公开路径（发布提交前写，与 Git 提交的文件路径一致）。
// 它不是 published_path：仅当发布成功回调时才把 publish_path 提升为 published_path。
func (r *Repository) SetAssetPublishPath(id int64, publishPath string) error {
	_, err := r.db.Exec(`UPDATE blog_assets SET publish_path = $1 WHERE id = $2`, publishPath, id)
	return err
}

// PromoteDraftAssetsPublished 发布成功回调：把暂存 publish_path 提升为 published_path。
func (r *Repository) PromoteDraftAssetsPublished(draftID int64) error {
	_, err := r.db.Exec(
		`UPDATE blog_assets SET published_path = publish_path WHERE draft_id = $1 AND publish_path IS NOT NULL`,
		draftID,
	)
	return err
}

// ==================== PublishJob ====================

const jobCols = "id, draft_id, draft_version, action, commit_sha, status, error, created_at, updated_at"

func scanJob(sc func(...any) error) (*PublishJob, error) {
	j := &PublishJob{}
	err := sc(&j.ID, &j.DraftID, &j.DraftVersion, &j.Action, &j.CommitSha, &j.Status, &j.Error, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (r *Repository) CreateJob(draftID int64, action string, draftVersion int64) (*PublishJob, error) {
	j, err := scanJob(func(dst ...any) error {
		return r.db.QueryRow(
			`INSERT INTO blog_publish_jobs (draft_id, draft_version, action, status) VALUES ($1,$2,$3,'queued')
			 RETURNING `+jobCols,
			draftID, draftVersion, action,
		).Scan(dst...)
	})
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (r *Repository) GetJob(id int64) (*PublishJob, error) {
	j, err := scanJob(func(dst ...any) error {
		return r.db.QueryRow(`SELECT `+jobCols+` FROM blog_publish_jobs WHERE id = $1`, id).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	return j, nil
}

// FindActiveJobByDraft 返回某草稿未终结的 job（queued/building），用于发布幂等。
func (r *Repository) FindActiveJobByDraft(draftID int64) (*PublishJob, error) {
	j, err := scanJob(func(dst ...any) error {
		return r.db.QueryRow(
			`SELECT `+jobCols+` FROM blog_publish_jobs WHERE draft_id = $1 AND status IN ('queued','building')
			 ORDER BY created_at DESC LIMIT 1`,
			draftID,
		).Scan(dst...)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return j, nil
}

// ListBuildingWithSha 返回所有 building 且已有 commit_sha 的 job（启动崩溃恢复用）。
func (r *Repository) ListBuildingWithSha() ([]PublishJob, error) {
	rows, err := r.db.Query(`SELECT ` + jobCols + ` FROM blog_publish_jobs WHERE status = 'building' AND commit_sha IS NOT NULL AND commit_sha <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PublishJob{}
	for rows.Next() {
		j, err := scanJob(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, rows.Err()
}

// MarkStaleQueuedFailed 把早于 cutoff 的 queued job 标记失败（进程崩溃后允许重新发布）。
func (r *Repository) MarkStaleQueuedFailed(cutoff time.Time) (int64, error) {
	res, err := r.db.Exec(
		`UPDATE blog_publish_jobs SET status = 'failed', error = 'stale queued job on startup', updated_at = NOW()
		 WHERE status = 'queued' AND created_at < $1`,
		cutoff,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (r *Repository) SetJobBuilding(id int64, commitSha string) error {
	_, err := r.db.Exec(
		`UPDATE blog_publish_jobs SET status = 'building', commit_sha = $1, updated_at = NOW() WHERE id = $2`,
		commitSha, id,
	)
	return err
}

func (r *Repository) SetJobResult(id int64, status, errMsg string) error {
	_, err := r.db.Exec(
		`UPDATE blog_publish_jobs SET status = $1, error = $2, updated_at = NOW() WHERE id = $3`,
		status, errMsg, id,
	)
	return err
}

// ==================== AuditLog ====================

func (r *Repository) InsertAudit(userID *int64, action, detail string) error {
	_, err := r.db.Exec(
		`INSERT INTO blog_audit_logs (user_id, action, detail) VALUES ($1,$2,$3)`,
		userID, action, detail,
	)
	return err
}
