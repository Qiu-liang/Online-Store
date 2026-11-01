package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

// oldSchema 是升级前的建表脚本（无 role / active / app_meta），用于验证旧库迁移。
const oldSchema = `
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL,
    email         TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,
    created_at    TEXT    NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
    token_hash TEXT    PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT    NOT NULL,
    created_at TEXT    NOT NULL
);
CREATE TABLE IF NOT EXISTS categories (
    slug       TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS products (
    id            TEXT    PRIMARY KEY,
    category_slug TEXT    NOT NULL REFERENCES categories(slug),
    title         TEXT    NOT NULL,
    description   TEXT    NOT NULL DEFAULT '',
    price_cents   INTEGER NOT NULL,
    unit          TEXT    NOT NULL DEFAULT '',
    image         TEXT    NOT NULL DEFAULT '',
    detail        TEXT    NOT NULL DEFAULT '',
    origin        TEXT    NOT NULL DEFAULT '',
    storage       TEXT    NOT NULL DEFAULT '',
    nutrients     TEXT    NOT NULL DEFAULT '',
    care          TEXT    NOT NULL DEFAULT '',
    color         TEXT    NOT NULL DEFAULT '',
    rating        REAL    NOT NULL DEFAULT 0,
    type          TEXT    NOT NULL DEFAULT '',
    sort_order    INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS cart_items (
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id TEXT    NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity   INTEGER NOT NULL,
    PRIMARY KEY (user_id, product_id)
);
CREATE TABLE IF NOT EXISTS orders (
    id               TEXT    PRIMARY KEY,
    user_id          INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_cents      INTEGER NOT NULL,
    status           TEXT    NOT NULL DEFAULT 'pending',
    shipping_name    TEXT    NOT NULL,
    shipping_phone   TEXT    NOT NULL,
    shipping_address TEXT    NOT NULL,
    created_at       TEXT    NOT NULL
);
CREATE TABLE IF NOT EXISTS order_items (
    order_id        TEXT    NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id      TEXT    NOT NULL,
    title           TEXT    NOT NULL,
    unit_price_cents INTEGER NOT NULL,
    unit            TEXT    NOT NULL DEFAULT '',
    image           TEXT    NOT NULL DEFAULT '',
    quantity        INTEGER NOT NULL,
    PRIMARY KEY (order_id, product_id)
);
`

// TestMigrateOldDatabase 旧库自动补列：老数据获得默认值，不丢失。
func TestMigrateOldDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("打开旧库: %v", err)
	}
	if _, err := db.Exec(oldSchema); err != nil {
		t.Fatalf("创建旧库结构: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO users (username, email, password_hash, created_at)
		VALUES ('老用户', 'old@example.com', 'hash', '2025-01-01T00:00:00Z')
	`); err != nil {
		t.Fatalf("写入旧用户: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO categories (slug, title, sort_order) VALUES ('vegetable', '田园鲜蔬', 1)
	`); err != nil {
		t.Fatalf("写入分类: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO products (id, category_slug, title, price_cents, unit, image, sort_order)
		VALUES ('veg-old', 'vegetable', '老番茄', 100, '500g', 'images/Products/tomato.jpg', 1)
	`); err != nil {
		t.Fatalf("写入旧商品: %v", err)
	}
	db.Close()

	// 用新版 Open 打开旧库：补列 + 默认值 + 种子标记
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("迁移旧库失败: %v", err)
	}
	defer reopened.Close()

	var role string
	if err := reopened.QueryRow(
		`SELECT role FROM users WHERE email = 'old@example.com'`,
	).Scan(&role); err != nil {
		t.Fatalf("users.role 列应存在: %v", err)
	}
	if role != "user" {
		t.Fatalf("旧用户角色应为默认 user, 实际: %q", role)
	}

	var active int64
	if err := reopened.QueryRow(
		`SELECT active FROM products WHERE id = 'veg-old'`,
	).Scan(&active); err != nil {
		t.Fatalf("products.active 列应存在: %v", err)
	}
	if active != 1 {
		t.Fatalf("旧商品 active 应为默认 1, 实际: %d", active)
	}

	// 旧库已有商品但无标记：首次升级只补标记，不覆盖已有商品
	var title string
	if err := reopened.QueryRow(
		`SELECT title FROM products WHERE id = 'veg-old'`,
	).Scan(&title); err != nil || title != "老番茄" {
		t.Fatalf("旧商品不应被种子覆盖: %q (%v)", title, err)
	}
	var marker string
	if err := reopened.QueryRow(
		`SELECT value FROM app_meta WHERE key = 'products_seeded'`,
	).Scan(&marker); err != nil || marker != "1" {
		t.Fatalf("迁移后应写入 products_seeded 标记: %v", err)
	}
}

// TestSeedOnlyOnce 种子只写一次：后台编辑与删除在重新 Seed 后保留。
func TestSeedOnlyOnce(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "seed-once.db"))
	if err != nil {
		t.Fatalf("初始化数据库: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if _, err := db.ExecContext(ctx,
		`UPDATE products SET title = '后台改名' WHERE id = 'veg-1'`,
	); err != nil {
		t.Fatalf("修改商品: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM products WHERE id = 'veg-2'`); err != nil {
		t.Fatalf("删除商品: %v", err)
	}

	// 模拟服务重启：再次执行种子
	if err := Seed(db); err != nil {
		t.Fatalf("重新执行种子: %v", err)
	}

	var title string
	if err := db.QueryRow(`SELECT title FROM products WHERE id = 'veg-1'`).Scan(&title); err != nil || title != "后台改名" {
		t.Fatalf("编辑应保留, 实际: %q (%v)", title, err)
	}
	var count int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM products WHERE id = 'veg-2'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("删除的商品不应复活, 实际: %d (%v)", count, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&count); err != nil || count != 9 {
		t.Fatalf("商品总数应保持 9, 实际: %d (%v)", count, err)
	}
}

// TestSeedFirstRun 全新数据库：灌入 10 个商品、3 个分类与标记。
func TestSeedFirstRun(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "fresh.db"))
	if err != nil {
		t.Fatalf("初始化数据库: %v", err)
	}
	defer db.Close()

	var products int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&products); err != nil || products != 10 {
		t.Fatalf("应写入 10 个种子商品, 实际: %d (%v)", products, err)
	}
	var categories int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&categories); err != nil || categories != 3 {
		t.Fatalf("应写入 3 个分类, 实际: %d (%v)", categories, err)
	}
	var marker string
	if err := db.QueryRow(
		`SELECT value FROM app_meta WHERE key = 'products_seeded'`,
	).Scan(&marker); err != nil || marker != "1" {
		t.Fatalf("应写入 products_seeded 标记: %v", err)
	}
}
