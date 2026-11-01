# 绿源农业 · Online Store

一个中文电商网站项目：原生 HTML/CSS/JavaScript 前端 + Go 后端（SQLite）。展示农产品、花卉、关于我们、购物车、登录注册与订单页面；商品、账号、购物车、订单均由后端 API 提供；另附同源管理后台（商品 / 用户 / 订单）。

## 项目组成

| 目录 | 说明 |
| --- | --- |
| `Online-Store-main/` | 前端静态站点（HTML/CSS/原生 JS） |
| `Online-Store-server/` | Go 后端：REST API + 静态托管 + SQLite 存储 |

## 页面

| 页面 | 文件 | 说明 |
| --- | --- | --- |
| 首页 | `index.html` | 品牌主视觉、卖点卡片、当季精选 |
| 产品中心 | `products.html` | 从 API 加载商品，分类展示、详情弹窗、加入购物车 |
| 关于我们 | `about.html` | 发展历程、农场数据、核心团队 |
| 购物车 | `shopping.html` | 数量增减、删除、金额汇总、结算下单 |
| 登录注册 | `login.html` | 登录 / 注册，成功后合并游客购物车 |
| 我的账户 | `account.html` | 用户信息、订单列表、退出登录 |
| 管理后台 | `admin.html` | 概览 / 商品 / 用户 / 订单管理（仅管理员） |
| 后台登录 | `admin-login.html` | 管理员登录（与商城登录独立限流） |

## 目录结构

```text
Online-Store-main/
├── index.html               # 首页
├── products.html            # 产品中心
├── about.html               # 关于我们
├── shopping.html            # 购物车
├── login.html               # 登录 / 注册
├── account.html             # 我的账户
├── admin.html               # 管理后台（hash 路由：概览/商品/用户/订单）
├── admin-login.html         # 管理员登录
├── css/
│   ├── base.css             # 设计变量、全局重置、通用按钮与表单控件
│   ├── nav.css              # 顶部导航（含登录态入口）
│   ├── footer.css           # 全站页脚
│   ├── index.css / products.css / about.css
│   ├── shopping.css         # 购物车 + 结算表单
│   ├── login.css            # 登录 / 注册
│   ├── account.css          # 账户与订单
│   └── admin.css            # 管理后台（侧边栏、表格、弹窗、登录卡片）
├── js/
│   ├── api.js               # 后端 API 封装（fetch + 错误处理）
│   ├── auth-nav.js          # 导航登录态（登录 ↔ 我的）
│   ├── cart-store.js        # 购物车双模式（游客 localStorage / 登录走服务端）
│   ├── index.js / about.js  # 视差、数字动画等页面效果
│   ├── products.js          # 商品列表、详情弹窗、加入购物车
│   ├── shopping.js          # 购物车渲染与结算
│   ├── login.js             # 登录 / 注册提交
│   ├── account.js           # 账户与订单展示
│   ├── admin.js             # 管理后台单页应用
│   └── admin-login.js       # 后台登录页逻辑
└── images/                    # 商品图、头图、团队头像等

Online-Store-server/         # Go 后端（详见其 README.md）
├── cmd/server/              # 启动入口
├── internal/                # config / database / auth / models / store / api
└── data/                    # 运行时 SQLite 数据库
```

## 本地运行

需要 Go 1.26+（首次可用 `brew install go`，与 `go.mod` 声明一致）。

```bash
cd Online-Store-server
go mod tidy
go run ./cmd/server
```

打开 <http://localhost:8080/> 即可浏览完整站点（后端同时托管前端静态文件）。

- 商品、购物车、登录注册、订单需要后端运行；`index.html`、`about.html` 直接双击打开仍可查看静态内容。
- 数据库默认写入 `Online-Store-server/data/store.db`，删除该文件即可重置数据。
- 商品数据由 `Online-Store-server/internal/database/seed_products.json` 提供，仅首次启动写入 SQLite；之后通过后台维护。
- 管理后台入口：<http://localhost:8080/admin>；初始管理员由 `ADMIN_EMAIL` / `ADMIN_PASSWORD` 配置（密码留空则随机生成并打印在启动日志）。
- 修改端口或路径：复制 `Online-Store-server/.env.example` 为 `.env`，详见后端 README。

## 功能说明

- **产品中心**：`GET /api/products` 加载商品与分类；详情弹窗、评分星级、加入购物车。
- **购物车**：游客数据保存在 `localStorage`（键 `greenCart`）；登录后走服务端 `/api/cart`，价格以后端商品表为准；跨标签页同步在游客模式下仍通过 `storage` 事件生效。
- **登录注册**：`/api/auth/*`；注册成功即登录；登录后自动把游客购物车合并到服务端。
- **订单**：购物车页填写收货信息提交 `POST /api/orders`，成功后跳转账户页查看订单。
- **管理后台**：`admin.html` 单页应用，hash 路由切换概览 / 商品管理 / 用户管理 / 订单管理；所有动态文本统一 `escapeHtml` 后插入，无内联脚本。
- **后台入口**：登录用户 `role=admin` 时，商城导航会自动显示"后台"链接（`js/auth-nav.js`）。
- **关于我们**：`js/about.js` 实现数字递增（保留 `+`、`%`、`亩` 后缀）、视差滚动和团队卡片倾斜。

## 维护约定

- 商品维护：日常通过管理后台增删改；种子文件仅在首次启动（或重建数据库）时写入，之后修改它不影响已有商品。
- 新增页面：引用 `base.css`、`nav.css`、`footer.css`，并在 `js/api.js`、`js/auth-nav.js` 之后引入页面脚本；导航中的登录入口使用 `<span class="nav-account" id="nav-account">` 占位。
- 公共样式（颜色、字体、`.modern-cta`、表单控件）放在 `base.css`，页面样式不要重复定义导航和页脚。
- 修改全站导航或页脚时，六个商城页面需同步更新（静态站点无模板机制）；后台两页（`admin.html` / `admin-login.html`）不加载商城导航与页脚，独立维护。
- 后端接口与安全细节见 `Online-Store-server/README.md`。

## 技术栈

- 前端：HTML5、CSS3（Flexbox / Grid、CSS 变量、响应式）、原生 JavaScript（无框架、无构建步骤）、Font Awesome 6（CDN）
- 后端：Go 标准库 `net/http`、`modernc.org/sqlite`（纯 Go）、`golang.org/x/crypto/bcrypt`、SQLite WAL

## 许可证

本项目采用 [MIT 许可证](../LICENSE)。
