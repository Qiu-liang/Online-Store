package store

import (
	"context"
	"database/sql"
	"fmt"

	"online-store/server/internal/models"
)

const (
	minQuantity = 1
	maxQuantity = 99
)

// CartStore 负责服务端购物车（按用户维度）。
type CartStore struct {
	db *sql.DB
}

func NewCartStore(db *sql.DB) *CartStore { return &CartStore{db: db} }

// GetCart 返回用户购物车（含金额汇总）。下架商品被静默隐藏，重新上架后恢复可见。
func (s *CartStore) GetCart(ctx context.Context, userID int64) (*models.Cart, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.title, p.price_cents, p.unit, p.image, ci.quantity
		FROM cart_items ci JOIN products p ON p.id = ci.product_id
		WHERE ci.user_id = ? AND p.active = 1
		ORDER BY ci.rowid
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询购物车: %w", err)
	}
	defer rows.Close()

	cart := &models.Cart{Items: []models.CartItem{}}
	for rows.Next() {
		var item models.CartItem
		if err := rows.Scan(&item.ProductID, &item.Title, &item.PriceCents, &item.Unit,
			&item.Image, &item.Quantity); err != nil {
			return nil, fmt.Errorf("读取购物车: %w", err)
		}
		item.PriceText = models.FormatPrice(item.PriceCents, item.Unit)
		item.LineTotalCents = item.PriceCents * item.Quantity
		cart.Items = append(cart.Items, item)
		cart.SubtotalCents += item.LineTotalCents
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取购物车: %w", err)
	}
	cart.TotalCents = cart.SubtotalCents
	return cart, nil
}

// AddItem 添加商品；数量与已有条目累加并限制在 1–99。
func (s *CartStore) AddItem(ctx context.Context, userID int64, productID string, quantity int64) (*models.Cart, error) {
	if quantity < minQuantity {
		quantity = minQuantity
	}
	if quantity > maxQuantity {
		quantity = maxQuantity
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO cart_items (user_id, product_id, quantity)
		SELECT ?, ?, ? WHERE EXISTS (SELECT 1 FROM products WHERE id = ? AND active = 1)
		ON CONFLICT(user_id, product_id)
		DO UPDATE SET quantity = MIN(cart_items.quantity + excluded.quantity, ?)
	`, userID, productID, quantity, productID, maxQuantity)
	if err != nil {
		return nil, fmt.Errorf("写入购物车: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, ErrNotFound
	}
	return s.GetCart(ctx, userID)
}

// UpdateQuantity 设置数量，条目不存在返回 ErrNotFound。
func (s *CartStore) UpdateQuantity(ctx context.Context, userID int64, productID string, quantity int64) (*models.Cart, error) {
	if quantity < minQuantity || quantity > maxQuantity {
		return nil, ErrNotFound
	}
	result, err := s.db.ExecContext(ctx,
		`UPDATE cart_items SET quantity = ? WHERE user_id = ? AND product_id = ?`,
		quantity, userID, productID,
	)
	if err != nil {
		return nil, fmt.Errorf("更新购物车: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, ErrNotFound
	}
	return s.GetCart(ctx, userID)
}

// RemoveItem 删除购物车条目，条目不存在返回 ErrNotFound。
func (s *CartStore) RemoveItem(ctx context.Context, userID int64, productID string) (*models.Cart, error) {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM cart_items WHERE user_id = ? AND product_id = ?`,
		userID, productID,
	)
	if err != nil {
		return nil, fmt.Errorf("删除购物车条目: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, ErrNotFound
	}
	return s.GetCart(ctx, userID)
}

// MergeItems 合并游客购物车：不存在的商品ID直接跳过，数量累加。整批放在一个事务里。
func (s *CartStore) MergeItems(ctx context.Context, userID int64, items []models.MergeItem) (*models.Cart, error) {
	if len(items) > 0 {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("开启合并事务: %w", err)
		}
		defer tx.Rollback()

		for _, item := range items {
			quantity := item.Quantity
			if quantity < minQuantity {
				quantity = minQuantity
			}
			if quantity > maxQuantity {
				quantity = maxQuantity
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO cart_items (user_id, product_id, quantity)
				SELECT ?, ?, ? WHERE EXISTS (SELECT 1 FROM products WHERE id = ? AND active = 1)
				ON CONFLICT(user_id, product_id)
				DO UPDATE SET quantity = MIN(cart_items.quantity + excluded.quantity, ?)
			`, userID, item.ProductID, quantity, item.ProductID, maxQuantity); err != nil {
				return nil, fmt.Errorf("合并购物车: %w", err)
			}
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("提交合并事务: %w", err)
		}
	}
	return s.GetCart(ctx, userID)
}

// Clear 清空用户购物车。
func (s *CartStore) Clear(ctx context.Context, userID int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM cart_items WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("清空购物车: %w", err)
	}
	return nil
}
