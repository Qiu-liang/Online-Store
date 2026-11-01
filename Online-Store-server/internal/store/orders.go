package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"online-store/server/internal/models"
)

// OrderStore 负责订单与订单明细。
type OrderStore struct {
	db *sql.DB
}

func NewOrderStore(db *sql.DB) *OrderStore { return &OrderStore{db: db} }

// CreateFromCart 在事务中把当前购物车转成订单：写入订单与快照明细并清空购物车。
func (s *OrderStore) CreateFromCart(ctx context.Context, userID int64, shipping models.Shipping) (*models.Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("开启事务: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT p.id, p.title, p.price_cents, p.unit, p.image, ci.quantity
		FROM cart_items ci JOIN products p ON p.id = ci.product_id
		WHERE ci.user_id = ? AND p.active = 1
		ORDER BY ci.rowid
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("读取购物车: %w", err)
	}

	items := make([]models.OrderItem, 0)
	var total int64
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Title, &item.UnitPriceCents,
			&item.Unit, &item.Image, &item.Quantity); err != nil {
			rows.Close()
			return nil, fmt.Errorf("读取购物车条目: %w", err)
		}
		item.UnitPriceText = models.FormatPrice(item.UnitPriceCents, item.Unit)
		items = append(items, item)
		total += item.UnitPriceCents * item.Quantity
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("读取购物车条目: %w", err)
	}
	rows.Close()

	if len(items) == 0 {
		return nil, ErrEmptyCart
	}

	orderID, err := generateOrderID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO orders (id, user_id, total_cents, status, shipping_name, shipping_phone, shipping_address, created_at)
		VALUES (?, ?, ?, 'pending', ?, ?, ?, ?)
	`, orderID, userID, total, shipping.Name, shipping.Phone, shipping.Address, now); err != nil {
		return nil, fmt.Errorf("写入订单: %w", err)
	}
	for _, item := range items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO order_items (order_id, product_id, title, unit_price_cents, unit, image, quantity)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, orderID, item.ProductID, item.Title, item.UnitPriceCents, item.Unit, item.Image, item.Quantity); err != nil {
			return nil, fmt.Errorf("写入订单明细: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM cart_items WHERE user_id = ?`, userID); err != nil {
		return nil, fmt.Errorf("清空购物车: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交订单事务: %w", err)
	}

	return &models.Order{
		ID:              orderID,
		UserID:          userID,
		TotalCents:      total,
		Status:          "pending",
		ShippingName:    shipping.Name,
		ShippingPhone:   shipping.Phone,
		ShippingAddress: shipping.Address,
		CreatedAt:       now,
		Items:           items,
	}, nil
}

// ListByUser 返回用户全部订单（含明细），按下单时间倒序。
func (s *OrderStore) ListByUser(ctx context.Context, userID int64) ([]models.Order, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, total_cents, status, shipping_name, shipping_phone, shipping_address, created_at
		FROM orders WHERE user_id = ? ORDER BY created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询订单: %w", err)
	}
	defer rows.Close()

	orders := make([]models.Order, 0)
	for rows.Next() {
		var order models.Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.TotalCents, &order.Status,
			&order.ShippingName, &order.ShippingPhone, &order.ShippingAddress, &order.CreatedAt); err != nil {
			return nil, fmt.Errorf("读取订单: %w", err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取订单: %w", err)
	}

	// 一次性取出全部订单明细，避免逐单查询（N+1）。
	orderIDs := make([]string, len(orders))
	for index := range orders {
		orderIDs[index] = orders[index].ID
	}
	itemsByOrder, err := s.listItemsForOrders(ctx, orderIDs)
	if err != nil {
		return nil, err
	}
	for index := range orders {
		orders[index].Items = itemsByOrder[orders[index].ID]
	}
	return orders, nil
}

