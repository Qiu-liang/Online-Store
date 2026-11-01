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

// UserFilter 是后台用户列表的过滤条件。
type UserFilter struct {
	Query  string // 匹配 username / email（LIKE）
	Role   string // "all"（默认）/ "user" / "admin"
	Limit  int
	Offset int
}

// UserStore 负责用户表。
type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore { return &UserStore{db: db} }

// Create 插入普通用户（role='user'）；邮箱已存在时返回 ErrEmailTaken。
func (s *UserStore) Create(ctx context.Context, username, email, passwordHash string) (*models.User, error) {
	return s.insert(ctx, username, email, passwordHash, models.RoleUser)
}

// CreateAdmin 插入管理员账号（role='admin'）。
func (s *UserStore) CreateAdmin(ctx context.Context, username, email, passwordHash string) (*models.User, error) {
	return s.insert(ctx, username, email, passwordHash, models.RoleAdmin)
}

func (s *UserStore) insert(ctx context.Context, username, email, passwordHash, role string) (*models.User, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO users (username, email, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)`,
		username, email, passwordHash, role, now,
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
	return &models.User{ID: id, Username: username, Email: email, Role: role, CreatedAt: now}, nil
}

func (s *UserStore) GetByID(ctx context.Context, id int64) (*models.User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, username, email, role, created_at FROM users WHERE id = ?`, id))
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*models.User, string, error) {
	var user models.User
	var passwordHash string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, role, created_at, password_hash FROM users WHERE email = ?`, email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt, &passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("查询用户: %w", err)
	}
	return &user, passwordHash, nil
}

// ListAdmin 返回后台用户列表：搜索/角色过滤/分页，附订单统计。
// orderCount 含已取消订单；totalSpentCents 不含已取消订单。
func (s *UserStore) ListAdmin(ctx context.Context, filter UserFilter) ([]models.AdminUser, int64, error) {
	where := " WHERE 1=1"
	args := []any{}
	if filter.Query != "" {
		where += ` AND (username LIKE ? ESCAPE '\' OR email LIKE ? ESCAPE '\')`
		like := "%" + escapeLike(filter.Query) + "%"
		args = append(args, like, like)
	}
	switch filter.Role {
	case models.RoleUser, models.RoleAdmin:
		where += ` AND role = ?`
		args = append(args, filter.Role)
	}

	var total int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users`+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计用户数量: %w", err)
	}

	query := `
		SELECT u.id, u.username, u.email, u.role, u.created_at,
		       COALESCE(o.order_count, 0),
		       COALESCE(o.total_spent, 0),
		       COALESCE(o.last_order_at, '')
		FROM users u
		LEFT JOIN (
			SELECT user_id,
			       COUNT(*) AS order_count,
			       SUM(CASE WHEN status != '` + models.OrderStatusCancelled + `' THEN total_cents ELSE 0 END) AS total_spent,
			       MAX(created_at) AS last_order_at
			FROM orders GROUP BY user_id
		) o ON o.user_id = u.id` + where + `
		ORDER BY u.id LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询用户列表: %w", err)
	}
	defer rows.Close()

	users := make([]models.AdminUser, 0)
	for rows.Next() {
		var item models.AdminUser
		if err := rows.Scan(&item.ID, &item.Username, &item.Email, &item.Role, &item.CreatedAt,
			&item.OrderCount, &item.TotalSpentCents, &item.LastOrderAt); err != nil {
			return nil, 0, fmt.Errorf("读取用户列表: %w", err)
		}
		users = append(users, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("读取用户列表: %w", err)
	}
	return users, total, nil
}

// UpdateRole 调整用户角色并返回最新用户信息。
func (s *UserStore) UpdateRole(ctx context.Context, id int64, role string) (*models.User, error) {
	result, err := s.db.ExecContext(ctx,
		`UPDATE users SET role = ? WHERE id = ?`, role, id)
	if err != nil {
		return nil, fmt.Errorf("更新用户角色: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, ErrNotFound
	}
	return s.GetByID(ctx, id)
}

// ResetPassword 在一个事务里更新密码并删除该用户全部会话，
// 保证“密码已改、会话已清”原子生效。
func (s *UserStore) ResetPassword(ctx context.Context, id int64, passwordHash string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启重置密码事务: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		`UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, id)
	if err != nil {
		return fmt.Errorf("更新密码: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, id); err != nil {
		return fmt.Errorf("删除用户会话: %w", err)
	}
	return tx.Commit()
}

// Delete 删除用户。外键级联删除该用户的会话、购物车、订单与订单明细。
func (s *UserStore) Delete(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除用户: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	return nil
}

// CountAdmins 统计管理员数量，用于“最后一名管理员”保护。
func (s *UserStore) CountAdmins(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE role = ?`, models.RoleAdmin,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计管理员数量: %w", err)
	}
	return count, nil
}

func (s *UserStore) scanUser(row *sql.Row) (*models.User, error) {
	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt)
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
