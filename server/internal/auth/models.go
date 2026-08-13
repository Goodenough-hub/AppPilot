package auth

import (
	"strings"
	"time"
)

type User struct {
	ID           int64     `json:"id,string"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	AppScope     []string  `json:"appScope"`
	Avatar       string    `json:"avatar"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CreateUserRequest struct {
	Username string   `json:"username" binding:"required,min=3,max=64"`
	Password string   `json:"password" binding:"required,min=6"`
	Role     string   `json:"role"`
	AppScope []string `json:"appScope"`
}

type UpdateAvatarRequest struct {
	Avatar string `json:"avatar" binding:"required"`
}

// UpdateUserRequest is used by admin PATCH /admin/users/:id.
// All fields are pointers so we can distinguish "omitted" from "explicit empty":
//   - Role/AppScope: nil = don't change; non-nil = replace.
//   - Password: nil = don't change; non-nil non-empty = reset (bcrypt).
type UpdateUserRequest struct {
	Role     *string   `json:"role"`
	AppScope *[]string `json:"appScope"`
	Password *string   `json:"password" binding:"omitempty,min=6"`
}

type TokenResponse struct {
	Token     string   `json:"token"`
	ExpiresAt int64    `json:"expiresAt"`
	UserID    string   `json:"userId"`
	Role      string   `json:"role"`
	AppScope  []string `json:"appScope"`
	Username  string   `json:"username"`
	Avatar    string   `json:"avatar"`
}

// NormalizeRole 限制 role 只能是 user 或 admin，默认 user
func NormalizeRole(r string) string {
	r = strings.ToLower(strings.TrimSpace(r))
	if r == "admin" {
		return "admin"
	}
	return "user"
}