// listItemsForOrders 一条 IN 查询取回多笔订单的明细，按 order_id 分组返回。
func (s *OrderStore) listItemsForOrders(ctx context.Context, orderIDs []string) (map[string][]models.OrderItem, error) {
	grouped := make(map[string][]models.OrderItem, len(orderIDs))
	if len(orderIDs) == 0 {
		return grouped, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(orderIDs)), ",")
	args := make([]any, len(orderIDs))
	for i, id := range orderIDs {
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT order_id, product_id, title, unit_price_cents, unit, image, quantity
		FROM order_items WHERE order_id IN (`+placeholders+`) ORDER BY rowid
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("批量查询订单明细: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var orderID string
		var item models.OrderItem
		if err := rows.Scan(&orderID, &item.ProductID, &item.Title, &item.UnitPriceCents,
			&item.Unit, &item.Image, &item.Quantity); err != nil {
			return nil, fmt.Errorf("读取订单明细: %w", err)
		}
		item.UnitPriceText = models.FormatPrice(item.UnitPriceCents, item.Unit)
		grouped[orderID] = append(grouped[orderID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取订单明细: %w", err)
	}
	return grouped, nil
}

// GetByID 返回指定订单，且校验归属。
func (s *OrderStore) GetByID(ctx context.Context, userID int64, orderID string) (*models.Order, error) {
	var order models.Order
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, total_cents, status, shipping_name, shipping_phone, shipping_address, created_at
		FROM orders WHERE id = ? AND user_id = ?
	`, orderID, userID).Scan(&order.ID, &order.UserID, &order.TotalCents, &order.Status,
		&order.ShippingName, &order.ShippingPhone, &order.ShippingAddress, &order.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询订单: %w", err)
	}
	items, err := s.listItems(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	order.Items = items
	return &order, nil
}

func (s *OrderStore) listItems(ctx context.Context, orderID string) ([]models.OrderItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT product_id, title, unit_price_cents, unit, image, quantity
		FROM order_items WHERE order_id = ? ORDER BY rowid
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("查询订单明细: %w", err)
	}
	defer rows.Close()

	items := make([]models.OrderItem, 0)
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Title, &item.UnitPriceCents,
			&item.Unit, &item.Image, &item.Quantity); err != nil {
			return nil, fmt.Errorf("读取订单明细: %w", err)
		}
		item.UnitPriceText = models.FormatPrice(item.UnitPriceCents, item.Unit)
		items = append(items, item)
	}
	return items, rows.Err()
}

// AdminOrderFilter 是后台订单列表的过滤条件。
type AdminOrderFilter struct {
	Status string // "all"（默认）/ 五种状态之一
	Query  string // 匹配订单号/邮箱/用户名/收货人/电话（LIKE）
	UserID int64  // >0 时按买家过滤
	Limit  int
	Offset int
}

// AdminList 返回后台订单列表（含买家信息与商品件数），支持状态/搜索/用户过滤与分页。
func (s *OrderStore) AdminList(ctx context.Context, filter AdminOrderFilter) ([]models.AdminOrder, int64, error) {
	where := " WHERE 1=1"
	args := []any{}
	if filter.Status != "" && filter.Status != "all" {
		where += ` AND o.status = ?`
		args = append(args, filter.Status)
	}
	if filter.Query != "" {
		where += ` AND (o.id LIKE ? ESCAPE '\' OR u.email LIKE ? ESCAPE '\' OR u.username LIKE ? ESCAPE '\'
			OR o.shipping_name LIKE ? ESCAPE '\' OR o.shipping_phone LIKE ? ESCAPE '\')`
		like := "%" + escapeLike(filter.Query) + "%"
		args = append(args, like, like, like, like, like)
	}
	if filter.UserID > 0 {
		where += ` AND o.user_id = ?`
		args = append(args, filter.UserID)
	}

	var total int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM orders o JOIN users u ON u.id = o.user_id`+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计订单数量: %w", err)
	}

	query := `
		SELECT o.id, o.user_id, u.username, u.email, o.total_cents, o.status,
		       o.shipping_name, o.shipping_phone, o.shipping_address, o.created_at,
		       (SELECT COUNT(*) FROM order_items oi WHERE oi.order_id = o.id) AS item_count
		FROM orders o JOIN users u ON u.id = o.user_id` + where + `
		ORDER BY o.created_at DESC, o.id DESC LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询订单列表: %w", err)
	}
	defer rows.Close()

	orders := make([]models.AdminOrder, 0)
	for rows.Next() {
		var order models.AdminOrder
		if err := rows.Scan(&order.ID, &order.UserID, &order.Username, &order.UserEmail,
			&order.TotalCents, &order.Status, &order.ShippingName, &order.ShippingPhone,
			&order.ShippingAddress, &order.CreatedAt, &order.ItemCount); err != nil {
			return nil, 0, fmt.Errorf("读取订单列表: %w", err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("读取订单列表: %w", err)
	}
	return orders, total, nil
}

// AdminGetByID 返回后台视角的订单详情（不校验归属，含明细）。
func (s *OrderStore) AdminGetByID(ctx context.Context, orderID string) (*models.AdminOrder, error) {
	var order models.AdminOrder
	err := s.db.QueryRowContext(ctx, `
		SELECT o.id, o.user_id, u.username, u.email, o.total_cents, o.status,
		       o.shipping_name, o.shipping_phone, o.shipping_address, o.created_at,
		       (SELECT COUNT(*) FROM order_items oi WHERE oi.order_id = o.id) AS item_count
		FROM orders o JOIN users u ON u.id = o.user_id
		WHERE o.id = ?
	`, orderID).Scan(&order.ID, &order.UserID, &order.Username, &order.UserEmail,
		&order.TotalCents, &order.Status, &order.ShippingName, &order.ShippingPhone,
		&order.ShippingAddress, &order.CreatedAt, &order.ItemCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询订单: %w", err)
	}
	items, err := s.listItems(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	order.Items = items
	return &order, nil
}

// UpdateStatus 修改订单状态，返回最新订单数据；订单不存在返回 ErrNotFound。
func (s *OrderStore) UpdateStatus(ctx context.Context, orderID, status string) (*models.AdminOrder, error) {
	result, err := s.db.ExecContext(ctx,
		`UPDATE orders SET status = ? WHERE id = ?`, status, orderID)
	if err != nil {
		return nil, fmt.Errorf("更新订单状态: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, ErrNotFound
	}
	return s.AdminGetByID(ctx, orderID)
}

// Delete 删除订单；order_items 依赖外键级联删除。
func (s *OrderStore) Delete(ctx context.Context, orderID string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM orders WHERE id = ?`, orderID)
	if err != nil {
		return fmt.Errorf("删除订单: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	return nil
}

// generateOrderID 生成形如 ORD-20261031-a1b2c3d4e5f6 的订单号（6 字节随机，碰撞概率可忽略）。
func generateOrderID() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成订单号: %w", err)
	}
	return fmt.Sprintf("ORD-%s-%s", time.Now().Format("20060102"), hex.EncodeToString(buf)), nil
}
