package blog

import "time"

// Draft 状态机：
//
//	draft      —— 草稿/已撤回（线上不可见）
//	published  —— 发布成功（线上可见，published_commit_sha 已记录）
//	publishing —— 发布 job 进行中（queued→building 期间）
//	unpublishing —— 撤回 job 进行中
//
// 失败的发布/撤回不改变线上旧版本，状态回退到上一个稳定态。
const (
	StatusDraft        = "draft"
	StatusPublished    = "published"
	StatusPublishing   = "publishing"
	StatusUnpublishing = "unpublishing"
)

// 可见性：public 公开（任何人可读），private 仅作者可读。
const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

// BlogUser 是 FluxBlog 独立账号，与 finflow/admin 的 users 表隔离。
// 软删除用 deleted_at；停用/删除时递增 token_version 使现有令牌立即失效。
type BlogUser struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	IsEnabled    bool       `json:"isEnabled"`
	TokenVersion int64      `json:"tokenVersion"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type Draft struct {
	ID                    int64      `json:"id"`
	UserID                int64      `json:"userId"`
	Slug                  string     `json:"slug"`
	Title                 string     `json:"title"`
	Description           string     `json:"description"`
	Tags                  []string   `json:"tags"`
	Cover                 *string    `json:"cover"`
	Markdown              string     `json:"markdown"`
	Status                string     `json:"status"`
	Visibility            string     `json:"visibility"`
	Version               int64      `json:"version"`
	PublishedCommitSha    *string    `json:"publishedCommitSha,omitempty"`
	PublishedVersion      *int64     `json:"publishedVersion,omitempty"`
	HasUnpublishedChanges bool       `json:"hasUnpublishedChanges"`
	ProjectID              *int64     `json:"projectId,omitempty"`
	ProjectName            *string    `json:"projectName,omitempty"`
	PublishedAt           *time.Time `json:"publishedAt,omitempty"`
	ScheduledPublishAt    *time.Time `json:"scheduledPublishAt,omitempty"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

type DraftVersion struct {
	ID          int64     `json:"id"`
	DraftID     int64     `json:"draftId"`
	Version     int64     `json:"version"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Cover       *string   `json:"cover"`
	Markdown    string    `json:"markdown"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Asset struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"userId"`
	DraftID       *int64    `json:"draftId,omitempty"`
	SHA256        string    `json:"sha256"`
	Filename      string    `json:"filename"`
	MIME          string    `json:"mime"`
	Size          int64     `json:"size"`
	StagingPath string `json:"-"` // 服务端路径，不外泄
	CreatedAt   time.Time `json:"createdAt"`
}

type AuditLog struct {
	ID        int64     `json:"id"`
	UserID    *int64    `json:"userId,omitempty"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"createdAt"`
}

// Project 是文章的归属分类，一篇文章只属于一个 project（可空）。
type Project struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Name      string    `json:"name"`
	Intro     string    `json:"intro"`
	SortOrder int       `json:"sortOrder"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ProjectPublic 是公开读场景的 project 简视图（含公开文章数）。
type ProjectPublic struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Intro     string `json:"intro"`
	PostCount int    `json:"postCount"`
}

// ---- 请求体 ----

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CreateDraftRequest struct {
	Slug        string   `json:"slug" binding:"required"`
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Cover       *string  `json:"cover"`
	Markdown    string   `json:"markdown"`
	Visibility  *string  `json:"visibility"`
	ProjectID   *int64   `json:"projectId"`
}

// UpdateDraftRequest 必须提交 BaseVersion 做乐观锁；冲突返回 409。
type UpdateDraftRequest struct {
	Slug        *string  `json:"slug"`
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	Tags        []string `json:"tags"`
	Cover       *string  `json:"cover"`
	Markdown    *string  `json:"markdown"`
	Visibility  *string  `json:"visibility"`
	ProjectID   *int64   `json:"projectId"`
	BaseVersion int64    `json:"baseVersion" binding:"required"`
}

// PublishRequest 可携带发布时的可见性、定时发布时间、项目归属、标签。
// 缺省字段保持原值；ScheduledPublishAt 非 nil 表示定时发布（status 保持 draft）。
type PublishRequest struct {
	Visibility         *string    `json:"visibility"`
	ScheduledPublishAt *time.Time `json:"scheduledPublishAt"`
	ProjectID          *int64     `json:"projectId"`
	Tags               []string   `json:"tags"`
}

// ---- Project 请求体 ----

type CreateProjectRequest struct {
	Name  string `json:"name" binding:"required"`
	Intro string `json:"intro"`
}

type UpdateProjectRequest struct {
	Name      *string `json:"name"`
	Intro     *string `json:"intro"`
	SortOrder *int    `json:"sortOrder"`
}

type ReorderItem struct {
	ID        int64 `json:"id"`
	SortOrder int   `json:"sortOrder"`
}

type ReorderProjectsRequest struct {
	Items []ReorderItem `json:"items" binding:"required"`
}

type SetDraftProjectRequest struct {
	ProjectID *int64 `json:"projectId"`
}

// DraftSummary 是列表场景的精简视图：不含 markdown 正文。
type DraftSummary struct {
	ID                int64      `json:"id"`
	Slug              string     `json:"slug"`
	Title             string     `json:"title"`
	Description       string     `json:"description"`
	Tags              []string   `json:"tags"`
	Cover             *string    `json:"cover,omitempty"`
	Status            string     `json:"status"`
	Visibility        string     `json:"visibility"`
	Version           int64      `json:"version"`
	ProjectID         *int64     `json:"projectId,omitempty"`
	ProjectName       *string    `json:"projectName,omitempty"`
	PublishedAt       *time.Time `json:"publishedAt,omitempty"`
	ScheduledPublishAt *time.Time `json:"scheduledPublishAt,omitempty"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

// ---- Admin 管理 blog 账号的请求体 ----

type CreateBlogUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6"`
}

type UpdateBlogUserRequest struct {
	Username  *string `json:"username"`
	IsEnabled *bool   `json:"isEnabled"`
}

type ResetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}
