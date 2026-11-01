package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"online-store/server/internal/models"
)

// ProductFilter 是后台商品列表的过滤条件。
type ProductFilter struct {
	Query    string // 匹配 id / title（LIKE）
	Category string // 为空表示全部分类
	Active   string // "all"（默认）/ "true" / "false"
	Limit    int
	Offset   int
}

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

// List 返回上架商品列表，category 为空时返回全部。商城侧只展示 active=1。
func (s *ProductStore) List(ctx context.Context, category string) ([]models.Product, error) {
	query := `SELECT id, category_slug, title, description, price_cents, unit, image,
	                 detail, origin, storage, nutrients, care, color, rating, type, active, sort_order
	          FROM products WHERE active = 1`
	args := []any{}
	if category != "" {
		query += ` AND category_slug = ?`
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

// GetByID 按 ID 查询单个上架商品；下架商品视为不存在。
func (s *ProductStore) GetByID(ctx context.Context, id string) (*models.Product, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, category_slug, title, description, price_cents, unit, image,
		       detail, origin, storage, nutrients, care, color, rating, type, active, sort_order
		FROM products WHERE id = ? AND active = 1`, id)
	product, err := scanProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return product, err
}

// AdminList 返回后台商品列表（含下架商品），支持搜索/分类/状态过滤与分页。
func (s *ProductStore) AdminList(ctx context.Context, filter ProductFilter) ([]models.Product, int64, error) {
	where := " WHERE 1=1"
	args := []any{}
	if filter.Query != "" {
		where += ` AND (id LIKE ? ESCAPE '\' OR title LIKE ? ESCAPE '\')`
		like := "%" + escapeLike(filter.Query) + "%"
		args = append(args, like, like)
	}
	if filter.Category != "" {
		where += ` AND category_slug = ?`
		args = append(args, filter.Category)
	}
	switch filter.Active {
	case "true":
		where += ` AND active = 1`
	case "false":
		where += ` AND active = 0`
	}

	var total int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM products`+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计商品数量: %w", err)
	}

	query := `SELECT id, category_slug, title, description, price_cents, unit, image,
	                 detail, origin, storage, nutrients, care, color, rating, type, active, sort_order
	          FROM products` + where + ` ORDER BY sort_order, id LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询商品: %w", err)
	}
	defer rows.Close()

	products := make([]models.Product, 0)
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, 0, err
		}
		products = append(products, *product)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("查询商品: %w", err)
	}
	return products, total, nil
}

// AdminGetByID 按 ID 查询商品（不过滤上下架），供后台编辑使用。
func (s *ProductStore) AdminGetByID(ctx context.Context, id string) (*models.Product, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, category_slug, title, description, price_cents, unit, image,
		       detail, origin, storage, nutrients, care, color, rating, type, active, sort_order
		FROM products WHERE id = ?`, id)
	product, err := scanProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return product, err
}

// Create 新增商品：ID 为空时生成 prd-<16位十六进制>；sortOrder 为 0 时追加到末尾。
func (s *ProductStore) Create(ctx context.Context, product *models.Product) error {
	if product.ID == "" {
		id, err := generateProductID()
		if err != nil {
			return err
		}
		product.ID = id
	}
	if product.SortOrder == 0 {
		if err := s.db.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM products`,
		).Scan(&product.SortOrder); err != nil {
			return fmt.Errorf("查询商品排序: %w", err)
		}
	}

	active := 0
	if product.Active {
		active = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO products (
			id, category_slug, title, description, price_cents, unit, image,
			detail, origin, storage, nutrients, care, color, rating, type, active, sort_order
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, product.ID, product.Category, product.Title, product.Description, product.PriceCents,
		product.Unit, product.Image, product.Detail, product.Origin, product.Storage,
		product.Nutrients, product.Care, product.Color, product.Rating, product.Type,
		active, product.SortOrder)
	if err != nil {
		return fmt.Errorf("新增商品: %w", err)
	}
	return nil
}

// Update 更新商品全部可编辑字段，然后按 ID 重新查询返回最新数据。
// （先更新再查询，避免同值更新导致 RowsAffected=0 被误判为 404。）
func (s *ProductStore) Update(ctx context.Context, product *models.Product) (*models.Product, error) {
	active := 0
	if product.Active {
		active = 1
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE products SET
			category_slug = ?, title = ?, description = ?, price_cents = ?, unit = ?,
			image = ?, detail = ?, origin = ?, storage = ?, nutrients = ?, care = ?,
			color = ?, rating = ?, type = ?, active = ?, sort_order = ?
		WHERE id = ?
	`, product.Category, product.Title, product.Description, product.PriceCents, product.Unit,
		product.Image, product.Detail, product.Origin, product.Storage, product.Nutrients,
		product.Care, product.Color, product.Rating, product.Type, active, product.SortOrder,
		product.ID)
	if err != nil {
		return nil, fmt.Errorf("更新商品: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, ErrNotFound
	}
	return s.AdminGetByID(ctx, product.ID)
}

// Delete 删除商品。cart_items 依赖外键级联删除；order_items 无外键，历史快照保留。
func (s *ProductStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM products WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除商品: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	return nil
}

// CategoryExists 判断分类是否存在，用于后台商品创建/编辑校验。
func (s *ProductStore) CategoryExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM categories WHERE slug = ?)`, slug,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("查询分类: %w", err)
	}
	return exists, nil
}

// escapeLike 转义 LIKE 通配符，配合 ESCAPE '\' 使用，保证按字面量匹配。
func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

// generateProductID 生成形如 prd-a1b2c3d4e5f60718 的商品 ID（8 字节随机）。
func generateProductID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成商品ID: %w", err)
	}
	return "prd-" + hex.EncodeToString(buf), nil
}

type rowScanner interface{ Scan(dest ...any) error }

func scanProduct(row rowScanner) (*models.Product, error) {
	var product models.Product
	var active int64
	err := row.Scan(&product.ID, &product.Category, &product.Title, &product.Description,
		&product.PriceCents, &product.Unit, &product.Image, &product.Detail, &product.Origin,
		&product.Storage, &product.Nutrients, &product.Care, &product.Color,
		&product.Rating, &product.Type, &active, &product.SortOrder)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("读取商品: %w", err)
	}
	product.Active = active == 1
	product.PriceText = models.FormatPrice(product.PriceCents, product.Unit)
	return &product, nil
}
