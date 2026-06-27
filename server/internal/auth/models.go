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

type TokenResponse struct {
	Token     string   `json:"token"`
	ExpiresAt int64    `json:"expiresAt"`
	UserID    string   `json:"userId"`
	Role      string   `json:"role"`
	AppScope  []string `json:"appScope"`
	Username  string   `json:"username"`
}

// NormalizeRole 限制 role 只能是 user 或 admin，默认 user
func NormalizeRole(r string) string {
	r = strings.ToLower(strings.TrimSpace(r))
	if r == "admin" {
		return "admin"
	}
	return "user"
}
