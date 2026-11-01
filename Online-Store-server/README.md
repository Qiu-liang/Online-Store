# Online-Store Server（绿源农业后端）

绿源农业静态商城的 Go 后端：提供商品、账号、购物车、订单 API 与管理后台 API（`/api/admin/*`），并同源托管前端静态页面（`../Online-Store-main`）。数据使用 SQLite（纯 Go 驱动，无 CGO），单文件即可运行。

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
- 管理后台：<http://localhost:8080/admin>
- 健康检查：<http://localhost:8080/api/health>

**初始管理员**：首次启动且 `ADMIN_EMAIL` 对应账号不存在时自动创建。`ADMIN_PASSWORD` 留空则生成 16 位随机密码并只在启动日志打印一次；账号已存在时不会修改密码。若该邮箱对应的是普通用户，仅记录告警，不自动提权。

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
| `SESSION_TTL_HOURS` | `168` | 商城用户会话有效期（小时） |
| `ADMIN_EMAIL` | `admin@greenfarm.local` | 初始管理员邮箱（仅首次启动创建） |
| `ADMIN_PASSWORD` | 空 | 初始管理员密码；留空则随机生成并打印一次日志 |
| `ADMIN_SESSION_TTL_HOURS` | `12` | 管理员会话有效期（小时） |
| `COOKIE_SECURE` | `false` | HTTPS 生产环境设为 `true` |
| `ALLOWED_ORIGIN` | 空 | 可选：允许跨域携带 Cookie 的前端来源 |
| `APP_ENV` | `development` | 运行环境标识 |

数据库文件和 WAL/SHM 伴随文件位于 `data/`（已在 `.gitignore` 中忽略）。删除 `data/store.db` 即可重置全部数据；下次启动会自动重建表结构并灌入种子数据。商品种子仅在首次启动写入（`app_meta.products_seeded` 标记），之后的编辑/删除由后台维护，重启不会覆盖或复活。

## 目录结构

```text
cmd/server/main.go          # 启动入口：配置、数据库、路由、优雅退出
internal/config/            # 环境变量与 .env 加载
internal/database/          # 打开数据库、schema.sql、种子数据（embed）
internal/auth/              # bcrypt 密码哈希、会话 token 与 Cookie
internal/models/            # API/存储共用数据结构与价格格式化
internal/store/             # SQL 访问层（users/sessions/products/cart/orders/stats）
internal/api/               # 路由、中间件、处理器、集成测试
data/                       # 运行时数据库文件
```

## 数据库表

- `users` 用户（bcrypt 密码哈希；`role` 区分 `user` / `admin`）
- `sessions` 会话（仅存 token 的 SHA-256，含过期时间）
- `categories` / `products` 分类与商品（分类随种子刷新；商品仅首次启动写入，`active` 标记上下架）
- `cart_items` 服务端购物车（按用户）
- `orders` / `order_items` 订单与商品快照（下单时的标题、单价、单位、图片）
- `app_meta` 内部键值标记（`products_seeded` 控制种子只写一次）

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
| POST | `/api/admin/auth/login` | 否 | 管理员登录（独立限流，签发短时会话） |
| GET | `/api/admin/auth/me` | 管理员 | 当前管理员 |
| GET | `/api/admin/stats` | 管理员 | 概览统计 + 最近 5 笔订单 |
| GET/POST | `/api/admin/products` | 管理员 | 商品列表（搜索/分类/状态/分页）与新增 |
| GET/PATCH/DELETE | `/api/admin/products/{id}` | 管理员 | 商品详情 / 编辑（部分更新）/ 删除 |
| GET | `/api/admin/users` | 管理员 | 用户列表（搜索/角色/分页 + 订单统计） |
| PATCH | `/api/admin/users/{id}` | 管理员 | 调整角色 `{role}` |
| POST | `/api/admin/users/{id}/password` | 管理员 | 重置密码（该用户全部会话失效） |
| DELETE | `/api/admin/users/{id}` | 管理员 | 删除用户（级联会话/购物车/订单） |
| GET | `/api/admin/orders` | 管理员 | 订单列表（状态/搜索/买家/分页） |
| GET/PATCH/DELETE | `/api/admin/orders/{id}` | 管理员 | 订单详情 / 修改状态 / 删除 |

错误响应统一为：

```json
{ "error": { "code": "unauthorized", "message": "请先登录" } }
```

常用状态码：400 参数错误（含 `self_operation_forbidden`）、401 未登录、403 跨站请求 / `not_admin`、404 资源不存在、409 邮箱已注册 / `last_admin`、429 触发限流。

## 安全设计

- 密码使用 bcrypt（cost 12），密码长度限制 8–72 字节；登录时对不存在的邮箱做等价时间校验，防止时序枚举。
- 会话 token 为 32 字节随机数，数据库仅存 SHA-256 哈希；Cookie `HttpOnly + SameSite=Lax`，`COOKIE_SECURE=true` 时附加 `Secure`；过期会话每小时自动清理。
- 所有写请求校验 `Origin`/`Referer` 同源（浏览器场景），缓解 CSRF。
- 登录/注册接口按 IP 限流（15 分钟 10 次）；管理员登录独立限流，避免与商城登录互相挤占。
- 后台接口统一要求 `role = admin`（实时读取，降级即刻生效）；后台页面与 `/api/admin/*` 响应携带 `Cache-Control: no-store`。
- 用户管理内置保护：不能修改/删除自己，不能移除最后一名管理员；重置密码在事务中同时删除该用户全部会话。
- 请求体限制 1MB，未知字段拒绝；金额、数量一律以后端校验与计算为准。
- 响应带 `Content-Security-Policy`（脚本/样式/字体限定 `self` 与 cdnjs，禁止内联脚本）、`X-Frame-Options: DENY`、`X-Content-Type-Options: nosniff`。
- 静态托管关闭目录列表：目录路径一律 404，仅暴露具体文件与根路径 `index.html`。

## 测试

```bash
go test ./...
```

集成测试使用临时 SQLite 覆盖：健康检查、商品目录、注册/登录/退出、购物车增删改、游客购物车合并、下单与订单查询、参数校验与错误码（`internal/api/api_test.go`）；后台认证与权限、商品/用户/订单管理、概览统计、上下架与下单联动（`internal/api/admin_test.go`）；旧库自动补列迁移与种子只写一次（`internal/database/database_test.go`）。

## 边界说明

- 未接入真实支付、库存扣减与邮件验证；订单状态为 `pending/paid/shipped/done/cancelled`，管理员可在五种状态间自由切换（不强制状态机）。
- 商品图片仍由前端 `images/` 静态目录提供，数据库只存路径；后台仅编辑相对路径，不做上传。
- 商品种子以 `internal/database/seed_products.json` 为初始来源，仅在首次启动写入；之后的商品维护通过后台进行，修改种子文件不影响已有数据。
