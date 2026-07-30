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

// Publish job 状态：
//
//	queued   —— 已创建 job，尚未调用 Git
//	building —— Git 提交成功，等待 GitHub Actions 构建回调
//	succeeded —— 构建回调成功，草稿才标记为已发布/已撤回
//	failed   —— 构建或 Git 失败，线上版本不变
const (
	JobQueued    = "queued"
	JobBuilding  = "building"
	JobSucceeded = "succeeded"
	JobFailed    = "failed"
)

const (
	ActionPublish   = "publish"
	ActionUnpublish = "unpublish"
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
	ID                 int64      `json:"id"`
	UserID             int64      `json:"userId"`
	Slug               string     `json:"slug"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	Tags               []string   `json:"tags"`
	Cover              *string    `json:"cover"`
	Markdown           string     `json:"markdown"`
	Status             string     `json:"status"`
	Version            int64      `json:"version"`
	PublishedCommitSha *string    `json:"publishedCommitSha,omitempty"`
	PublishedAt        *time.Time `json:"publishedAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
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
	StagingPath   string    `json:"-"` // 服务端路径，不外泄
	PublishPath   *string   `json:"-"` // 计划公开路径（暂存，成功回调才转 published_path）
	PublishedPath *string   `json:"publishedPath,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type PublishJob struct {
	ID        int64     `json:"id"`
	DraftID   int64     `json:"draftId"`
	Action    string    `json:"action"`
	CommitSha *string   `json:"commitSha,omitempty"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AuditLog struct {
	ID        int64     `json:"id"`
	UserID    *int64    `json:"userId,omitempty"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"createdAt"`
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
}

// UpdateDraftRequest 必须提交 BaseVersion 做乐观锁；冲突返回 409。
type UpdateDraftRequest struct {
	Slug        *string  `json:"slug"`
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	Tags        []string `json:"tags"`
	Cover       *string  `json:"cover"`
	Markdown    *string  `json:"markdown"`
	BaseVersion int64    `json:"baseVersion" binding:"required"`
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

// ---- 发布回调请求体 ----
// GitHub Actions 无法获知 AppPilot 内部 jobId，回调以 commitSha 定位 job。
// jobId 可选，提供时用于二次校验。
type PublishCallbackRequest struct {
	JobID     int64  `json:"jobId"`
	CommitSha string `json:"commitSha" binding:"required"`
	Status    string `json:"status" binding:"required"` // succeeded | failed
	Error     string `json:"error"`
}
