package blog

import (
	"crypto/sha256"
	"encoding/hex"
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

// Handler 是 FluxBlog 写作 API 入口。与 finflow/admin handler 隔离：
// 独立 JWT、独立 repo、独立鉴权中间件。发布/撤回为 DB 内同步状态翻转，不再走 Git。
type Handler struct {
	repo      *Repository
	jwtSecret string
	assetDir  string
}

func NewHandler(repo *Repository, jwtSecret string, assetDir string) *Handler {
	return &Handler{
		repo:      repo,
		jwtSecret: jwtSecret,
		assetDir:  assetDir,
	}
}

// configGuard 在缺少 Blog JWT 配置时统一返回 503，影响全部 blog 接口
// （含 login：无法签发令牌）。
func (h *Handler) configGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(h.jwtSecret) < 32 {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "blog not configured"})
			return
		}
		c.Next()
	}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.Use(h.configGuard())
	blogAuth := AuthRequired(h.repo, h.jwtSecret)

	ag := rg.Group("/auth")
	ag.POST("/login", h.login)
	ag.POST("/refresh", h.refresh)
	ag.POST("/logout", blogAuth, h.logout)
	ag.GET("/me", blogAuth, h.me)

	// 公开读：无需登录（仍受 configGuard 约束）。内容来自 blog_drafts，不再依赖 Git。
	pub := rg.Group("/posts")
	pub.GET("", h.listPublicPosts)
	pub.GET("/search", h.searchPublic)
	pub.GET("/:slug", h.getPublicPost)

	// 私有读：仅作者本人（blogAuth + 属主）。
	me := rg.Group("/me", blogAuth)
	me.GET("/posts", h.listMyPrivatePosts)
	me.GET("/posts/:slug", h.getMyPrivatePost)

	rg.GET("/drafts", blogAuth, h.listDrafts)
	rg.POST("/drafts", blogAuth, h.createDraft)
	rg.GET("/drafts/:id", blogAuth, h.getDraft)
	rg.PATCH("/drafts/:id", blogAuth, h.updateDraft)
	rg.DELETE("/drafts/:id", blogAuth, h.deleteDraft)
	rg.GET("/drafts/:id/versions", blogAuth, h.listVersions)
	rg.POST("/drafts/:id/versions", blogAuth, h.createCheckpoint)
	rg.POST("/drafts/:id/versions/:version/restore", blogAuth, h.restoreVersion)

	rg.POST("/assets", blogAuth, h.uploadAsset)
	rg.GET("/assets/:id", blogAuth, h.getAsset)

	rg.POST("/drafts/:id/publish", blogAuth, h.publish)
	rg.POST("/drafts/:id/unpublish", blogAuth, h.unpublish)
	rg.GET("/tags", blogAuth, h.listTags)

		// Project 管理（需登录）
		rg.POST("/projects", blogAuth, h.createProject)
		rg.PATCH("/projects/:id", blogAuth, h.updateProject)
		rg.DELETE("/projects/:id", blogAuth, h.deleteProject)
		rg.POST("/projects/reorder", blogAuth, h.reorderProjects)
		rg.PUT("/drafts/:id/project", blogAuth, h.setDraftProject)

		// 公开 project 读（无需登录）
		pubProjects := rg.Group("/projects")
		pubProjects.GET("", h.listPublicProjects)
		pubProjects.GET("/:id", h.getPublicProject)
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
	SetSessionCookies(c, token)
	c.JSON(http.StatusOK, tokenResponse{
		Token:     token,
		ExpiresAt: exp,
		UserID:    encodeID(u.ID),
		Username:  u.Username,
	})
}

func (h *Handler) refresh(c *gin.Context) {
	token := requestToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid auth header"})
		return
	}
	claims, err := ParseTokenForRefresh(token, h.jwtSecret)
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
	SetSessionCookies(c, token)
	c.JSON(http.StatusOK, tokenResponse{
		Token:     token,
		ExpiresAt: exp,
		UserID:    encodeID(u.ID),
		Username:  u.Username,
	})
}

