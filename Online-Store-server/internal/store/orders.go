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
		WHERE ci.user_id = ?
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

// generateOrderID 生成形如 ORD-20261031-a1b2c3d4e5f6 的订单号（6 字节随机，碰撞概率可忽略）。
func generateOrderID() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成订单号: %w", err)
	}
	return fmt.Sprintf("ORD-%s-%s", time.Now().Format("20060102"), hex.EncodeToString(buf)), nil
}
