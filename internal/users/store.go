// Package users persists user accounts and their credentials.
package users

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/djdietrick/radio/internal/auth"
	"github.com/djdietrick/radio/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound is returned when no user matches a lookup.
var ErrNotFound = errors.New("users: not found")

// ErrDuplicate is returned when creating a user whose username already exists.
var ErrDuplicate = errors.New("users: username already exists")

// User is an account record. PasswordHash is never serialized.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	IsAdmin      bool      `json:"isAdmin"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Store provides user persistence.
type Store struct {
	db *db.DB
}

func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// ByUsername looks up a user by username (case-insensitive), returning
// ErrNotFound when absent.
func (s *Store) ByUsername(ctx context.Context, username string) (*User, error) {
	return s.scanOne(ctx, `
		SELECT id, username, is_admin, password_hash, created_at
		FROM users WHERE lower(username) = lower($1)`, username)
}

// ByID looks up a user by id.
func (s *Store) ByID(ctx context.Context, id string) (*User, error) {
	return s.scanOne(ctx, `
		SELECT id, username, is_admin, password_hash, created_at
		FROM users WHERE id = $1`, id)
}

// Create inserts a new user with the given credentials and admin flag.
func (s *Store) Create(ctx context.Context, username, password string, isAdmin bool) (*User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, errors.New("users: username and password are required")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	var u User
	err = s.db.Pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, is_admin)
		VALUES ($1, $2, $3)
		RETURNING id, username, is_admin, password_hash, created_at`,
		username, hash, isAdmin,
	).Scan(&u.ID, &u.Username, &u.IsAdmin, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicate
		}
		return nil, err
	}
	return &u, nil
}

// List returns all users (without password hashes in the JSON response).
func (s *Store) List(ctx context.Context) ([]User, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, username, is_admin, password_hash, created_at
		FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.IsAdmin, &u.PasswordHash, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// BootstrapAdmin promotes the seeded default user (id) to an admin account with
// the given credentials. It sets the username and password only when the user
// currently has no password (first run), so an operator-changed password is
// never clobbered on restart. Returns true when a bootstrap was applied.
func (s *Store) BootstrapAdmin(ctx context.Context, id, username, password string) (bool, error) {
	if password == "" {
		return false, nil // bootstrap disabled
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return false, err
	}
	tag, err := s.db.Pool.Exec(ctx, `
		UPDATE users
		SET username = $2, password_hash = $3, is_admin = TRUE
		WHERE id = $1 AND password_hash = ''`,
		id, strings.TrimSpace(username), hash)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) scanOne(ctx context.Context, sql string, args ...any) (*User, error) {
	var u User
	err := s.db.Pool.QueryRow(ctx, sql, args...).
		Scan(&u.ID, &u.Username, &u.IsAdmin, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// isUniqueViolation reports whether err is a Postgres unique-constraint error
// (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
