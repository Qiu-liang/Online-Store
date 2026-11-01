// Package api 实现 HTTP 路由与处理器。
package api

import (
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"online-store/server/internal/config"
	"online-store/server/internal/store"
)

// Server 聚合配置、存储与路由。
type Server struct {
	cfg              *config.Config
	logger           *slog.Logger
	users            *store.UserStore
	sessions         *store.SessionStore
	products         *store.ProductStore
	carts            *store.CartStore
	orders           *store.OrderStore
	stats            *store.StatsStore
	authLimiter      *rateLimiter // 客户登录/注册限流
	adminAuthLimiter *rateLimiter // 管理员登录独立限流
}

// NewServer 组装 Server。
func NewServer(cfg *config.Config, logger *slog.Logger, users *store.UserStore, sessions *store.SessionStore,
	products *store.ProductStore, carts *store.CartStore, orders *store.OrderStore,
	stats *store.StatsStore) *Server {
	return &Server{
		cfg:              cfg,
		logger:           logger,
		users:            users,
		sessions:         sessions,
		products:         products,
		carts:            carts,
		orders:           orders,
		stats:            stats,
		authLimiter:      newRateLimiter(10, 15*time.Minute),
		adminAuthLimiter: newRateLimiter(10, 15*time.Minute),
	}
}

// Handler 返回完整的 HTTP 处理器：API 路由 + 静态文件。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// 健康检查与目录
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/categories", s.handleListCategories)

	// 商品
	mux.HandleFunc("GET /api/products", s.handleListProducts)
	mux.HandleFunc("GET /api/products/{id}", s.handleGetProduct)

	// 认证
	mux.HandleFunc("POST /api/auth/register", s.withRateLimit(s.handleRegister))
	mux.HandleFunc("POST /api/auth/login", s.withRateLimit(s.handleLogin))
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/me", s.withAuth(s.handleMe))

	// 购物车
	mux.HandleFunc("GET /api/cart", s.withAuth(s.handleGetCart))
	mux.HandleFunc("POST /api/cart/items", s.withAuth(s.handleAddCartItem))
	mux.HandleFunc("PATCH /api/cart/items/{productId}", s.withAuth(s.handleUpdateCartItem))
	mux.HandleFunc("DELETE /api/cart/items/{productId}", s.withAuth(s.handleRemoveCartItem))
	mux.HandleFunc("POST /api/cart/merge", s.withAuth(s.handleMergeCart))

	// 订单
	mux.HandleFunc("POST /api/orders", s.withAuth(s.handleCreateOrder))
	mux.HandleFunc("GET /api/orders", s.withAuth(s.handleListOrders))
	mux.HandleFunc("GET /api/orders/{id}", s.withAuth(s.handleGetOrder))

	// 后台认证（独立限流）
	mux.HandleFunc("POST /api/admin/auth/login", s.withAdminRateLimit(s.handleAdminLogin))
	mux.HandleFunc("GET /api/admin/auth/me", s.withAdmin(s.handleAdminMe))

	// 后台概览
	mux.HandleFunc("GET /api/admin/stats", s.withAdmin(s.handleAdminStats))

	// 后台商品
	mux.HandleFunc("GET /api/admin/products", s.withAdmin(s.handleAdminListProducts))
	mux.HandleFunc("POST /api/admin/products", s.withAdmin(s.handleAdminCreateProduct))
	mux.HandleFunc("GET /api/admin/products/{id}", s.withAdmin(s.handleAdminGetProduct))
	mux.HandleFunc("PATCH /api/admin/products/{id}", s.withAdmin(s.handleAdminUpdateProduct))
	mux.HandleFunc("DELETE /api/admin/products/{id}", s.withAdmin(s.handleAdminDeleteProduct))

	// 后台用户
	mux.HandleFunc("GET /api/admin/users", s.withAdmin(s.handleAdminListUsers))
	mux.HandleFunc("PATCH /api/admin/users/{id}", s.withAdmin(s.handleAdminUpdateRole))
	mux.HandleFunc("POST /api/admin/users/{id}/password", s.withAdmin(s.handleAdminResetPassword))
	mux.HandleFunc("DELETE /api/admin/users/{id}", s.withAdmin(s.handleAdminDeleteUser))

	// 后台订单
	mux.HandleFunc("GET /api/admin/orders", s.withAdmin(s.handleAdminListOrders))
	mux.HandleFunc("GET /api/admin/orders/{id}", s.withAdmin(s.handleAdminGetOrder))
	mux.HandleFunc("PATCH /api/admin/orders/{id}", s.withAdmin(s.handleAdminUpdateOrderStatus))
	mux.HandleFunc("DELETE /api/admin/orders/{id}", s.withAdmin(s.handleAdminDeleteOrder))

	// 未匹配的 API 路径返回 JSON 404
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
	})

	// /admin 302 跳转到 /admin.html，方便直接输入目录地址
	mux.HandleFunc("GET /admin", s.redirectToAdminHTML)
	mux.HandleFunc("GET /admin/", s.redirectToAdminHTML)

	// 静态前端
	mux.Handle("/", s.staticHandler())

	return s.withRecover(s.withSecureHeaders(s.withLogging(s.withSameOrigin(mux))))
}

// redirectToAdminHTML 把 /admin 与 /admin/ 重定向到管理后台页面。
func (s *Server) redirectToAdminHTML(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin.html", http.StatusFound)
}

// staticHandler 托管前端静态目录，并为不同资源设置缓存策略。
// 目录路径一律 404（根路径由 index.html 兜底），避免暴露目录列表。
func (s *Server) staticHandler() http.Handler {
	if _, err := os.Stat(s.cfg.FrontendDir); err != nil {
		s.logger.Warn("前端目录不存在，静态页面将不可用", "dir", s.cfg.FrontendDir)
	}
	fileServer := http.FileServer(http.Dir(s.cfg.FrontendDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			cleaned := path.Clean("/" + r.URL.Path)
			if info, err := os.Stat(filepath.Join(s.cfg.FrontendDir, filepath.FromSlash(cleaned))); err == nil && info.IsDir() {
				writeError(w, http.StatusNotFound, "not_found", "资源不存在")
				return
			}
		}
		ext := strings.ToLower(filepath.Ext(r.URL.Path))
		switch {
		case r.URL.Path == "/" || ext == ".html":
			w.Header().Set("Cache-Control", "no-cache")
		case ext == ".css" || ext == ".js":
			// 开发期页面经常改动，css/js 使用协商缓存（304）而非强缓存。
			w.Header().Set("Cache-Control", "no-cache")
		case ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" || ext == ".avif" || ext == ".svg":
			w.Header().Set("Cache-Control", "public, max-age=86400")
		}
		fileServer.ServeHTTP(w, r)
	})
}
