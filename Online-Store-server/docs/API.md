# 绿源农业后端接口文档

Base URL：`http://localhost:8080`

所有业务接口都以 `/api` 为前缀；`/` 及其余路径由服务端托管的前端静态页面使用。除静态资源外，请求与响应均为 JSON（`Content-Type: application/json; charset=utf-8`）。

- 通用约定、错误格式见 [通用说明](#通用说明)；
- 逐个接口的参数、示例与错误码见 [接口列表](#接口列表)；
- 典型调用流程见 [调用示例](#调用示例)。

## 目录

- [通用说明](#通用说明)
  - [认证方式](#认证方式)
  - [金额与数量](#金额与数量)
  - [错误响应格式](#错误响应格式)
  - [状态码总览](#状态码总览)
  - [安全限制](#安全限制)
- [接口列表](#接口列表)
  - [健康检查](#健康检查)
  - [分类与商品](#分类与商品)
  - [账号认证](#账号认证)
  - [购物车](#购物车)
  - [订单](#订单)
  - [后台管理](#后台管理)
- [调用示例](#调用示例)

---

## 通用说明

### 认证方式

采用 **Cookie 会话**：调用 `POST /api/auth/register` 或 `POST /api/auth/login` 成功后，服务端通过 `Set-Cookie` 下发会话凭证，浏览器后续请求自动携带。

| 项目 | 值 |
| --- | --- |
| Cookie 名 | `store_session` |
| 属性 | `HttpOnly`、`SameSite=Lax`、`Path=/`；商城用户有效期 7 天（`SESSION_TTL_HOURS` 可配），管理员有效期 12 小时（`ADMIN_SESSION_TTL_HOURS` 可配）；生产环境 HTTPS 下加 `Secure` |
| 数据库存储 | 仅存 token 的 SHA-256 哈希 |
| 需要认证的接口 | `/api/auth/me`、全部 `/api/cart/*`、全部 `/api/orders*` |
| 管理员接口 | 全部 `/api/admin/*` 要求会话所属用户 `role = admin`，否则返回 `403 not_admin` |

用户信息中包含 `role` 字段（`user` / `admin`）。会话校验实时读取最新角色：管理员被降级后，旧会话下一次访问后台即返回 403，无需重新登录。

需要认证的接口未登录或会话过期时返回 `401`。前端使用 `fetch` 时请保证 `credentials: 'same-origin'`（`js/api.js` 已统一处理）。

### 金额与数量

- 所有金额使用整数 **分**（`priceCents` / `subtotalCents` / `totalCents`），避免浮点误差；同时返回拼好的展示文本 `priceText`（如 `¥26.8/500g`）。
- 购物车与订单价格一律由服务端按商品表计算，客户端提交的价格字段会被忽略。
- 购物车数量范围 **1–99**。
- 订单商品行保存下单时的标题、单价、单位、图片快照，后续改价不影响历史订单。

### 错误响应格式

所有业务错误返回统一结构：

```json
{
  "error": {
    "code": "unauthorized",
    "message": "请先登录"
  }
}
```

注意：`/api` 下所有未匹配到的请求——包括对已知路径使用了错误的 HTTP 方法（如对 `GET /api/health` 发 `DELETE`）——都会进入兜底处理，返回 `404` + 同样的 JSON 结构（`code: "not_found"`），而不是 Go 默认的 405。

### 状态码总览

| 状态码 | 含义 | 常见触发场景 |
| --- | --- | --- |
| 200 | 成功 | 查询、修改、删除类操作 |
| 201 | 创建成功 | 注册、下单 |
| 400 | 参数不合法 | 字段缺失、格式错误、非法数量、空购物车下单、`self_operation_forbidden`（修改自己的角色 / 删除自己） |
| 401 | 未登录或凭证失效 | 未携带会话 Cookie、会话过期、密码错误 |
| 403 | 拒绝访问 | 浏览器跨站写请求（Origin/Referer 校验失败）；非管理员访问 `/api/admin/*`（`not_admin`） |
| 404 | 资源不存在 | 商品/购物车条目/订单不存在、未知 API 路径或方法 |
| 409 | 冲突 | 注册时邮箱已被使用；`last_admin`（试图移除最后一名管理员） |
| 429 | 请求过于频繁 | 登录/注册触发 IP 限流（商城与管理员登录各自独立计数） |
| 500 | 服务器内部错误 | 数据库异常等 |

### 安全限制

| 限制 | 规则 |
| --- | --- |
| 请求体大小 | 最大 1MB，超出返回 400 |
| 未知字段 | 拒绝（`DisallowUnknownFields`），返回 400 |
| 登录/注册限流 | 同一 IP 15 分钟内最多 10 次，超出返回 429 |
| 管理员登录限流 | 与商城登录独立计数，同一 IP 15 分钟内最多 10 次 |
| 后台响应缓存 | `/api/admin/*` 与后台页面统一返回 `Cache-Control: no-store` |
| 跨站写保护 | `POST/PATCH/DELETE` 请求若携带 `Origin` 或 `Referer` 头，必须与当前服务同源（或 `ALLOWED_ORIGIN` 配置的来源），否则 403；两者都不携带（如 curl）则放行 |
| 输入校验 | 见各接口的“校验规则” |

---

## 接口列表

### 健康检查

#### `GET /api/health`

无参数、无需认证。

**响应 200**

```json
{ "status": "ok" }
```

---

### 分类与商品

#### `GET /api/categories`

返回全部分类，按 `sort_order` 排序。无需认证。

**响应 200**

```json
{
  "categories": [
    { "slug": "vegetable", "title": "田园鲜蔬·自然之选" },
    { "slug": "fruit", "title": "四季鲜果·阳光之味" },
    { "slug": "flowers", "title": "花漾时光·生活美学" }
  ]
}
```

#### `GET /api/products`

商品列表，无需认证。

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `category` | string | 否 | 分类 slug（`vegetable` / `fruit` / `flowers`），缺省返回全部 |

**响应 200**

```json
{
  "products": [
    {
      "id": "veg-1",
      "category": "vegetable",
      "title": "番茄",
      "description": "每颗番茄都经过严格筛选",
      "priceCents": 2680,
      "priceText": "¥26.8/500g",
      "unit": "500g",
      "image": "images/Products/tomato.jpg",
      "detail": "我们的有机番茄种植在阳光充足的山谷中……",
      "origin": "山东寿光",
      "storage": "冷藏保存7天",
      "nutrients": "维生素C、番茄红素、钾",
      "care": "",
      "color": "",
      "rating": 4.8,
      "type": "蔬菜",
      "active": true,
      "sortOrder": 1
    }
  ]
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `priceCents` | 单价，整数分 |
| `active` | 是否上架；公开列表与详情只返回 `active=1` 的商品，下架商品返回 404 |
| `priceText` | 展示文本，由 `priceCents` + `unit` 生成 |
| `image` | 图片路径，相对前端站点根目录 |
| `storage` / `nutrients` | 蔬果类商品使用 |
| `care` / `color` | 花卉类商品使用（此时 `storage` / `nutrients` 为空字符串） |
| `type` | `蔬菜` / `水果` / `花卉`，前端据此切换详情弹窗字段 |

#### `GET /api/products/{id}`

商品详情，无需认证。

**响应 200**：`{ "product": { … } }`，字段同上。

**错误**

| 状态码 | code | 场景 |
| --- | --- | --- |
| 404 | `product_not_found` | 商品ID不存在 |

---

### 账号认证

#### `POST /api/auth/register`

注册新用户并**直接登录**（响应会携带 `Set-Cookie`）。

**请求体**

| 字段 | 类型 | 必填 | 校验规则 |
| --- | --- | --- | --- |
| `username` | string | 是 | 去除首尾空格后 2–30 个字符 |
| `email` | string | 是 | 合法邮箱格式（入库前转为小写） |
| `password` | string | 是 | 8–72 字节 |

**请求示例**

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"李雷","email":"lilei@example.com","password":"password123"}' \
  -c cookies.txt
```

**响应 201**

```json
{
  "user": {
    "id": 1,
    "username": "李雷",
    "email": "lilei@example.com",
    "createdAt": "2026-10-31T15:25:50Z"
  }
}
```

**错误**

| 状态码 | code | message |
| --- | --- | --- |
| 400 | `invalid_username` | 用户名需要 2-30 个字符 |
| 400 | `invalid_email` | 邮箱格式不正确 |
| 400 | `invalid_password` | 密码长度需要 8-72 个字符 |
| 409 | `email_taken` | 该邮箱已注册，请直接登录 |
| 429 | `rate_limited` | 尝试过于频繁，请稍后再试 |

#### `POST /api/auth/login`

邮箱 + 密码登录。

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `email` | string | 是 | 大小写不敏感，自动转小写 |
| `password` | string | 是 | 明文密码（HTTPS 下传输） |

**响应 200**：同注册的 `user` 结构，并携带 `Set-Cookie`。

**错误**

| 状态码 | code | message |
| --- | --- | --- |
| 400 | `invalid_request` | 请输入邮箱和密码 |
| 401 | `invalid_credentials` | 邮箱或密码不正确（邮箱不存在与密码错误返回同一提示，避免枚举） |
| 429 | `rate_limited` | 尝试过于频繁，请稍后再试 |

#### `POST /api/auth/logout`

删除当前会话并清除 Cookie。无需请求体；未登录调用也返回 200。

**响应 200**

```json
{ "ok": true }
```

#### `GET /api/auth/me`

查询当前登录用户。**需要认证**。

**响应 200**

```json
{
  "user": {
    "id": 1,
    "username": "李雷",
    "email": "lilei@example.com",
    "createdAt": "2026-10-31T15:25:50Z"
  }
}
```

**错误**：`401` + `code: "unauthorized"`（未登录 / 会话过期）。

---

### 购物车

以下接口均**需要认证**。所有成功响应都返回操作后的完整购物车，前端无需再发查询请求。

购物车数据结构：

```json
{
  "cart": {
    "items": [
      {
        "productId": "veg-1",
        "title": "番茄",
        "priceCents": 2680,
        "priceText": "¥26.8/500g",
        "unit": "500g",
        "image": "images/Products/tomato.jpg",
        "quantity": 2,
        "lineTotalCents": 5360
      }
    ],
    "subtotalCents": 5360,
    "totalCents": 5360
  }
}
```

> 第一版不含运费与优惠，`totalCents` 恒等于 `subtotalCents`。

#### `GET /api/cart`

查询当前用户购物车。

**响应 200**：上述结构；空购物车时 `items` 为 `[]`、金额为 `0`。

#### `POST /api/cart/items`

添加商品；若已存在则数量累加。

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `productId` | string | 是 | 商品ID |
| `quantity` | int | 否 | 默认 `1`；服务端夹取到 1–99 |

**请求示例**

```bash
curl -X POST http://localhost:8080/api/cart/items \
  -H 'Content-Type: application/json' -b cookies.txt \
  -d '{"productId":"veg-1","quantity":2}'
```

**响应 200**：操作后的购物车。

**错误**

| 状态码 | code | 场景 |
| --- | --- | --- |
| 400 | `invalid_product` | 缺少 `productId` |
| 404 | `product_not_found` | 商品ID不存在 |

#### `PATCH /api/cart/items/{productId}`

把指定商品数量设置为给定值（覆盖而非累加）。

**请求体**

| 字段 | 类型 | 必填 | 校验 |
| --- | --- | --- | --- |
| `quantity` | int | 是 | 1–99 |

**响应 200**：操作后的购物车。

**错误**

| 状态码 | code | 场景 |
| --- | --- | --- |
| 400 | `invalid_quantity` | 数量不在 1–99 之间 |
| 404 | `cart_item_not_found` | 购物车中没有该商品 |

#### `DELETE /api/cart/items/{productId}`

删除购物车中的指定商品。

**响应 200**：操作后的购物车。

**错误**：`404` + `cart_item_not_found`（条目不存在）。

#### `POST /api/cart/merge`

合并游客（localStorage）购物车到服务端，用于登录/注册成功后同步。幂等可重复调用。

**请求体**

```json
{
  "items": [
    { "productId": "fruit-1", "quantity": 2 }
  ]
}
```

**合并规则**

- 服务端不存在的商品ID直接跳过，不报错；
- 与服务端已有条目按数量累加，结果夹取到 1–99；
- 合并后服务端为准，客户端应清空本地购物车。

**响应 200**：合并后的购物车。

---

### 订单

以下接口均**需要认证**。

订单数据结构：

```json
{
  "order": {
    "id": "ORD-20261031-8bcdeb",
    "totalCents": 5360,
    "status": "pending",
    "shippingName": "李雷",
    "shippingPhone": "13800138000",
    "shippingAddress": "上海市静安区南京西路 100 号",
    "createdAt": "2026-10-31T15:32:00Z",
    "items": [
      {
        "productId": "veg-1",
        "title": "番茄",
        "unitPriceCents": 2680,
        "unitPriceText": "¥26.8/500g",
        "unit": "500g",
        "image": "images/Products/tomato.jpg",
        "quantity": 2
      }
    ]
  }
}
```

- 订单号格式 `ORD-YYYYMMDD-6位随机十六进制`；
- `status` 取值：`pending`（待发货）/ `paid`（已支付）/ `shipped`（已发货）/ `done`（已完成）/ `cancelled`（已取消）；新订单固定为 `pending`，管理员可在后台修改（不强制状态机，允许人工纠错）；
- `items` 为下单时的商品快照，商品删除后仍保留。

#### `POST /api/orders`

用**当前服务端购物车**创建订单（事务：写订单与明细 → 清空购物车）。

**请求体**

| 字段 | 类型 | 必填 | 校验规则 |
| --- | --- | --- | --- |
| `shippingName` | string | 是 | 去除首尾空格后 2–30 个字符 |
| `shippingPhone` | string | 是 | 6–20 位，仅允许数字、`+`、`-`、空格 |
| `shippingAddress` | string | 是 | 去除首尾空格后 5–200 个字符 |

**请求示例**

```bash
curl -X POST http://localhost:8080/api/orders \
  -H 'Content-Type: application/json' -b cookies.txt \
  -d '{"shippingName":"李雷","shippingPhone":"13800138000","shippingAddress":"上海市静安区南京西路 100 号"}'
```

**响应 201**：上述 `order` 结构。

**错误**

| 状态码 | code | 场景 |
| --- | --- | --- |
| 400 | `cart_empty` | 购物车为空 |
| 400 | `invalid_shipping_name` | 收货人姓名不合法 |
| 400 | `invalid_shipping_phone` | 联系电话不合法 |
| 400 | `invalid_shipping_address` | 收货地址不合法 |

#### `GET /api/orders`

当前用户的订单列表，按下单时间倒序。

**响应 200**

```json
{ "orders": [ { … } ] }
```

无订单时 `orders` 为 `[]`。

#### `GET /api/orders/{id}`

订单详情。只能查看自己的订单。

**响应 200**：`{ "order": { … } }`

**错误**：`404` + `order_not_found`（不存在或不属于当前用户）。

---

## 后台管理

管理后台页面：`/admin-login.html`（登录）与 `/admin`（管理台，302 跳转到 `/admin.html`）。

- 全部 `/api/admin/*` 接口要求管理员会话；未登录返回 `401 unauthorized`，非管理员返回 `403 not_admin`。
- 响应统一携带 `Cache-Control: no-store`。
- 列表接口分页参数：`page`（默认 1）、`pageSize`（默认 20，最大 100），非法值自动夹取；响应包含 `total` / `page` / `pageSize`。
- 管理员登录、商品/用户/订单的写操作会记录结构化日志（管理员 ID/邮箱、动作、目标 ID），不记录密码与完整收货地址。

### 后台认证

#### `POST /api/admin/auth/login`

管理员登录。密码正确且 `role = admin` 才签发会话（管理员 TTL）；普通用户返回 403 且不创建会话（不影响浏览器中已有的客户会话）。

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `email` | string | 是 | 大小写不敏感 |
| `password` | string | 是 | 明文密码 |

**响应 200**：`{ "user": { "id":1, "username":"管理员", "email":"...", "role":"admin", "createdAt":"..." } }` + `Set-Cookie`。

**错误**

| 状态码 | code | 场景 |
| --- | --- | --- |
| 400 | `invalid_request` | 缺少邮箱或密码 |
| 401 | `invalid_credentials` | 邮箱或密码错误（含邮箱不存在，等价时间校验） |
| 403 | `not_admin` | 密码正确但账号不是管理员 |
| 429 | `rate_limited` | 触发管理员登录限流 |

#### `GET /api/admin/auth/me`

查询当前管理员。**需要管理员会话**。

**响应 200**：`{ "user": { … } }`；`401` / `403` 同上。

### 概览

#### `GET /api/admin/stats`

**响应 200**

```json
{
  "stats": {
    "userCount": 8,
    "adminCount": 1,
    "productCount": 10,
    "activeProductCount": 9,
    "orderCount": 5,
    "pendingOrderCount": 2,
    "todayOrderCount": 1,
    "revenueCents": 32600
  },
  "recentOrders": [
    {
      "id": "ORD-20261101-abcdef123456",
      "userId": 2,
      "username": "小明",
      "userEmail": "ming@example.com",
      "totalCents": 11600,
      "status": "pending",
      "shippingName": "小明",
      "shippingPhone": "13800138000",
      "shippingAddress": "上海市静安区南京西路 100 号",
      "createdAt": "2026-11-01T10:00:00Z",
      "itemCount": 2
    }
  ]
}
```

- `recentOrders` 固定取最近 5 条，不含商品明细；
- 销售额 = 非 `cancelled` 订单的 `total_cents` 之和；
- “今日订单”按服务端本地时区当天 0 点划分。

### 商品管理

#### `GET /api/admin/products`

查询参数：`query`（匹配 ID / 标题）、`category`（slug）、`active`（`all` 默认 / `true` / `false`）、`page`、`pageSize`。

**响应 200**

```json
{ "products": [ { …完整商品，含 active/sortOrder… } ], "total": 10, "page": 1, "pageSize": 20 }
```

#### `POST /api/admin/products`

**请求示例**

```json
{
  "title": "有机生菜",
  "category": "vegetable",
  "priceCents": 1280,
  "unit": "500g",
  "image": "images/Products/tomato.jpg",
  "description": "清晨采摘",
  "detail": "……",
  "origin": "上海崇明",
  "storage": "冷藏保存 3 天",
  "nutrients": "维生素、膳食纤维",
  "care": "",
  "color": "",
  "rating": 4.8,
  "type": "蔬菜",
  "active": true,
  "sortOrder": 11
}
```

校验规则：

| 字段 | 规则 |
| --- | --- |
| `title` | 必填，去空格后 1–80 字符 |
| `category` | 必填，必须已存在（`invalid_category`） |
| `priceCents` | 必填，整数，0–100000000（¥0–¥1,000,000） |
| `unit` | 必填，1–20 字符 |
| `image` | 必填，1–300 字符；站内相对路径；禁止 `://`、`..`、开头 `/`、空白/控制字符/引号；扩展名限 jpg/jpeg/png/webp/avif/svg |
| `description` | 可选，≤200 字符 |
| `detail` | 可选，≤2000 字符 |
| `origin` / `storage` / `nutrients` / `care` / `color` | 可选，各 ≤200 字符 |
| `rating` | 可选，0–5 |
| `type` | 必填，`蔬菜` / `水果` / `花卉` |
| `active` | 可选，默认 `true` |
| `sortOrder` | 可选，≥0；不传则追加到末尾 |

**响应 201**：`{ "product": { … } }`，服务端生成 ID（`prd-` + 16 位十六进制）。

**错误**：400 + `invalid_title` / `invalid_category` / `invalid_priceCents` / `invalid_unit` / `invalid_image` / `invalid_type` / `invalid_rating` / `invalid_sortOrder` 等。

#### `GET /api/admin/products/{id}`

后台商品详情（含下架商品）。**响应 200**：`{ "product": { … } }`；404 `product_not_found`。

#### `PATCH /api/admin/products/{id}`

编辑商品：请求体为任意字段子集（同创建），**只更新出现的字段**，未出现字段保持原值（历史数据不会阻断上下架等小改动）。

**响应 200**：`{ "product": { … } }`；404 `product_not_found`；400 同创建。

#### `DELETE /api/admin/products/{id}`

删除商品：`cart_items` 中的对应条目级联删除；`order_items` 无外键，历史订单快照保留。

**响应 200**：`{ "ok": true }`；404 `product_not_found`。

### 用户管理

#### `GET /api/admin/users`

查询参数：`query`（匹配用户名 / 邮箱）、`role`（`all` 默认 / `user` / `admin`）、`page`、`pageSize`。

**响应 200**

```json
{
  "users": [
    {
      "id": 2,
      "username": "小明",
      "email": "ming@example.com",
      "role": "user",
      "createdAt": "2026-10-31T15:25:50Z",
      "orderCount": 2,
      "totalSpentCents": 11600,
      "lastOrderAt": "2026-11-01T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 20
}
```

- `orderCount` 包含已取消订单；`totalSpentCents` 不含已取消订单；无订单时 `lastOrderAt` 为空字符串。

#### `PATCH /api/admin/users/{id}`

请求：`{ "role": "admin" }` 或 `{ "role": "user" }`。角色变更实时生效，无需强制下线。

**响应 200**：`{ "user": { … } }`。

**错误**

| 状态码 | code | 场景 |
| --- | --- | --- |
| 400 | `invalid_user_id` | 路径参数不是正整数 |
| 400 | `invalid_role` | 角色不是 user / admin |
| 400 | `self_operation_forbidden` | 修改自己的角色（唯一管理员降级自己走 `last_admin`） |
| 404 | `user_not_found` | 用户不存在 |
| 409 | `last_admin` | 系统只剩一名管理员时尝试降级 |

#### `POST /api/admin/users/{id}/password`

请求：`{ "password": "new-password" }`（8–72 字节）。在事务中更新密码并删除该用户全部会话；允许重置自己的密码（会话随即失效）。

**响应 200**：`{ "ok": true }`。错误：400 `invalid_password` / `invalid_user_id`；404 `user_not_found`。

#### `DELETE /api/admin/users/{id}`

删除用户：外键级联删除其会话、购物车、订单与订单明细。

**响应 200**：`{ "ok": true }`。错误：400 `invalid_user_id` / `self_operation_forbidden`（删自己；唯一管理员删自己走 `last_admin`）；404 `user_not_found`；409 `last_admin`。

### 订单管理

#### `GET /api/admin/orders`

查询参数：`status`（`all` 默认 / 五种状态）、`query`（订单号 / 邮箱 / 用户名 / 收货人 / 电话）、`userId`（正整数，按买家过滤）、`page`、`pageSize`。按 `createdAt DESC` 排序。

**响应 200**

```json
{
  "orders": [
    {
      "id": "ORD-20261101-abcdef123456",
      "userId": 2,
      "username": "小明",
      "userEmail": "ming@example.com",
      "totalCents": 11600,
      "status": "pending",
      "shippingName": "小明",
      "shippingPhone": "13800138000",
      "shippingAddress": "上海市静安区南京西路 100 号",
      "createdAt": "2026-11-01T10:00:00Z",
      "itemCount": 2
    }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 20
}
```

错误：400 `invalid_user_id`（`userId` 不是正整数）。

#### `GET /api/admin/orders/{id}`

订单详情（不校验归属，含商品明细）。**响应 200**：`{ "order": { …同列表结构 + items… } }`；404 `order_not_found`。

#### `PATCH /api/admin/orders/{id}`

请求：`{ "status": "shipped" }`，允许 `pending/paid/shipped/done/cancelled` 五种状态自由切换（不强制状态机，便于人工纠错）。

**响应 200**：`{ "order": { …含 items… } }`。错误：400 `invalid_status`；404 `order_not_found`。

#### `DELETE /api/admin/orders/{id}`

**响应 200**：`{ "ok": true }`；订单明细级联删除。404 `order_not_found`。

---

## 调用示例

一次完整购物流程（curl）：

```bash
# 1. 浏览商品（游客）
curl -s http://localhost:8080/api/products?category=fruit

# 2. 注册（成功即登录，Cookie 存入 cookies.txt）
curl -s -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"李雷","email":"lilei@example.com","password":"password123"}' \
  -c cookies.txt

# 3. 合并游客购物车（本地购物车数据）
curl -s -X POST http://localhost:8080/api/cart/merge \
  -H 'Content-Type: application/json' -b cookies.txt \
  -d '{"items":[{"productId":"veg-1","quantity":1}]}'

# 4. 再加一件并改数量
curl -s -X POST http://localhost:8080/api/cart/items \
  -H 'Content-Type: application/json' -b cookies.txt \
  -d '{"productId":"fruit-1"}'
curl -s -X PATCH http://localhost:8080/api/cart/items/veg-1 \
  -H 'Content-Type: application/json' -b cookies.txt \
  -d '{"quantity":2}'

# 5. 下单
curl -s -X POST http://localhost:8080/api/orders \
  -H 'Content-Type: application/json' -b cookies.txt \
  -d '{"shippingName":"李雷","shippingPhone":"13800138000","shippingAddress":"上海市静安区南京西路 100 号"}'

# 6. 查看订单列表
curl -s http://localhost:8080/api/orders -b cookies.txt

# 7. 退出登录
curl -s -X POST http://localhost:8080/api/auth/logout -b cookies.txt
```

管理员操作流程（curl）：

```bash
# 1. 管理员登录（Cookie 存入 admin.txt）
curl -s -X POST http://localhost:8080/api/admin/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@greenfarm.local","password":"admin12345"}' \
  -c admin.txt

# 2. 查看概览统计与最近订单
curl -s http://localhost:8080/api/admin/stats -b admin.txt

# 3. 新增商品（商城列表立即可见）
curl -s -X POST http://localhost:8080/api/admin/products \
  -H 'Content-Type: application/json' -b admin.txt \
  -d '{"title":"有机生菜","category":"vegetable","priceCents":1280,"unit":"500g","image":"images/Products/tomato.jpg","type":"蔬菜"}'

# 4. 下架商品（商城列表/详情/购物车立即不可见）
curl -s -X PATCH http://localhost:8080/api/admin/products/prd-a1b2c3d4e5f60718 \
  -H 'Content-Type: application/json' -b admin.txt \
  -d '{"active":false}'

# 5. 修改订单状态（客户账户页可见）
curl -s -X PATCH http://localhost:8080/api/admin/orders/ORD-20261101-abcdef123456 \
  -H 'Content-Type: application/json' -b admin.txt \
  -d '{"status":"shipped"}'

# 6. 非管理员访问后台接口 → 403 not_admin
curl -s http://localhost:8080/api/admin/stats -b cookies.txt
```

前端对应实现见 `Online-Store-main/js/api.js`（请求封装）与各页面脚本；启动与配置说明见 [README.md](../README.md)。
