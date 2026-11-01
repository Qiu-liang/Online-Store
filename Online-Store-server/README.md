# Online-Store Server（绿源农业后端）

绿源农业静态商城的 Go 后端：提供商品、账号、购物车与订单 API，并同源托管前端静态页面（`../Online-Store-main`）。数据使用 SQLite（纯 Go 驱动，无 CGO），单文件即可运行。

## 环境要求

- Go 1.26 及以上（与 `go.mod` 声明一致；开发机使用 `brew install go` 安装）
- 无需 CGO、无需系统 SQLite 开发库

## 快速开始

```bash
cd Online-Store-server
go mod tidy        # 首次运行拉取依赖
go run ./cmd/server
```

启动后访问：

- 商城首页：<http://localhost:8080/>
- 健康检查：<http://localhost:8080/api/health>

常用命令：

```bash
make run     # go run ./cmd/server
make test    # go test ./...
make vet     # go vet ./...
make fmt     # gofmt 格式化
make build   # CGO_ENABLED=0 go build -o bin/server ./cmd/server
```

也可以先 `make build` 再运行 `./bin/server`；二进制启动时同样读取环境变量/`.env`。

## 配置

支持环境变量，也可把 `.env.example` 复制为 `.env`（已存在的环境变量优先）：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `8080` | HTTP 监听端口 |
| `FRONTEND_DIR` | `../Online-Store-main` | 前端静态目录 |
| `DB_PATH` | `./data/store.db` | SQLite 数据库文件 |
| `SESSION_TTL_HOURS` | `168` | 会话有效期（小时） |
| `COOKIE_SECURE` | `false` | HTTPS 生产环境设为 `true` |
| `ALLOWED_ORIGIN` | 空 | 可选：允许跨域携带 Cookie 的前端来源 |
| `APP_ENV` | `development` | 运行环境标识 |

数据库文件和 WAL/SHM 伴随文件位于 `data/`（已在 `.gitignore` 中忽略）。删除 `data/store.db` 即可重置全部数据；下次启动会自动重建表结构并灌入种子数据。

## 目录结构

```text
cmd/server/main.go          # 启动入口：配置、数据库、路由、优雅退出
internal/config/            # 环境变量与 .env 加载
internal/database/          # 打开数据库、schema.sql、种子数据（embed）
internal/auth/              # bcrypt 密码哈希、会话 token 与 Cookie
internal/models/            # API/存储共用数据结构与价格格式化
internal/store/             # SQL 访问层（users/sessions/products/cart/orders）
internal/api/               # 路由、中间件、处理器、集成测试
data/                       # 运行时数据库文件
```

## 数据库表

- `users` 用户（bcrypt 密码哈希）
- `sessions` 会话（仅存 token 的 SHA-256，含过期时间）
- `categories` / `products` 分类与商品（启动时随种子数据刷新）
- `cart_items` 服务端购物车（按用户）
- `orders` / `order_items` 订单与商品快照（下单时的标题、单价、单位、图片）

金额统一使用整数“分”（`price_cents`），接口返回 `priceCents` 与可直接展示的 `priceText`。

## API 一览

> 完整接口文档（字段说明、校验规则、错误码、curl 示例）见 [docs/API.md](docs/API.md)。

| 方法 | 路径 | 认证 | 说明 |
| --- | --- | --- | --- |
| GET | `/api/health` | 否 | 健康检查 |
| GET | `/api/categories` | 否 | 分类列表 |
| GET | `/api/products?category=fruit` | 否 | 商品列表，可按分类过滤 |
| GET | `/api/products/{id}` | 否 | 商品详情 |
| POST | `/api/auth/register` | 否 | 注册（成功即登录） |
| POST | `/api/auth/login` | 否 | 登录 |
| POST | `/api/auth/logout` | 否 | 退出并删除会话 |
| GET | `/api/auth/me` | 是 | 当前用户 |
| GET | `/api/cart` | 是 | 购物车 |
| POST | `/api/cart/items` | 是 | 添加商品 `{productId, quantity}` |
| PATCH | `/api/cart/items/{productId}` | 是 | 修改数量 `{quantity}`（1–99） |
| DELETE | `/api/cart/items/{productId}` | 是 | 删除商品 |
| POST | `/api/cart/merge` | 是 | 合并游客购物车 `{items:[{productId, quantity}]}` |
| POST | `/api/orders` | 是 | 下单 `{shippingName, shippingPhone, shippingAddress}` |
| GET | `/api/orders` | 是 | 当前用户订单列表 |
| GET | `/api/orders/{id}` | 是 | 订单详情 |

错误响应统一为：

```json
{ "error": { "code": "unauthorized", "message": "请先登录" } }
```

常用状态码：400 参数错误、401 未登录、403 跨站请求、404 资源不存在、409 邮箱已注册、429 触发限流。

## 安全设计

- 密码使用 bcrypt（cost 12），密码长度限制 8–72 字节；登录时对不存在的邮箱做等价时间校验，防止时序枚举。
- 会话 token 为 32 字节随机数，数据库仅存 SHA-256 哈希；Cookie `HttpOnly + SameSite=Lax`，`COOKIE_SECURE=true` 时附加 `Secure`；过期会话每小时自动清理。
- 所有写请求校验 `Origin`/`Referer` 同源（浏览器场景），缓解 CSRF。
- 登录/注册接口按 IP 限流（15 分钟 10 次）。
- 请求体限制 1MB，未知字段拒绝；金额、数量一律以后端校验与计算为准。
- 响应带 `Content-Security-Policy`（脚本/样式/字体限定 `self` 与 cdnjs，禁止内联脚本）、`X-Frame-Options: DENY`、`X-Content-Type-Options: nosniff`。
- 静态托管关闭目录列表：目录路径一律 404，仅暴露具体文件与根路径 `index.html`。

## 测试

```bash
go test ./...
```

集成测试（`internal/api/api_test.go`）使用临时 SQLite 覆盖：健康检查、商品目录、注册/登录/退出、购物车增删改、游客购物车合并、下单与订单查询、参数校验与错误码。

## 边界说明

- 未接入真实支付、库存扣减与邮件验证；订单状态固定为 `pending`。
- 商品图片仍由前端 `images/` 静态目录提供，数据库只存路径。
- 商品数据以 `internal/database/seed_products.json` 为单一来源，启动时刷新到 SQLite；修改种子文件后重启服务即可生效。
