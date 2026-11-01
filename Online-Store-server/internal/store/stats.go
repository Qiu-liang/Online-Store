package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"online-store/server/internal/models"
)

// StatsStore 负责后台概览统计。
type StatsStore struct {
	db *sql.DB
}

func NewStatsStore(db *sql.DB) *StatsStore { return &StatsStore{db: db} }

// Overview 用单条 SQL 的标量子查询统计后台概览数据。
// 销售额不含已取消订单；“今日”按服务端本地时区的当天 0 点（转 UTC RFC3339）划分。
func (s *StatsStore) Overview(ctx context.Context) (*models.AdminStats, error) {
	now := time.Now()
	midnightLocal := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayStart := midnightLocal.UTC().Format(time.RFC3339)

	const query = `
		SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM users WHERE role = 'admin'),
			(SELECT COUNT(*) FROM products),
			(SELECT COUNT(*) FROM products WHERE active = 1),
			(SELECT COUNT(*) FROM orders),
			(SELECT COUNT(*) FROM orders WHERE status = 'pending'),
			(SELECT COUNT(*) FROM orders WHERE created_at >= ?),
			(SELECT COALESCE(SUM(total_cents), 0) FROM orders WHERE status != 'cancelled')`

	stats := &models.AdminStats{}
	err := s.db.QueryRowContext(ctx, query, todayStart).Scan(
		&stats.UserCount, &stats.AdminCount, &stats.ProductCount, &stats.ActiveProductCount,
		&stats.OrderCount, &stats.PendingOrderCount, &stats.TodayOrderCount, &stats.RevenueCents,
	)
	if err != nil {
		return nil, fmt.Errorf("统计概览数据: %w", err)
	}
	return stats, nil
}
