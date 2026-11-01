// Package database 负责打开 SQLite、执行建表语句与灌入种子数据。
package database

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动，注册 "sqlite" 驱动名
	"online-store/server/internal/models"
)

//go:embed schema.sql seed_products.json
var embedFS embed.FS

// Open 打开（必要时创建）SQLite 数据库，执行迁移并灌入种子数据。
func Open(path string) (*sql.DB, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("解析数据库路径: %w", err)
	}
	if dir := filepath.Dir(absPath); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建数据目录: %w", err)
		}
	}

	dsn := "file:" + absPath + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库: %w", err)
	}
	// 单连接 + WAL，避免演示项目中的并发写锁问题。
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接数据库: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := Seed(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	schema, err := embedFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("读取建表脚本: %w", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("执行建表脚本: %w", err)
	}
	return nil
}

type seedFile struct {
	Categories []models.Category `json:"categories"`
	Products   []seedProduct     `json:"products"`
}

type seedProduct struct {
	ID          string  `json:"id"`
	Category    string  `json:"category"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	PriceCents  int64   `json:"priceCents"`
	Unit        string  `json:"unit"`
	Image       string  `json:"image"`
	Detail      string  `json:"detail"`
	Origin      string  `json:"origin"`
	Storage     string  `json:"storage"`
	Nutrients   string  `json:"nutrients"`
	Care        string  `json:"care"`
	Color       string  `json:"color"`
	Rating      float64 `json:"rating"`
	Type        string  `json:"type"`
}

// Seed 以幂等方式写入分类与商品数据；商品记录在启动时随种子文件刷新。
func Seed(db *sql.DB) error {
	raw, err := embedFS.ReadFile("seed_products.json")
	if err != nil {
		return fmt.Errorf("读取种子数据: %w", err)
	}

	var data seedFile
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("解析种子数据: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, category := range data.Categories {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO categories (slug, title, sort_order) VALUES (?, ?, ?)
			ON CONFLICT(slug) DO UPDATE SET title = excluded.title, sort_order = excluded.sort_order
		`, category.Slug, category.Title, category.SortOrder); err != nil {
			return fmt.Errorf("写入分类 %s: %w", category.Slug, err)
		}
	}

	for index, product := range data.Products {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO products (
				id, category_slug, title, description, price_cents, unit, image,
				detail, origin, storage, nutrients, care, color, rating, type, sort_order
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				category_slug = excluded.category_slug,
				title         = excluded.title,
				description   = excluded.description,
				price_cents   = excluded.price_cents,
				unit          = excluded.unit,
				image         = excluded.image,
				detail        = excluded.detail,
				origin        = excluded.origin,
				storage       = excluded.storage,
				nutrients     = excluded.nutrients,
				care          = excluded.care,
				color         = excluded.color,
				rating        = excluded.rating,
				type          = excluded.type,
				sort_order    = excluded.sort_order
		`, product.ID, product.Category, product.Title, product.Description, product.PriceCents,
			product.Unit, product.Image, product.Detail, product.Origin, product.Storage,
			product.Nutrients, product.Care, product.Color, product.Rating, product.Type, index+1); err != nil {
			return fmt.Errorf("写入商品 %s: %w", product.ID, err)
		}
	}
	return nil
}
