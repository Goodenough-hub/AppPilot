package blog

import (
	"crypto/rand"
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
	"golang.org/x/crypto/bcrypt"
)

// AdminUserVerifier 是 admin users 表的只读访问接口（由 cmd 注入 auth.Repository
// 实现）。blog 包不直接依赖 internal/auth，保持与 admin 用户体系隔离；
// 仅在 login 交叉验证、admin-preview 端点身份校验两个场景使用。
type AdminUserVerifier interface {
	// FindAdminCredentials 返回 admin 角色用户的 id 与密码哈希。
	// 用户不存在或非 admin 角色都返回 auth.ErrUserNotFound（接口约定返回 error）。
	FindAdminCredentials(username string) (int64, string, error)
}

// Handler 是 FluxBlog 写作 API 入口。与 finflow/admin handler 隔离：
// 独立 JWT、独立 repo、独立鉴权中间件。发布/撤回为 DB 内同步状态翻转，不再走 Git。
type Handler struct {
	repo          *Repository
	jwtSecret     string
	assetDir      string
	adminVerifier AdminUserVerifier // 可空：未注入时 blog login 不做交叉验证、admin-preview 端点 503
}

func NewHandler(repo *Repository, jwtSecret string, assetDir string, adminVerifier AdminUserVerifier) *Handler {
	return &Handler{
		repo:          repo,
		jwtSecret:     jwtSecret,
		assetDir:      assetDir,
		adminVerifier: adminVerifier,
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

	// admin 预览读：blog JWT 鉴权 + admin 身份校验。展示所有已发布文章（公开+私有），
	// 供 FluxBlog /blog/preview/ 页面使用。普通 blog 用户调此端点会被 403 挡住。
	preview := rg.Group("/admin-preview", blogAuth, h.adminOnlyGuard())
	preview.GET("/posts", h.listAllPostsForAdmin)
	preview.GET("/posts/:slug", h.getAdminPreviewPost)
	preview.GET("/projects", h.listAllProjectsForAdmin)

	// Studio 写作 API：仅 admin 可访问。普通 blog 用户调任意 /drafts* 端点均 403，
	// 因为第一版 FluxBlog Studio 是 admin 专区，普通用户只读公开博客前台。
	studio := rg.Group("/drafts", blogAuth, h.adminOnlyGuard())
	studio.GET("", h.listDrafts)
	studio.POST("", h.createDraft)
	studio.GET("/:id", h.getDraft)
	studio.PATCH("/:id", h.updateDraft)
	studio.DELETE("/:id", h.deleteDraft)
	studio.GET("/:id/versions", h.listVersions)
	studio.POST("/:id/versions", h.createCheckpoint)
	studio.POST("/:id/versions/:version/restore", h.restoreVersion)
	studio.POST("/:id/publish", h.publish)
	studio.POST("/:id/unpublish", h.unpublish)
	studio.PUT("/:id/project", h.setDraftProject)

	// 资源上传同样仅 admin
	assetsAdmin := rg.Group("/assets", blogAuth, h.adminOnlyGuard())
	assetsAdmin.POST("", h.uploadAsset)
	assetsAdmin.GET("/:id", h.getAsset)

	// Tag 列表 + Project 管理：仅 admin
	studioAdmin := rg.Group("", blogAuth, h.adminOnlyGuard())
	studioAdmin.GET("/tags", h.listTags)
	studioAdmin.POST("/projects", h.createProject)
	studioAdmin.PATCH("/projects/:id", h.updateProject)
	studioAdmin.DELETE("/projects/:id", h.deleteProject)
	studioAdmin.POST("/projects/reorder", h.reorderProjects)

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
	// 优先按 blog_users 表登录（独立博客账号）。命中即走原有 bcrypt 校验流程。
	u, err := h.repo.FindByUsernameActive(req.Username)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, ErrUserNotFound) {
		// blog_users 未命中：尝试 admin 凭据交叉验证（仅当注入了 adminVerifier）。
		// 命中 users 表且 role=admin 且密码对：自动建/查 blog_users stub 后签发 blog JWT，
		// 让 admin 用户能用同一套凭据登录 FluxBlog 预览页（/blog/preview/）。
		u = h.tryAdminCrossLogin(req.Username, req.Password)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
			return
		}
	} else {
		// blog_users 命中：原校验链
		if !u.IsEnabled {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
			return
		}
		if err := h.repo.VerifyPassword(u, req.Password); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
			return
		}
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