func (h *Handler) logout(c *gin.Context) {
	ClearSessionCookies(c)
	_ = h.repo.InsertAudit(int64Ptr(blogUserID(c)), "logout", "")
	c.Status(http.StatusNoContent)
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

// ==================== 公开 / 私有读 ====================

// listPublicPosts 列出所有公开已发布文档（无正文，供首页/列表/RSS/sitemap）。
// 可选 query: ?projectId= 过滤指定 project 下的文章。
func (h *Handler) listPublicPosts(c *gin.Context) {
	projectID := parseOptionalID(c, "projectId")
	posts, err := h.repo.ListPublishedPublic(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, posts)
}

// getPublicPost 取单篇公开已发布文档（含 markdown 正文）。
func (h *Handler) getPublicPost(c *gin.Context) {
	slug := c.Param("slug")
	if !ValidSlug(slug) {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}
	d, err := h.repo.GetPublishedPublicBySlug(slug)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, d)
}

// searchPublic 全文搜索公开已发布文档。GET /posts/search?q=&page=&pageSize=&projectId=
func (h *Handler) searchPublic(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusOK, gin.H{"items": []any{}, "total": 0, "page": 1, "pageSize": 10})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	projectID := parseOptionalID(c, "projectId")
	items, total, err := h.repo.SearchPublic(q, projectID, pageSize, (page-1)*pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// listMyPrivatePosts 列出本人私有已发布文档。
func (h *Handler) listMyPrivatePosts(c *gin.Context) {
	posts, err := h.repo.ListPublishedPrivate(blogUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, posts)
}

// getMyPrivatePost 取本人私有已发布文档（含正文）。
func (h *Handler) getMyPrivatePost(c *gin.Context) {
	slug := c.Param("slug")
	if !ValidSlug(slug) {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}
	d, err := h.repo.GetPublishedPrivateBySlug(blogUserID(c), slug)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, d)
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
	d, err := h.repo.GetDraft(blogUserID(c), id)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 已发布草稿禁止直接删除，必须先撤回。
	if d.Status == StatusPublished {
		c.JSON(http.StatusConflict, gin.H{"error": "已发布草稿须先撤回再删除"})
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

// createCheckpoint 为当前草稿内容显式创建一个检查点（手动"保存版本"）。
func (h *Handler) createCheckpoint(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	d, err := h.repo.GetDraft(blogUserID(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}
	if err := h.repo.CreateCheckpoint(d); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"version": d.Version})
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
	// 仅接受 sniff 后的 JPEG/PNG/WebP/GIF；拒绝 SVG 与其他类型。
	switch mime {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅接受 JPEG/PNG/WebP/GIF"})
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

// publish 把草稿置为已发布或定时发布。
// 请求体可选字段：visibility（同时调整可见性）、scheduledPublishAt（定时）、projectId、tags。
// 缺省保持原值；scheduledPublishAt 非 nil 表示定时发布（status 保持 draft）。
func (h *Handler) publish(c *gin.Context) {
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
	var req PublishRequest
	_ = c.ShouldBindJSON(&req)
	// 校验定时时间：非 nil 时必须 > now（1 分钟容差防时钟漂移）
	if req.ScheduledPublishAt != nil {
		if req.ScheduledPublishAt.Before(time.Now().Add(-time.Minute)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "scheduledPublishAt 不能是过去时间"})
			return
		}
	}
	// 缺省 visibility 保持原值
	if req.Visibility == nil {
		v := d.Visibility
		req.Visibility = &v
	}
	// 已发布且无可见性变更、无待发布修改、无定时 → no-op
	if d.Status == StatusPublished && d.PublishedVersion != nil && d.Version == *d.PublishedVersion &&
		*req.Visibility == d.Visibility && req.ScheduledPublishAt == nil &&
		req.ProjectID == nil && req.Tags == nil {
		c.JSON(http.StatusOK, gin.H{
			"id":         encodeID(d.ID),
			"status":     StatusPublished,
			"visibility": d.Visibility,
			"noop":       true,
		})
		return
	}
	// 发布前创建检查点，便于事后回滚到发布时的内容。
	// 先把 req.Tags/req.ProjectID 应用到 d，确保快照里含用户在 PublishModal
	// 选的最终值——否则 PublishDraft 在 CreateCheckpoint 之后才写 blog_drafts.tags，
	// 公开读取走 published_version 快照时会拿到旧 tags。
	if req.Tags != nil {
		d.Tags = req.Tags
	}
	if req.ProjectID != nil {
		d.ProjectID = req.ProjectID
	}
	_ = h.repo.CreateCheckpoint(d)
	updated, err := h.repo.PublishDraft(d.ID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditAction := "publish"
	if req.ScheduledPublishAt != nil {
		auditAction = "schedule_publish"
	}
	_ = h.repo.InsertAudit(int64Ptr(blogUserID(c)), auditAction,
		fmt.Sprintf("%s@v%d", updated.Slug, updated.Version))
	if req.ScheduledPublishAt != nil {
		c.JSON(http.StatusOK, gin.H{
			"id":                encodeID(updated.ID),
			"status":            updated.Status,
			"visibility":        updated.Visibility,
			"scheduled":          true,
			"scheduledPublishAt": updated.ScheduledPublishAt,
			"updatedAt":         updated.UpdatedAt,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":               encodeID(updated.ID),
		"status":           updated.Status,
		"visibility":       updated.Visibility,
		"publishedVersion": updated.PublishedVersion,
		"publishedAt":      updated.PublishedAt,
		"updatedAt":        updated.UpdatedAt,
	})
}

// listTags 返回当前用户所有草稿的去重标签列表。
func (h *Handler) listTags(c *gin.Context) {
	tags, err := h.repo.ListTags(blogUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// unpublish 把已发布草稿撤回为草稿（保留 visibility，仅改 status）。
func (h *Handler) unpublish(c *gin.Context) {
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
	if d.Status != StatusPublished {
		c.JSON(http.StatusConflict, gin.H{"error": "仅已发布草稿可撤回", "status": d.Status})
		return
	}
	_ = h.repo.CreateCheckpoint(d)
	updated, err := h.repo.UnpublishDraft(d.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.repo.InsertAudit(int64Ptr(blogUserID(c)), "unpublish", updated.Slug)
	c.JSON(http.StatusOK, gin.H{
		"id":         encodeID(updated.ID),
		"status":     updated.Status,
		"visibility": updated.Visibility,
		"updatedAt":  updated.UpdatedAt,
	})
}

// ==================== Projects ====================

// listPublicProjects 列出所有 project（含公开文章数），无需鉴权。Studio 侧栏也复用此接口。
func (h *Handler) listPublicProjects(c *gin.Context) {
	projects, err := h.repo.ListPublicProjectsWithCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, projects)
}

// getPublicProject 取单个 project 元数据，无需鉴权。
func (h *Handler) getPublicProject(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	p, err := h.repo.GetProject(blogUserID(c), id)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) createProject(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.repo.CreateProject(blogUserID(c), Project{Name: req.Name, Intro: req.Intro})
	if err != nil {
		if errors.Is(err, ErrConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "项目名称已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) updateProject(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.repo.UpdateProject(blogUserID(c), id, req)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		if errors.Is(err, ErrConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "项目名称已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) deleteProject(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.repo.DeleteProject(blogUserID(c), id); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) reorderProjects(c *gin.Context) {
	var req ReorderProjectsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.ReorderProjects(blogUserID(c), req.Items); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// setDraftProject 切换文章归属（拖拽/下拉用）。不 bump version。
func (h *Handler) setDraftProject(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req SetDraftProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.SetDraftProject(blogUserID(c), id, req.ProjectID); err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ==================== Helpers ====================

func int64Ptr(v int64) *int64 { return &v }

// parseOptionalID 从 query 参数里解析可选 int64；空字符串返回 nil。
func parseOptionalID(c *gin.Context, key string) *int64 {
	s := c.Query(key)
	if s == "" {
		return nil
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &id
}
