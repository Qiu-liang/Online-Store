# 绿源农业 · Online Store

一个中文农产品电商全栈项目：原生 HTML/CSS/JavaScript 前端 + Go 后端（SQLite）。展示蔬菜、水果、鲜花等农产品，支持商品浏览、购物车、登录注册与下单，订单可在个人账户页查看；并内置一套同源的管理后台（商品 / 用户 / 订单管理）。

## 项目结构

| 目录 | 说明 | 文档 |
| --- | --- | --- |
| `Online-Store-main/` | 前端静态站点（HTML/CSS/原生 JS，无框架、无构建步骤） | [README.md](Online-Store-main/README.md) |
| `Online-Store-server/` | Go 后端：REST API + 静态托管 + SQLite 存储 | [README.md](Online-Store-server/README.md) |

```text
Online-Store/
├── README.md                        # 项目总览（本文件）
├── LICENSE                          # MIT 许可证
├── Online-Store-main/               # 前端：6 个商城页面 + 后台 2 页 + css/ + js/ + images/
│   ├── index.html                   # 首页
│   ├── products.html                # 产品中心
│   ├── about.html                   # 关于我们
│   ├── shopping.html                # 购物车
│   ├── login.html                   # 登录 / 注册
│   ├── account.html                 # 我的账户
│   ├── admin.html                   # 管理后台（概览 / 商品 / 用户 / 订单）
│   └── admin-login.html             # 管理员登录
└── Online-Store-server/             # 后端：Go 模块 online-store/server
    ├── cmd/server/main.go           # 启动入口：配置 → 建库建表 → 初始管理员 → 路由 → 优雅退出
    ├── internal/
    │   ├── config/config.go         # 环境变量与 .env 加载（端口、前端目录、库路径、会话时长）
    │   ├── database/                # 打开库、建表与补列迁移、首次启动灌种子
    │   │   ├── database.go          # 连接初始化、执行 schema、幂等迁移、种子只写一次
    │   │   ├── schema.sql           # 表结构定义（go:embed 编译进二进制）
    │   │   └── seed_products.json   # 商品 / 分类种子（go:embed，仅首次启动写入）
    │   ├── auth/                    # 密码哈希与会话 Cookie
    │   │   ├── password.go          # bcrypt 哈希与校验（cost 12，含防时序枚举的哑校验）
    │   │   └── session.go           # 生成并哈希 token、读写 Cookie、清理过期会话
    │   ├── models/models.go         # API 与存储层共用结构体、金额（分）与展示文本转换
    │   ├── store/                   # SQL 访问层：按资源拆分，不感知 HTTP
    │   │   ├── users.go             # 用户查询 / 创建 / 改角色 / 重置密码 / 删除
    │   │   ├── sessions.go          # 会话读写与过期清理
    │   │   ├── products.go          # 分类与商品：列表、详情、后台搜索分页、上下架
    │   │   ├── cart.go              # 服务端购物车与游客购物车合并
    │   │   ├── orders.go            # 下单事务、订单快照、列表与状态变更
    │   │   └── stats.go             # 后台概览统计与最近订单
    │   └── api/                     # HTTP 层：路由、中间件、处理器、集成测试
    │       ├── router.go            # 注册全部商城与后台路由、/admin 跳转、静态托管
    │       ├── middleware.go        # 鉴权、管理员校验、限流、同源校验、安全头、日志、panic 恢复
    │       ├── response.go          # 统一 JSON 响应与错误结构
    │       ├── validate.go          # 商城侧请求解析与字段校验
    │       ├── admin_validate.go    # 后台请求校验（商品 / 用户 / 订单）
    │       ├── pagination.go        # 后台列表分页参数解析
    │       ├── *_handlers.go        # 处理器：auth、product、cart、order、admin_{auth,product,user,order}
    │       └── *_test.go            # 集成测试：api_test.go 商城、admin_test.go 后台权限
    ├── docs/API.md                  # 完整接口文档（字段、校验规则、错误码、curl 示例）
    ├── data/                        # 运行时 SQLite（store.db + WAL/SHM，已忽略，仅留 .gitkeep）
    ├── bin/                         # make build 产物（已忽略）
    ├── Makefile                     # run / test / vet / fmt / build / clean
    ├── .env.example                 # 配置样例，复制为 .env 后生效
    ├── go.mod / go.sum              # 依赖 modernc.org/sqlite、golang.org/x/crypto
    └── README.md                    # 后端详细文档（配置、表结构、安全设计、测试）
```

### 后端分层

代码按职责分为四层，依赖方向单一（`api → store → database`）；`config` / `auth` / `models` 作为横切支撑被各层复用。配置项、表结构与安全设计的完整说明见 [后端 README](Online-Store-server/README.md)，接口细节见 [API 文档](Online-Store-server/docs/API.md)。

| 层 | 职责 | 不做什么 |
| --- | --- | --- |
| `cmd/server` | 装配依赖、启动 HTTP 服务、监听信号并优雅关闭；首次启动创建管理员 | 不含业务逻辑 |
| `internal/api` | 解析请求、校验参数、鉴权与限流、组织 JSON 响应；同时托管前端静态文件 | 不直接写 SQL |
| `internal/store` | 封装全部 SQL 与事务（下单、合并购物车、重置密码等原子操作） | 不感知 HTTP |
| `internal/database` | 建库、建表与幂等迁移、嵌入并写入种子数据 | 不含查询逻辑 |

三条链路可作为阅读入口：