// tryAdminCrossLogin 用 admin 凭据交叉登录：查 users 表中 role=admin 的同名用户，
// 校验 bcrypt 密码；通过则查找或自动创建 blog_users stub（随机密码，admin 永不需要），
// 返回可用于签发 blog JWT 的 *BlogUser。任何环节失败均返回 nil（不泄露具体原因，
// 与原 login 错误一致）。
func (h *Handler) tryAdminCrossLogin(username, password string) *BlogUser {
	if h.adminVerifier == nil {
		return nil
	}
	_, hash, err := h.adminVerifier.FindAdminCredentials(username)
	if err != nil {
		return nil
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil
	}
	// 复用 admin SSO 逻辑：查找或创建 blog_users stub。stub 的随机密码 admin 永不需要。
	u, err := h.repo.FindByUsernameActive(username)
	if err == nil {
		if !u.IsEnabled {
			return nil
		}
		return u
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil
	}
	pw, err := randomHex(32)
	if err != nil {
		return nil
	}
	u, err = h.repo.Create(username, pw)
	if err != nil {
		// 并发：同 admin 用户名已被另一请求创建 → 重查
		if errors.Is(err, ErrUserExists) {
			u, err = h.repo.FindByUsernameActive(username)
			if err != nil || !u.IsEnabled {
				return nil
			}
			return u
		}
		return nil
	}
	_ = h.repo.InsertAudit(nil, "admin_cross_login_create", u.Username)
	return u
}

// randomHex 返回 n 字节的十六进制字符串（n=32 时为 64 字符）。
// 用于为 admin 交叉登录自动创建的 blog_users stub 生成永不复用的随机密码。
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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
		"isAdmin":      h.isAdminUser(c),
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
	// admin 用户返回所有草稿（含其他用户的），普通用户只返回自己的。
	// 复用 adminOnlyGuard 的判断逻辑：通过 adminVerifier 校验 username 对应 users 表 admin 角色。
	var drafts []Draft
	var err error
	if h.isAdminUser(c) {
		drafts, err = h.repo.ListAllDrafts()
	} else {
		drafts, err = h.repo.ListDrafts(blogUserID(c))
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, drafts)
}

// isAdminUser 判断当前 blog 用户是否对应 users 表的 admin 角色。
// 与 adminOnlyGuard 同源，但不 abort；供 listDrafts 等需要分支判断的 handler 复用。
// adminVerifier 未注入时返回 false（退化到普通用户行为）。
func (h *Handler) isAdminUser(c *gin.Context) bool {
	if h.adminVerifier == nil {
		return false
	}
	v, ok := c.Get("blogUsername")
	if !ok {
		return false
	}
	username, _ := v.(string)
	if username == "" {
		return false
	}
	if _, _, err := h.adminVerifier.FindAdminCredentials(username); err != nil {
		return false
	}
	return true
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
			"id":                 encodeID(updated.ID),
			"status":             updated.Status,
			"visibility":         updated.Visibility,
			"scheduled":          true,
			"scheduledPublishAt": updated.ScheduledPublishAt,
			"updatedAt":          updated.UpdatedAt,
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
	p, err := h.repo.GetProjectByID(id)
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

// ==================== admin 预览 ====================
// /admin-preview/* 供 FluxBlog /blog/preview/ 页面拉取所有已发布文章（公开+私有）。
// 走 blogAuth（fluxblog_token cookie 或 Authorization: Bearer）解出 blogUserID，
// 再用 adminOnlyGuard 校验该 blog user 的 username 对应 users 表的 admin 角色。
// 普通 blog 用户调此端点直接 403，避免泄露他人 private 文章。

// adminOnlyGuard 校验当前 blog 用户是否对应 admin 角色用户。
// 实现：从 ctx 取 blogUsername（由 blogAuth 注入），查 users 表 role=admin。
// 未注入 adminVerifier 时 503（服务端配置错误）。
func (h *Handler) adminOnlyGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.adminVerifier == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "admin verifier not configured"})
			return
		}
		v, ok := c.Get("blogUsername")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing auth"})
			return
		}
		username, _ := v.(string)
		if username == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing username"})
			return
		}
		if _, _, err := h.adminVerifier.FindAdminCredentials(username); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin only"})
			return
		}
		c.Next()
	}
}

// listAllPostsForAdmin 返回所有已发布文章（公开+私有）。
func (h *Handler) listAllPostsForAdmin(c *gin.Context) {
	posts, err := h.repo.ListAllPublished()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, posts)
}

// getAdminPreviewPost 返回单篇已发布文章（含 markdown 正文），按 slug 查询。
func (h *Handler) getAdminPreviewPost(c *gin.Context) {
	slug := c.Param("slug")
	d, err := h.repo.GetPublishedBySlug(slug)
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

// listAllProjectsForAdmin 返回所有 project 及已发布文章数（公开+私有）。
// 供 FluxBlog /blog/preview/projects 页面使用。
func (h *Handler) listAllProjectsForAdmin(c *gin.Context) {
	projects, err := h.repo.ListAllProjectsWithCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, projects)
}
