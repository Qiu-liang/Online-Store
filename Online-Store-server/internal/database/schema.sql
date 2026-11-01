CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL,
    email         TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,
    role          TEXT    NOT NULL DEFAULT 'user',
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
    active        INTEGER NOT NULL DEFAULT 1,
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

-- 后台用键值表：目前仅用于 products_seeded 标记，保证商品种子只在首次启动写入。
CREATE TABLE IF NOT EXISTS app_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_user     ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expiry   ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category_slug);
CREATE INDEX IF NOT EXISTS idx_orders_user       ON orders(user_id);
-- 注意：涉及新增列（role/active）的索引不能写在这里——旧库此时还没有这些列，
-- 它们由 database.go 的 ensureIndexes 在补列之后创建。
