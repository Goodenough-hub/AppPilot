package auth

import (
	"database/sql"
	"errors"
	"fmt"

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

func (r *Repository) CreateAdmin(username, password string) (*User, error) {
	return r.Create(username, password, "admin", []string{"finflow", "admin"})
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