- **启动**：`main.go` → `config.Load` → `database.Open`（建表 + 迁移 + 种子）→ `bootstrapAdmin` → `api.NewServer` → `Handler()` 挂载路由 → 监听并等待退出信号。
- **一次业务请求**：`router.go` 匹配路由 → 中间件链（安全头 / 日志 / 同源校验 / `withAuth` 或 `withAdmin` / 限流）→ `*_handlers.go` 解析校验 → `store/*.go` 执行 SQL → `response.go` 输出统一 JSON。
- **静态页面**：未被 `/api/` 匹配的请求交给 `staticHandler()`，从 `FRONTEND_DIR`（默认 `../Online-Store-main`）读取文件；目录路径返回 404，HTML/CSS/JS 使用协商缓存，图片缓存一天。

## 功能特性

- **产品中心**：从 API 加载商品与分类，支持分类过滤、详情弹窗、评分星级、加入购物车。
- **购物车**：游客数据保存在 `localStorage`，登录后自动合并到服务端购物车；支持数量增减、删除、金额汇总。
- **登录注册**：bcrypt 密码哈希 + 服务端会话（Cookie），注册成功即登录。
- **订单**：购物车页填写收货信息即可下单，订单保存商品快照（标题、单价、单位、图片）。
- **管理后台**：管理员登录后可管理商品（新增 / 编辑 / 上下架 / 删除）、用户（搜索 / 角色 / 重置密码 / 删除）、订单（筛选 / 详情 / 状态修改 / 删除），概览页提供统计与最近订单。
- **安全**：同源写请求校验（防 CSRF）、登录限流（商城与管理员独立）、会话仅存哈希、CSP 等安全响应头、金额数量后端校验、后台接口角色校验。

## 快速开始

需要 Go 1.26+（macOS 可用 `brew install go`），无需 CGO 与系统 SQLite 开发库。

```bash
cd Online-Store-server
go mod tidy        # 首次运行拉取依赖
go run ./cmd/server
```

打开 <http://localhost:8080/> 即可浏览完整站点（后端同源托管前端静态文件，无需单独起前端服务）。

- 管理后台：<http://localhost:8080/admin>（首次启动会创建初始管理员；`ADMIN_EMAIL` / `ADMIN_PASSWORD` 可在 `.env` 配置，密码留空则随机生成并打印在启动日志中）。
- 健康检查：<http://localhost:8080/api/health>
- 数据库文件位于 `Online-Store-server/data/store.db`，删除后重启即可重置数据（商品种子仅在首次启动写入，之后由后台维护）。
- 端口、前端目录、管理员账号等可用环境变量或 `.env` 覆盖，见 `Online-Store-server/.env.example`。

## 常用命令

在后端目录 `Online-Store-server/` 下：

```bash
make run     # 开发运行
make test    # go test ./...（含 API 集成测试）
make vet     # go vet ./...
make fmt     # gofmt 格式化
make build   # 交叉编译产物输出到 bin/server
```

## API 概览

| 方法 | 路径 | 认证 | 说明 |
| --- | --- | --- | --- |
| GET | `/api/products` | 否 | 商品列表（可按分类过滤） |
| GET | `/api/products/{id}` | 否 | 商品详情 |
| POST | `/api/auth/register` / `login` / `logout` | 否 | 注册 / 登录 / 退出 |
| GET | `/api/auth/me` | 是 | 当前用户 |
| GET/POST/PATCH/DELETE | `/api/cart…` | 是 | 购物车增删改与游客合并 |
| POST/GET | `/api/orders` | 是 | 下单与订单列表 |
| POST | `/api/admin/auth/login` | 否 | 管理员登录（独立限流） |
| GET | `/api/admin/stats` | 管理员 | 概览统计 + 最近订单 |
| GET/POST/PATCH/DELETE | `/api/admin/products…` | 管理员 | 商品管理 |
| GET/PATCH/DELETE | `/api/admin/users…` | 管理员 | 用户管理 |
| GET/PATCH/DELETE | `/api/admin/orders…` | 管理员 | 订单管理 |

完整字段说明、校验规则与 curl 示例见 [Online-Store-server/docs/API.md](Online-Store-server/docs/API.md)。

## 技术栈

- **前端**：HTML5、CSS3（Flexbox / Grid、CSS 变量、响应式）、原生 JavaScript、Font Awesome 6（CDN）
- **后端**：Go 标准库 `net/http`、`modernc.org/sqlite`（纯 Go 驱动）、`golang.org/x/crypto/bcrypt`、SQLite WAL
- **数据**：SQLite 单文件，金额统一用整数"分"存储；商品种子数据来自 `internal/database/seed_products.json`（仅首次启动写入，之后由后台维护）

## 维护约定

- **商品维护**：日常通过后台管理；种子文件仅在首次启动（或重建数据库）时写入，之后修改它不会影响已有商品。
- **新增页面**：引入 `base.css`、`nav.css`、`footer.css` 与 `js/api.js`、`js/auth-nav.js`；修改全站导航或页脚时，六个商城页面需同步更新（静态站点无模板机制）；后台两页（`admin.html` / `admin-login.html`）不加载商城导航与页脚，独立维护。
- **边界说明**：未接入真实支付、库存扣减与邮件验证；订单状态为 `pending/paid/shipped/done/cancelled` 五种，由管理员在后台调整；商品图片由前端 `images/` 静态目录提供，后台仅编辑图片相对路径，不做上传。

## 许可证

本项目采用 [MIT 许可证](LICENSE)。
