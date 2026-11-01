// Package store 封装全部 SQL 访问。
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"online-store/server/internal/models"
)

// 常见存储错误，handler 层据此映射 HTTP 状态码。
var (
	ErrNotFound   = errors.New("record not found")
	ErrEmailTaken = errors.New("email already registered")
	ErrEmptyCart  = errors.New("cart is empty")
)

// UserStore 负责用户表。
type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore { return &UserStore{db: db} }

// Create 插入新用户；邮箱已存在时返回 ErrEmailTaken。
func (s *UserStore) Create(ctx context.Context, username, email, passwordHash string) (*models.User, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO users (username, email, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		username, email, passwordHash, now,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("插入用户: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取用户ID: %w", err)
	}
	return &models.User{ID: id, Username: username, Email: email, CreatedAt: now}, nil
}

func (s *UserStore) GetByID(ctx context.Context, id int64) (*models.User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, username, email, created_at FROM users WHERE id = ?`, id))
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*models.User, string, error) {
	var user models.User
	var passwordHash string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, created_at, password_hash FROM users WHERE email = ?`, email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("查询用户: %w", err)
	}
	return &user, passwordHash, nil
}

func (s *UserStore) scanUser(row *sql.Row) (*models.User, error) {
	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户: %w", err)
	}
	return &user, nil
}

// isUniqueViolation 判断是否为 SQLite 唯一约束冲突。
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToUpper(err.Error()), "UNIQUE CONSTRAINT FAILED")
}
