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
- [调用示例](#调用示例)

---

## 通用说明

### 认证方式

采用 **Cookie 会话**：调用 `POST /api/auth/register` 或 `POST /api/auth/login` 成功后，服务端通过 `Set-Cookie` 下发会话凭证，浏览器后续请求自动携带。

| 项目 | 值 |
| --- | --- |
| Cookie 名 | `store_session` |
| 属性 | `HttpOnly`、`SameSite=Lax`、`Path=/`、有效期 7 天（`SESSION_TTL_HOURS` 可配）；生产环境 HTTPS 下加 `Secure` |
| 数据库存储 | 仅存 token 的 SHA-256 哈希 |
| 需要认证的接口 | `/api/auth/me`、全部 `/api/cart/*`、全部 `/api/orders*` |

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
| 400 | 参数不合法 | 字段缺失、格式错误、非法数量、空购物车下单 |
| 401 | 未登录或凭证失效 | 未携带会话 Cookie、会话过期、密码错误 |
| 403 | 拒绝访问 | 浏览器跨站写请求（Origin/Referer 校验失败） |
| 404 | 资源不存在 | 商品/购物车条目/订单不存在、未知 API 路径或方法 |
| 409 | 冲突 | 注册时邮箱已被使用 |
| 429 | 请求过于频繁 | 登录/注册触发 IP 限流 |
| 500 | 服务器内部错误 | 数据库异常等 |

### 安全限制

| 限制 | 规则 |
| --- | --- |
| 请求体大小 | 最大 1MB，超出返回 400 |
| 未知字段 | 拒绝（`DisallowUnknownFields`），返回 400 |
| 登录/注册限流 | 同一 IP 15 分钟内最多 10 次，超出返回 429 |
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
      "type": "蔬菜"
    }
  ]
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `priceCents` | 单价，整数分 |
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
- `status` 第一版固定为 `pending`（未接入支付/发货流程）；
- `items` 为下单时的商品快照。

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

前端对应实现见 `Online-Store-main/js/api.js`（请求封装）与各页面脚本；启动与配置说明见 [README.md](../README.md)。
