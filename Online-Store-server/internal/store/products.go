package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"online-store/server/internal/models"
)

// ProductStore 负责商品与分类查询。
type ProductStore struct {
	db *sql.DB
}

func NewProductStore(db *sql.DB) *ProductStore { return &ProductStore{db: db} }

// ListCategories 返回全部分类（按 sort_order 排序）。
func (s *ProductStore) ListCategories(ctx context.Context) ([]models.Category, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT slug, title, sort_order FROM categories ORDER BY sort_order`)
	if err != nil {
		return nil, fmt.Errorf("查询分类: %w", err)
	}
	defer rows.Close()

	categories := make([]models.Category, 0)
	for rows.Next() {
		var category models.Category
		if err := rows.Scan(&category.Slug, &category.Title, &category.SortOrder); err != nil {
			return nil, fmt.Errorf("读取分类: %w", err)
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

// List 返回商品列表，category 为空时返回全部。
func (s *ProductStore) List(ctx context.Context, category string) ([]models.Product, error) {
	query := `SELECT id, category_slug, title, description, price_cents, unit, image,
	                 detail, origin, storage, nutrients, care, color, rating, type
	          FROM products`
	args := []any{}
	if category != "" {
		query += ` WHERE category_slug = ?`
		args = append(args, category)
	}
	query += ` ORDER BY sort_order`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询商品: %w", err)
	}
	defer rows.Close()

	products := make([]models.Product, 0)
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, *product)
	}
	return products, rows.Err()
}

// GetByID 按ID查询单个商品。
func (s *ProductStore) GetByID(ctx context.Context, id string) (*models.Product, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, category_slug, title, description, price_cents, unit, image,
		       detail, origin, storage, nutrients, care, color, rating, type
		FROM products WHERE id = ?`, id)
	product, err := scanProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return product, err
}

type rowScanner interface{ Scan(dest ...any) error }

func scanProduct(row rowScanner) (*models.Product, error) {
	var product models.Product
	err := row.Scan(&product.ID, &product.Category, &product.Title, &product.Description,
		&product.PriceCents, &product.Unit, &product.Image, &product.Detail, &product.Origin,
		&product.Storage, &product.Nutrients, &product.Care, &product.Color,
		&product.Rating, &product.Type)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("读取商品: %w", err)
	}
	product.PriceText = models.FormatPrice(product.PriceCents, product.Unit)
	return &product, nil
}
