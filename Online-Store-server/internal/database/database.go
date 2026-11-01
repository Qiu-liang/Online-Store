// Package database 负责打开 SQLite、执行建表与增量迁移、灌入种子数据。
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

// appMetaProductsSeeded 标记商品种子已写入；存在即跳过商品灌入，
// 保证后台对商品的编辑与删除不会被重启覆盖或复活。
const appMetaProductsSeeded = "products_seeded"

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

// migrate 执行建表脚本，再为旧库补齐新增列与新索引。
// 迁移只做增量（新增列/表/索引），不删除、不改写现有数据。
func migrate(db *sql.DB) error {
	schema, err := embedFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("读取建表脚本: %w", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("执行建表脚本: %w", err)
	}
	if err := ensureColumns(db); err != nil {
		return err
	}
	return ensureIndexes(db)
}

// ensureColumns 通过 PRAGMA table_info 检查并补齐旧库缺失的列；已存在则跳过。
func ensureColumns(db *sql.DB) error {
	additions := []struct {
		table  string
		column string
		ddl    string
	}{
		{"users", "role", `ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user'`},
		{"products", "active", `ALTER TABLE products ADD COLUMN active INTEGER NOT NULL DEFAULT 1`},
	}
	for _, item := range additions {
		exists, err := columnExists(db, item.table, item.column)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := db.Exec(item.ddl); err != nil {
			return fmt.Errorf("补齐列 %s.%s: %w", item.table, item.column, err)
		}
	}
	return nil
}

// columnExists 检查表中的某一列是否存在。
func columnExists(db *sql.DB, table, column string) (bool, error) {
	if column == "" {
		return false, fmt.Errorf("列名为空")
	}
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, fmt.Errorf("读取表结构 %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return false, fmt.Errorf("读取表结构 %s: %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// ensureIndexes 创建后台查询需要的索引。涉及新增列的索引不能放进 schema.sql
// （旧库在补列前执行会失败），统一放这里用 IF NOT EXISTS 保证幂等。
func ensureIndexes(db *sql.DB) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)`,
		`CREATE INDEX IF NOT EXISTS idx_products_active ON products(active)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_created ON orders(created_at)`,
	}
	for _, ddl := range indexes {
		if _, err := db.Exec(ddl); err != nil {
			return fmt.Errorf("创建索引: %w", err)
		}
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

// Seed 以幂等方式写入分类；商品仅在首次启动（app_meta 无 products_seeded 标记）时写入，
// 之后修改种子文件不再作用于已有商品，需要重置数据库才能重新灌入。
// 全部写入放在一个事务里，失败回滚，不留下半套数据。
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

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启种子事务: %w", err)
	}
	defer tx.Rollback()

	// 分类始终由种子维护（幂等 UPSERT）。
	for _, category := range data.Categories {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO categories (slug, title, sort_order) VALUES (?, ?, ?)
			ON CONFLICT(slug) DO UPDATE SET title = excluded.title, sort_order = excluded.sort_order
		`, category.Slug, category.Title, category.SortOrder); err != nil {
			return fmt.Errorf("写入分类 %s: %w", category.Slug, err)
		}
	}

	// 商品仅首次写入：事务内检查标记（单连接下事务内查询必须用 tx）。
	var marker string
	err = tx.QueryRowContext(ctx,
		`SELECT value FROM app_meta WHERE key = ?`, appMetaProductsSeeded).Scan(&marker)
	switch {
	case err == nil && marker == "1":
		// 已灌入过：跳过商品写入，保留后台编辑/删除结果。
		return tx.Commit()
	case err != nil && err != sql.ErrNoRows:
		return fmt.Errorf("读取种子标记: %w", err)
	}

	for index, product := range data.Products {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO products (
				id, category_slug, title, description, price_cents, unit, image,
				detail, origin, storage, nutrients, care, color, rating, type, active, sort_order
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?)
			ON CONFLICT(id) DO NOTHING
		`, product.ID, product.Category, product.Title, product.Description, product.PriceCents,
			product.Unit, product.Image, product.Detail, product.Origin, product.Storage,
			product.Nutrients, product.Care, product.Color, product.Rating, product.Type, index+1); err != nil {
			return fmt.Errorf("写入商品 %s: %w", product.ID, err)
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO app_meta (key, value) VALUES (?, '1')
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		appMetaProductsSeeded); err != nil {
		return fmt.Errorf("写入种子标记: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交种子事务: %w", err)
	}
	return nil
}
