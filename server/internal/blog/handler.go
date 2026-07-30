package blog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// jsonMustMarshal 将 v 序列化为 JSON；用于生成安全的 YAML 标量。
func jsonMustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("null")
	}
	return b
}

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }

// Handler 是 FluxBlog 写作 API 入口。与 finflow/admin handler 隔离：
// 独立 JWT、独立 repo、独立鉴权中间件。
type Handler struct {
	repo           *Repository
	jwtSecret      string
	publisher      *Publisher
	assetDir       string
	callbackSecret string
}

func NewHandler(repo *Repository, jwtSecret string, publisher *Publisher, assetDir, callbackSecret string) *Handler {
	return &Handler{
		repo:           repo,
		jwtSecret:      jwtSecret,
		publisher:      publisher,
		assetDir:       assetDir,
		callbackSecret: callbackSecret,
	}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	blogAuth := AuthRequired(h.repo, h.jwtSecret)

	ag := rg.Group("/auth")
	ag.POST("/login", h.login)
	ag.POST("/refresh", h.refresh)
	ag.GET("/me", blogAuth, h.me)

	rg.GET("/drafts", blogAuth, h.listDrafts)
	rg.POST("/drafts", blogAuth, h.createDraft)
	rg.GET("/drafts/:id", blogAuth, h.getDraft)
	rg.PATCH("/drafts/:id", blogAuth, h.updateDraft)
	rg.DELETE("/drafts/:id", blogAuth, h.deleteDraft)
	rg.GET("/drafts/:id/versions", blogAuth, h.listVersions)
	rg.POST("/drafts/:id/versions/:version/restore", blogAuth, h.restoreVersion)

	rg.POST("/assets", blogAuth, h.uploadAsset)
	rg.GET("/assets/:id", blogAuth, h.getAsset)

	rg.POST("/drafts/:id/publish", blogAuth, h.publish)
	rg.POST("/drafts/:id/unpublish", blogAuth, h.unpublish)
	rg.GET("/publish-jobs/:id", blogAuth, h.getJob)

	rg.POST("/deploy/callback", h.deployCallback)
}

// ==================== Auth ====================

