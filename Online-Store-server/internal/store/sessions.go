package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"online-store/server/internal/models"
)

// SessionStore 负责会话表。数据库只保存 token 的 SHA-256 哈希。
type SessionStore struct {
	db *sql.DB
}

func NewSessionStore(db *sql.DB) *SessionStore { return &SessionStore{db: db} }

// Create 写入一条会话记录。
func (s *SessionStore) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		tokenHash, userID, expiresAt.UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("写入会话: %w", err)
	}
	return nil
}

// Validate 校验 token 并返回所属用户（实时读取最新 role，角色调整立即生效）；
// 过期或不存在返回 ErrNotFound。
func (s *SessionStore) Validate(ctx context.Context, tokenHash string) (*models.User, error) {
	var user models.User
	var expiresAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.email, u.role, u.created_at, se.expires_at
		FROM sessions se JOIN users u ON u.id = se.user_id
		WHERE se.token_hash = ?
	`, tokenHash).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询会话: %w", err)
	}

	expires, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(expires) {
		_ = s.Delete(ctx, tokenHash)
		return nil, ErrNotFound
	}
	return &user, nil
}

// Delete 删除指定会话。
func (s *SessionStore) Delete(ctx context.Context, tokenHash string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash); err != nil {
		return fmt.Errorf("删除会话: %w", err)
	}
	return nil
}

// DeleteByUser 删除某用户的全部会话。密码重置走 UserStore.ResetPassword 的事务版本，
// 这里留给需要单独注销用户的场景。
func (s *SessionStore) DeleteByUser(ctx context.Context, userID int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("删除用户会话: %w", err)
	}
	return nil
}

// DeleteExpired 清理过期会话，服务启动时调用一次，之后由后台任务每小时调用。
func (s *SessionStore) DeleteExpired(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at <= ?`, time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("清理过期会话: %w", err)
	}
	return nil
}
