package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists   = errors.New("username already exists")
	ErrUserNotFound = errors.New("user not found")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByUsername(username string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(
		`SELECT id, username, password_hash, role, app_scope, avatar, created_at, updated_at
		 FROM users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, pq.Array(&u.AppScope), &u.Avatar, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *Repository) FindByID(id int64) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(
		`SELECT id, username, password_hash, role, app_scope, avatar, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, pq.Array(&u.AppScope), &u.Avatar, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

// FindAdminCredentials 返回 admin 角色用户的 id 与密码哈希。
// 用户不存在或不是 admin 角色都返回 ErrUserNotFound。
// 供 blog 包交叉验证 admin 凭据用（blog login 端点：blog_users 未命中时降级查 users 表）。
func (r *Repository) FindAdminCredentials(username string) (int64, string, error) {
	u, err := r.FindByUsername(username)
	if err != nil {
		return 0, "", err
	}
	if u.Role != "admin" {
		return 0, "", ErrUserNotFound
	}
	return u.ID, u.PasswordHash, nil
}

func (r *Repository) List() ([]User, error) {
	rows, err := r.db.Query(
		`SELECT id, username, role, app_scope, avatar, created_at, updated_at FROM users ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, pq.Array(&u.AppScope), &u.Avatar, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repository) Create(username, password, role string, appScope []string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash: %w", err)
	}
	if appScope == nil {
		appScope = []string{}
	}
	u := &User{}
	err = r.db.QueryRow(
		`INSERT INTO users (username, password_hash, role, app_scope)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, username, role, app_scope, avatar, created_at, updated_at`,
		username, string(hash), role, pq.Array(appScope),
	).Scan(&u.ID, &u.Username, &u.Role, pq.Array(&u.AppScope), &u.Avatar, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return u, nil
}

func (r *Repository) UpdateAvatar(id int64, avatar string) error {
	_, err := r.db.Exec(`UPDATE users SET avatar = $1, updated_at = NOW() WHERE id = $2`, avatar, id)
	return err
}

// UserPatch describes a partial update. nil fields are left untouched.
// Password is plaintext; hashed inside Update.
type UserPatch struct {
	Role     *string
	AppScope *[]string
	Password *string
}

// Update applies a partial update in a single transaction.
// Role↔AppScope invariant: whenever role or app_scope is touched, the
// "admin" element inside app_scope is kept in sync with role:
//   - role="admin" → app_scope must contain "admin"
//   - role="user"  → app_scope must NOT contain "admin"
//
// If the caller only patches Password, role/app_scope columns are left alone
// (we do not "fix" pre-existing drift on unrelated writes).
func (r *Repository) Update(id int64, p UserPatch) (*User, error) {
	if p.Role == nil && p.AppScope == nil && p.Password == nil {
		return r.FindByID(id)
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var currRole string
	var currScope []string
	err = tx.QueryRow(
		`SELECT role, app_scope FROM users WHERE id = $1 FOR UPDATE`, id,
	).Scan(&currRole, pq.Array(&currScope))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	setClauses := []string{}
	args := []any{}
	arg := 1

	if p.Role != nil || p.AppScope != nil {
		newRole := currRole
		if p.Role != nil {
			newRole = *p.Role
		}
		var newScope []string
		if p.AppScope != nil {
			newScope = append([]string{}, (*p.AppScope)...)
		} else {
			newScope = append([]string{}, currScope...)
		}
		newScope = syncAdminScope(newScope, newRole)

		setClauses = append(setClauses, fmt.Sprintf("role = $%d", arg))
		args = append(args, newRole)
		arg++
		setClauses = append(setClauses, fmt.Sprintf("app_scope = $%d", arg))
		args = append(args, pq.Array(newScope))
		arg++
	}

	if p.Password != nil && *p.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*p.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("password_hash = $%d", arg))
		args = append(args, string(hash))
		arg++
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(
		`UPDATE users SET %s WHERE id = $%d
		 RETURNING id, username, role, app_scope, avatar, created_at, updated_at`,
		strings.Join(setClauses, ", "), arg,
	)

	u := &User{}
	err = tx.QueryRow(query, args...).Scan(
		&u.ID, &u.Username, &u.Role, pq.Array(&u.AppScope), &u.Avatar, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return u, nil
}

// syncAdminScope ensures the "admin" pseudo-app element is present iff role == "admin".
// Order-preserving; deduplicates any accidental duplicate "admin" entries.
func syncAdminScope(scope []string, role string) []string {
	out := make([]string, 0, len(scope)+1)
	seenAdmin := false
	for _, s := range scope {
		if s == "admin" {
			if seenAdmin {
				continue
			}
			seenAdmin = true
			if role != "admin" {
				continue
			}
		}
		out = append(out, s)
	}
	if role == "admin" && !seenAdmin {
		out = append(out, "admin")
	}
	return out
}

func (r *Repository) CreateAdmin(username, password string) (*User, error) {
	return r.Create(username, password, "admin", []string{"finflow", "typresume", "admin"})
}

func (r *Repository) Delete(id int64) error {
	res, err := r.db.Exec(`DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) VerifyPassword(u *User, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
}