func (h *Handler) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.repo.FindByUsernameActive(req.Username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !u.IsEnabled {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	if err := h.repo.VerifyPassword(u, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	token, exp, err := GenerateToken(u, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.repo.InsertAudit(&u.ID, "login", "")
	c.JSON(http.StatusOK, tokenResponse{
		Token:     token,
		ExpiresAt: exp,
		UserID:    encodeID(u.ID),
		Username:  u.Username,
	})
}

func (h *Handler) refresh(c *gin.Context) {
	parts := bearerParts(c)
	if parts == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid auth header"})
		return
	}
	claims, err := ParseTokenForRefresh(parts, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	u, err := h.repo.FindActiveByID(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	if !u.IsEnabled || u.TokenVersion != claims.TokenVersion {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	token, exp, err := GenerateToken(u, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tokenResponse{
		Token:     token,
		ExpiresAt: exp,
		UserID:    encodeID(u.ID),
		Username:  u.Username,
	})
}

func (h *Handler) me(c *gin.Context) {
	uid := blogUserID(c)
	u, err := h.repo.FindActiveByID(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"userId":       encodeID(u.ID),
		"username":     u.Username,
		"isEnabled":    u.IsEnabled,
		"tokenVersion": u.TokenVersion,
	})
}

// ==================== Drafts ====================

func (h *Handler) listDrafts(c *gin.Context) {
	drafts, err := h.repo.ListDrafts(blogUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, drafts)
}

func (h *Handler) createDraft(c *gin.Context) {
	var req CreateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !ValidSlug(req.Slug) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug 只允许小写字母、数字和连字符"})
		return
	}
	d, err := h.repo.CreateDraft(blogUserID(c), Draft{
		Slug:        req.Slug,
		Title:       req.Title,
		Description: req.Description,
		Tags:        req.Tags,
		Cover:       req.Cover,
		Markdown:    req.Markdown,
	})
	if err != nil {
		if errors.Is(err, ErrConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "slug 已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, d)
}

func (h *Handler) getDraft(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	d, err := h.repo.GetDraft(blogUserID(c), id)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, d)
}

func (h *Handler) updateDraft(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req UpdateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d, serverVersion, err := h.repo.UpdateDraft(blogUserID(c), id, req.BaseVersion, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrDraftNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		case errors.Is(err, ErrConflict):
			c.JSON(http.StatusConflict, gin.H{
				"error":         "version conflict",
				"serverVersion": serverVersion,
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, d)
}

func (h *Handler) deleteDraft(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.repo.DeleteDraft(blogUserID(c), id); err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listVersions(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if _, err := h.repo.GetDraft(blogUserID(c), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}
	vs, err := h.repo.ListVersions(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, vs)
}

func (h *Handler) restoreVersion(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	version, _ := strconv.ParseInt(c.Param("version"), 10, 64)
	if version <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version"})
		return
	}
	d, err := h.repo.RestoreVersion(blogUserID(c), id, version)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.repo.InsertAudit(int64Ptr(blogUserID(c)), "restore", fmt.Sprintf("v%d", version))
	c.JSON(http.StatusOK, d)
}

// ==================== Assets ====================

const maxAssetSize = 8 << 20 // 8 MiB

func (h *Handler) uploadAsset(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	if file.Size > maxAssetSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image too large (max 8MiB)"})
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()
	data, err := io.ReadAll(io.LimitReader(src, maxAssetSize+1))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if int64(len(data)) > maxAssetSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image too large (max 8MiB)"})
		return
	}
	mime := http.DetectContentType(data)
	if !strings.HasPrefix(mime, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only images allowed"})
		return
	}
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])
	// 扩展名按 mime 推断；浏览器端已转 WebP，这里保留 .webp 或原扩展名
	ext := ".webp"
	if strings.Contains(mime, "png") {
		ext = ".png"
	} else if strings.Contains(mime, "jpeg") || strings.Contains(mime, "jpg") {
		ext = ".jpg"
	}
	if err := os.MkdirAll(h.assetDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	staging := filepath.Join(h.assetDir, sha+ext)
	if err := os.WriteFile(staging, data, 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var draftID *int64
	if s := c.PostForm("draftId"); s != "" {
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			// 校验草稿属于当前用户，防止把图片绑定到他人草稿。
			if _, err := h.repo.GetDraft(blogUserID(c), id); err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
				return
			}
			draftID = &id
		}
	}
	a, err := h.repo.CreateAsset(Asset{
		UserID:      blogUserID(c),
		DraftID:     draftID,
		SHA256:      sha,
		Filename:    file.Filename,
		MIME:        mime,
		Size:        int64(len(data)),
		StagingPath: staging,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":         encodeID(a.ID),
		"sha256":     a.SHA256,
		"filename":   a.Filename,
		"mime":       a.MIME,
		"size":       a.Size,
		"previewUrl": "/api/v1/blog/assets/" + encodeID(a.ID),
		"createdAt":  a.CreatedAt,
	})
}

func (h *Handler) getAsset(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	a, err := h.repo.GetAsset(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}
	if a.UserID != blogUserID(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}
	c.Header("Content-Type", a.MIME)
	c.Header("Cache-Control", "private, max-age=60")
	c.File(a.StagingPath)
}

// ==================== Publish / Unpublish ====================

func (h *Handler) publish(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	d, err := h.repo.GetDraft(blogUserID(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}
	if h.publisher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "publishing not configured"})
		return
	}
	// 起始状态校验：仅 draft 可发布；publishing/unpublishing 进行中拒绝重复。
	if d.Status == StatusPublishing || d.Status == StatusUnpublishing {
		c.JSON(http.StatusConflict, gin.H{"error": "已有进行中的发布任务", "status": d.Status})
		return
	}
	// 幂等：已有进行中 job 直接返回
	if active, _ := h.repo.FindActiveJobByDraft(id); active != nil {
		c.JSON(http.StatusAccepted, gin.H{"jobId": encodeID(active.ID), "status": active.Status})
		return
	}
	job, err := h.repo.CreateJob(id, ActionPublish)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 首次发布固定 published_at；后续更新只改 updatedAt。
	pa := time.Now().UTC()
	if d.PublishedAt != nil {
		pa = d.PublishedAt.UTC()
	} else if err := h.repo.EnsurePublishedAt(id); err != nil {
		_ = h.repo.SetJobResult(job.ID, JobFailed, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	files, err := h.buildPublishFiles(d, false, pa)
	if err != nil {
		_ = h.repo.SetJobResult(job.ID, JobFailed, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	commitSHA, err := h.publisher.Commit(ctx, fmt.Sprintf("content(blog): 发布 %s", d.Slug), files)
	if err != nil {
		msg := err.Error()
		if errors.Is(err, ErrGitConflict) {
			_ = h.repo.SetJobResult(job.ID, JobFailed, "git conflict")
			c.JSON(http.StatusConflict, gin.H{"error": "git conflict", "jobId": encodeID(job.ID)})
			return
		}
		_ = h.repo.SetJobResult(job.ID, JobFailed, msg)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg, "jobId": encodeID(job.ID)})
		return
	}
	// Git 提交成功：job=building，等待 Actions 回调后才标记草稿 published。
	if err := h.repo.SetJobBuilding(job.ID, commitSHA); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.repo.SetDraftStatus(id, StatusPublishing)
	_ = h.repo.InsertAudit(int64Ptr(blogUserID(c)), "publish", fmt.Sprintf("%s@%s", d.Slug, commitSHA))
	c.JSON(http.StatusAccepted, gin.H{"jobId": encodeID(job.ID), "commitSha": commitSHA, "status": JobBuilding})
}

func (h *Handler) unpublish(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	d, err := h.repo.GetDraft(blogUserID(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}
	if h.publisher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "publishing not configured"})
		return
	}
	// 仅已发布草稿可撤回。
	if d.Status != StatusPublished {
		c.JSON(http.StatusConflict, gin.H{"error": "仅已发布草稿可撤回", "status": d.Status})
		return
	}
	if active, _ := h.repo.FindActiveJobByDraft(id); active != nil {
		c.JSON(http.StatusAccepted, gin.H{"jobId": encodeID(active.ID), "status": active.Status})
		return
	}
	job, err := h.repo.CreateJob(id, ActionUnpublish)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	pa := time.Now().UTC()
	if d.PublishedAt != nil {
		pa = d.PublishedAt.UTC()
	}
	files, err := h.buildPublishFiles(d, true, pa)
	if err != nil {
		_ = h.repo.SetJobResult(job.ID, JobFailed, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	commitSHA, err := h.publisher.Commit(ctx, fmt.Sprintf("content(blog): 撤回 %s", d.Slug), files)
	if err != nil {
		if errors.Is(err, ErrGitConflict) {
			_ = h.repo.SetJobResult(job.ID, JobFailed, "git conflict")
			c.JSON(http.StatusConflict, gin.H{"error": "git conflict", "jobId": encodeID(job.ID)})
			return
		}
		_ = h.repo.SetJobResult(job.ID, JobFailed, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "jobId": encodeID(job.ID)})
		return
	}
	if err := h.repo.SetJobBuilding(job.ID, commitSHA); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.repo.SetDraftStatus(id, StatusUnpublishing)
	_ = h.repo.InsertAudit(int64Ptr(blogUserID(c)), "unpublish", d.Slug)
	c.JSON(http.StatusAccepted, gin.H{"jobId": encodeID(job.ID), "commitSha": commitSHA, "status": JobBuilding})
}

func (h *Handler) getJob(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	job, err := h.repo.GetJob(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	// 仅允许查询本人草稿的 job
	if _, err := h.repo.GetDraft(blogUserID(c), job.DraftID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, job)
}

// ==================== Deploy callback ====================

func (h *Handler) deployCallback(c *gin.Context) {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !VerifyCallbackHMAC([]byte(h.callbackSecret), raw, c.GetHeader("Authorization")) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}
	var req PublishCallbackRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status != JobSucceeded && req.Status != JobFailed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}
	// 按 commitSha 定位 job：优先 building，其次任意状态（幂等重复回调）。
	job, err := h.repo.FindBuildingJobByCommit(req.CommitSha)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if job == nil {
		job, err = h.repo.FindJobByCommitAnySha(req.CommitSha)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// 幂等：已终结 job 重复回调直接返回当前状态。
		if job != nil && (job.Status == JobSucceeded || job.Status == JobFailed) {
			c.JSON(http.StatusOK, gin.H{"jobId": encodeID(job.ID), "status": job.Status})
			return
		}
	}
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no job for commitSha"})
		return
	}
	// 二次校验：若回调带了 jobId，必须与按 SHA 查到的 job 一致。
	if req.JobID != 0 && req.JobID != job.ID {
		c.JSON(http.StatusConflict, gin.H{"error": "jobId does not match commitSha"})
		return
	}

	if req.Status == JobSucceeded {
		if err := h.applyJobSucceeded(job); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_ = h.repo.InsertAudit(nil, "callback", fmt.Sprintf("job %d succeeded", job.ID))
		c.JSON(http.StatusOK, gin.H{"jobId": encodeID(job.ID), "status": JobSucceeded})
		return
	}
	// 失败：回滚草稿到上一个稳定态，线上版本不变。
	if err := h.applyJobFailed(job, req.Error); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"jobId": encodeID(job.ID), "status": JobFailed})
}

// applyJobSucceeded 在单个事务里：标记 job 成功 + 草稿状态/发布 SHA + 提升图片 published_path。
// 任一步失败整体回滚，避免出现“job 成功但草稿仍 publishing”。
func (h *Handler) applyJobSucceeded(job *PublishJob) error {
	tx, err := h.repo.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.Exec(
		`UPDATE blog_publish_jobs SET status = 'succeeded', error = '', updated_at = NOW() WHERE id = $1`,
		job.ID,
	); err != nil {
		return err
	}
	sha := ""
	if job.CommitSha != nil {
		sha = *job.CommitSha
	}
	if job.Action == ActionPublish {
		if _, err := tx.Exec(
			`UPDATE blog_drafts SET status = 'published', published_commit_sha = $1, updated_at = NOW() WHERE id = $2`,
			sha, job.DraftID,
		); err != nil {
			return err
		}
		// 把暂存 publish_path 提升为 published_path（与 Git 提交的文件路径一致）。
		if _, err := tx.Exec(
			`UPDATE blog_assets SET published_path = publish_path WHERE draft_id = $1 AND publish_path IS NOT NULL`,
			job.DraftID,
		); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(
			`UPDATE blog_drafts SET status = 'draft', updated_at = NOW() WHERE id = $1`,
			job.DraftID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// applyJobFailed 标记 job 失败并把草稿回滚到上一个稳定态（不改变线上版本）。
func (h *Handler) applyJobFailed(job *PublishJob, errMsg string) error {
	tx, err := h.repo.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.Exec(
		`UPDATE blog_publish_jobs SET status = 'failed', error = $1, updated_at = NOW() WHERE id = $2`,
		errMsg, job.ID,
	); err != nil {
		return err
	}
	prev := StatusPublished
	if job.Action == ActionPublish {
		prev = StatusDraft
	}
	if _, err := tx.Exec(
		`UPDATE blog_drafts SET status = $1, updated_at = NOW() WHERE id = $2`,
		prev, job.DraftID,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// ==================== Helpers ====================

// buildPublishFiles 组装提交到 Git 的文件：Markdown 正文 + 草稿图片。
// draftFlag=true 表示撤回（frontmatter draft:true，不提交新图片）。
// publishedAt 为首次发布固定时间（后续更新只改 updatedAt）；now 为本次提交时间。
// 发布时把正文里 /api/v1/blog/assets/:id 的预览链接改写为公开 /blog/media/...，
// 并把计划公开路径暂存到 asset.publish_path（仅成功回调才提升为 published_path）。
func (h *Handler) buildPublishFiles(d *Draft, draftFlag bool, publishedAt time.Time) ([]GitFile, error) {
	now := time.Now().UTC()
	ym := now.Format("2006/01")
	md := d.Markdown
	var cover = d.Cover

	files := []GitFile{}
	if !draftFlag {
		assets, err := h.repo.ListDraftAssets(d.ID)
		if err != nil {
			return nil, err
		}
		for _, a := range assets {
			data, err := os.ReadFile(a.StagingPath)
			if err != nil {
				return nil, err
			}
			pubPath := "/blog/media/" + ym + "/" + a.SHA256 + extOf(a.MIME)
			repoPath := "public/media/" + ym + "/" + a.SHA256 + extOf(a.MIME)
			files = append(files, GitFile{Path: repoPath, Content: data})
			// 改写正文与封面里的受保护预览链接
			preview := "/api/v1/blog/assets/" + encodeID(a.ID)
			md = strings.ReplaceAll(md, preview, pubPath)
			if cover != nil && *cover == preview {
				cover = strPtr(pubPath)
			}
			// 暂存计划路径（不在此标记为已发布，避免提交/部署失败留下错误状态）
			if err := h.repo.SetAssetPublishPath(a.ID, pubPath); err != nil {
				return nil, err
			}
		}
	}

	body := assembleFrontmatter(d, cover, draftFlag, publishedAt, now) + md + "\n"
	// FluxBlog 内容集合只加载 src/content/posts，必须写到这里文章才会被收录。
	files = append([]GitFile{{Path: PostFilePath(d.Slug), Content: []byte(body)}}, files...)
	return files, nil
}

// assembleFrontmatter 生成合法 YAML frontmatter：以 --- 闭合，正文后空一行；
// 无封面时省略 cover（schema cover 为 optional，不接受 null）。
// 用 JSON 序列化各值以避免转义 bug（JSON 字符串与流式数组都是合法 YAML 标量/序列）。
func assembleFrontmatter(d *Draft, cover *string, draftFlag bool, publishedAt, now time.Time) string {
	tags, _ := jsonMarshal(d.Tags)
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: %s\n", jsonMustMarshal(d.Title)))
	b.WriteString(fmt.Sprintf("slug: %s\n", jsonMustMarshal(d.Slug)))
	b.WriteString(fmt.Sprintf("description: %s\n", jsonMustMarshal(d.Description)))
	b.WriteString(fmt.Sprintf("publishedAt: %s\n", publishedAt.Format("2006-01-02")))
	b.WriteString(fmt.Sprintf("updatedAt: %s\n", now.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("draft: %t\n", draftFlag))
	b.WriteString(fmt.Sprintf("tags: %s\n", tags))
	if cover != nil && *cover != "" {
		b.WriteString(fmt.Sprintf("cover: %s\n", jsonMustMarshal(*cover)))
	}
	b.WriteString("---\n\n")
	return b.String()
}

func extOf(mime string) string {
	switch {
	case strings.Contains(mime, "png"):
		return ".png"
	case strings.Contains(mime, "jpeg") || strings.Contains(mime, "jpg"):
		return ".jpg"
	case strings.Contains(mime, "webp"):
		return ".webp"
	case strings.Contains(mime, "gif"):
		return ".gif"
	default:
		return ".webp"
	}
}

func bearerParts(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func int64Ptr(v int64) *int64 { return &v }
