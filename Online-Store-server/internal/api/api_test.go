package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"online-store/server/internal/auth"
	"online-store/server/internal/config"
	"online-store/server/internal/database"
	"online-store/server/internal/store"
)

// testClient 包装 httptest.Server，自动携带会话 Cookie。
type testClient struct {
	t      *testing.T
	server *httptest.Server
	client *http.Client
}

// testDB 持有测试环境的数据库与存储，便于测试直接构造管理员等数据。
type testDB struct {
	db       *sql.DB
	users    *store.UserStore
	orders   *store.OrderStore
	products *store.ProductStore
}

var currentDB *testDB

func newTestEnv(t *testing.T) *testClient {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("初始化测试数据库: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	currentDB = &testDB{
		db:       db,
		users:    store.NewUserStore(db),
		orders:   store.NewOrderStore(db),
		products: store.NewProductStore(db),
	}

	cfg := &config.Config{
		Port:            "0",
		FrontendDir:     newStaticDir(t),
		DBPath:          "unused",
		SessionTTL:      time.Hour,
		AdminSessionTTL: time.Hour,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	server := NewServer(
		cfg,
		logger,
		currentDB.users,
		store.NewSessionStore(db),
		currentDB.products,
		store.NewCartStore(db),
		currentDB.orders,
		store.NewStatsStore(db),
	)
	ts := httptest.NewServer(server.Handler())
	t.Cleanup(ts.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("创建 CookieJar: %v", err)
	}
	return &testClient{t: t, server: ts, client: &http.Client{Jar: jar}}
}

// newClient 基于同一个测试服务器创建独立 Cookie 会话的客户端，
// 用于在同一环境里同时模拟普通用户与管理员。
func (c *testClient) newClient() *testClient {
	c.t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		c.t.Fatalf("创建 CookieJar: %v", err)
	}
	return &testClient{t: c.t, server: c.server, client: &http.Client{Jar: jar}}
}

// createAdmin 直接在数据库中创建管理员账号。
func (c *testClient) createAdmin(email, password string) {
	c.t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		c.t.Fatalf("哈希管理员密码: %v", err)
	}
	if _, err := currentDB.users.CreateAdmin(context.Background(), "管理员", email, hash); err != nil {
		c.t.Fatalf("创建管理员: %v", err)
	}
}

// createUser 直接在数据库中创建普通用户账号。
func (c *testClient) createUser(email, password string) {
	c.t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		c.t.Fatalf("哈希用户密码: %v", err)
	}
	if _, err := currentDB.users.Create(context.Background(), "用户", email, hash); err != nil {
		c.t.Fatalf("创建用户: %v", err)
	}
}

// newStaticDir 创建一个带 index.html 与 js/app.js 的临时前端目录。
func newStaticDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html><title>ok</title>"), 0o644); err != nil {
		t.Fatalf("写入测试首页: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "js"), 0o755); err != nil {
		t.Fatalf("创建测试 js 目录: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "js", "app.js"), []byte("window.t=1;\n"), 0o644); err != nil {
		t.Fatalf("写入测试脚本: %v", err)
	}
	return dir
}

// get 发送不带请求体的 GET，返回状态码、响应头与原始响应体。
func (c *testClient) get(path string) (int, http.Header, string) {
	c.t.Helper()
	res, err := c.client.Get(c.server.URL + path)
	if err != nil {
		c.t.Fatalf("GET %s 请求失败: %v", path, err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatalf("读取响应: %v", err)
	}
	return res.StatusCode, res.Header, string(raw)
}

// TestStaticServingAndSecurityHeaders 验证静态托管与安全响应头。
func TestStaticServingAndSecurityHeaders(t *testing.T) {
	c := newTestEnv(t)

	// 根路径返回 index.html
	status, header, body := c.get("/")
	if status != http.StatusOK || !strings.Contains(body, "<!doctype html>") {
		t.Fatalf("根路径应返回 index.html, 实际: %d %q", status, body)
	}

	// 具体静态文件可访问
	status, _, body = c.get("/js/app.js")
	if status != http.StatusOK || !strings.Contains(body, "window.t=1") {
		t.Fatalf("静态脚本应可访问, 实际: %d %q", status, body)
	}

	// 目录路径一律 404，禁止目录列表
	for _, dirPath := range []string{"/js/", "/js"} {
		status, _, _ = c.get(dirPath)
		if status != http.StatusNotFound {
			t.Fatalf("目录路径 %s 应返回 404, 实际: %d", dirPath, status)
		}
	}

	// 安全响应头
	status, header, _ = c.get("/api/health")
	if status != http.StatusOK {
		t.Fatalf("健康检查状态码: %d", status)
	}
	for _, name := range []string{
		"Content-Security-Policy", "X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy",
	} {
		if header.Get(name) == "" {
			t.Fatalf("缺少安全响应头 %s", name)
		}
	}
}

// do 发送 JSON 请求并返回状态码与解析后的响应体。
func (c *testClient) do(method, path string, body any) (int, map[string]any) {
	c.t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("序列化请求体: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.server.URL+path, reader)
	if err != nil {
		c.t.Fatalf("构造请求: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.client.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s 请求失败: %v", method, path, err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatalf("读取响应: %v", err)
	}
	payload := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			c.t.Fatalf("解析响应 JSON: %v (%s)", err, raw)
		}
	}
	return res.StatusCode, payload
}

func TestHealthAndCatalog(t *testing.T) {
	c := newTestEnv(t)

	status, body := c.do(http.MethodGet, "/api/health", nil)
	if status != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("健康检查异常: %d %v", status, body)
	}

	status, body = c.do(http.MethodGet, "/api/products", nil)
	if status != http.StatusOK {
		t.Fatalf("商品列表状态码: %d", status)
	}
	products, ok := body["products"].([]any)
	if !ok || len(products) != 10 {
		t.Fatalf("期望 10 个种子商品, 实际: %d", len(products))
	}

	status, body = c.do(http.MethodGet, "/api/products?category=fruit", nil)
	if status != http.StatusOK {
		t.Fatalf("分类过滤状态码: %d", status)
	}
	products, _ = body["products"].([]any)
	if len(products) != 3 {
		t.Fatalf("期望 3 个水果, 实际: %d", len(products))
	}

	status, _ = c.do(http.MethodGet, "/api/products/veg-1", nil)
	if status != http.StatusOK {
		t.Fatalf("商品详情状态码: %d", status)
	}

	status, _ = c.do(http.MethodGet, "/api/products/not-exist", nil)
	if status != http.StatusNotFound {
		t.Fatalf("不存在的商品应返回 404, 实际: %d", status)
	}

	if status, _ := c.do(http.MethodGet, "/api/cart", nil); status != http.StatusUnauthorized {
		t.Fatalf("未登录访问购物车应返回 401, 实际: %d", status)
	}

	if status, _ := c.do(http.MethodGet, "/api/unknown", nil); status != http.StatusNotFound {
		t.Fatalf("未知接口应返回 404, 实际: %d", status)
	}
}

func TestRegisterLoginCartOrder(t *testing.T) {
	c := newTestEnv(t)

	// 注册并自动登录
	status, body := c.do(http.MethodPost, "/api/auth/register", map[string]any{
		"username": "小明", "email": "ming@example.com", "password": "password123",
	})
	if status != http.StatusCreated {
		t.Fatalf("注册状态码: %d %v", status, body)
	}
	user, ok := body["user"].(map[string]any)
	if !ok || user["username"] != "小明" {
		t.Fatalf("注册响应缺少用户: %v", body)
	}

	// 重复邮箱 → 409
	status, _ = c.do(http.MethodPost, "/api/auth/register", map[string]any{
		"username": "另一个人", "email": "ming@example.com", "password": "password123",
	})
	if status != http.StatusConflict {
		t.Fatalf("重复邮箱应返回 409, 实际: %d", status)
	}

	// 错误密码 → 401
	status, _ = c.do(http.MethodPost, "/api/auth/login", map[string]any{
		"email": "ming@example.com", "password": "wrong-password",
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("错误密码应返回 401, 实际: %d", status)
	}

	// 正确登录
	status, _ = c.do(http.MethodPost, "/api/auth/login", map[string]any{
		"email": "MING@example.com ", "password": "password123",
	})
	if status != http.StatusOK {
		t.Fatalf("登录状态码: %d", status)
	}

	// 加购两个番茄（¥26.8 × 2 = ¥53.6）
	status, body = c.do(http.MethodPost, "/api/cart/items", map[string]any{
		"productId": "veg-1", "quantity": 2,
	})
	if status != http.StatusOK {
		t.Fatalf("加入购物车状态码: %d %v", status, body)
	}
	cart := body["cart"].(map[string]any)
	if cart["subtotalCents"].(float64) != 5360 {
		t.Fatalf("小计应为 5360, 实际: %v", cart["subtotalCents"])
	}

	// 不存在的商品 → 404
	status, _ = c.do(http.MethodPost, "/api/cart/items", map[string]any{"productId": "nope"})
	if status != http.StatusNotFound {
		t.Fatalf("无效商品应返回 404, 实际: %d", status)
	}

	// 修改数量为 3 → ¥80.4
	status, body = c.do(http.MethodPatch, "/api/cart/items/veg-1", map[string]any{"quantity": 3})
	if status != http.StatusOK {
		t.Fatalf("修改数量状态码: %d", status)
	}
	cart = body["cart"].(map[string]any)
	if cart["subtotalCents"].(float64) != 8040 {
		t.Fatalf("小计应为 8040, 实际: %v", cart["subtotalCents"])
	}

	// 非法数量 → 400
	status, _ = c.do(http.MethodPatch, "/api/cart/items/veg-1", map[string]any{"quantity": 0})
	if status != http.StatusBadRequest {
		t.Fatalf("非法数量应返回 400, 实际: %d", status)
	}

	// 删除 → 购物车为空
	status, body = c.do(http.MethodDelete, "/api/cart/items/veg-1", nil)
	if status != http.StatusOK {
		t.Fatalf("删除商品状态码: %d", status)
	}
	cart = body["cart"].(map[string]any)
	if cart["subtotalCents"].(float64) != 0 {
		t.Fatalf("删除后小计应为 0, 实际: %v", cart["subtotalCents"])
	}

	// 合并游客购物车：两盒蓝莓
	status, body = c.do(http.MethodPost, "/api/cart/merge", map[string]any{
		"items": []map[string]any{{"productId": "fruit-1", "quantity": 2}},
	})
	if status != http.StatusOK {
		t.Fatalf("合并购物车状态码: %d", status)
	}
	cart = body["cart"].(map[string]any)
	if cart["subtotalCents"].(float64) != 11600 {
		t.Fatalf("合并后小计应为 11600, 实际: %v", cart["subtotalCents"])
	}

	// 下单
	status, body = c.do(http.MethodPost, "/api/orders", map[string]any{
		"shippingName": "小明", "shippingPhone": "13800138000", "shippingAddress": "上海市静安区南京西路 100 号",
	})
	if status != http.StatusCreated {
		t.Fatalf("下单状态码: %d %v", status, body)
	}
	order := body["order"].(map[string]any)
	orderID, _ := order["id"].(string)
	if !strings.HasPrefix(orderID, "ORD-") {
		t.Fatalf("订单号格式异常: %q", orderID)
	}
	if order["totalCents"].(float64) != 11600 {
		t.Fatalf("订单总额应为 11600, 实际: %v", order["totalCents"])
	}

	// 下单后购物车应清空
	status, body = c.do(http.MethodGet, "/api/cart", nil)
	if status != http.StatusOK {
		t.Fatalf("查询购物车状态码: %d", status)
	}
	cart = body["cart"].(map[string]any)
	if cart["subtotalCents"].(float64) != 0 {
		t.Fatalf("下单后购物车应为空, 实际: %v", cart)
	}

	// 订单列表与详情
	status, body = c.do(http.MethodGet, "/api/orders", nil)
	if status != http.StatusOK {
		t.Fatalf("订单列表状态码: %d", status)
	}
	orders, _ := body["orders"].([]any)
	if len(orders) != 1 {
		t.Fatalf("期望 1 笔订单, 实际: %d", len(orders))
	}

	status, _ = c.do(http.MethodGet, "/api/orders/"+orderID, nil)
	if status != http.StatusOK {
		t.Fatalf("订单详情状态码: %d", status)
	}

	// 空购物车下单 → 400
	status, _ = c.do(http.MethodPost, "/api/orders", map[string]any{
		"shippingName": "小明", "shippingPhone": "13800138000", "shippingAddress": "上海市静安区南京西路 100 号",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("空购物车下单应返回 400, 实际: %d", status)
	}

	// 退出登录 → me 返回 401
	status, _ = c.do(http.MethodPost, "/api/auth/logout", nil)
	if status != http.StatusOK {
		t.Fatalf("退出状态码: %d", status)
	}
	status, _ = c.do(http.MethodGet, "/api/auth/me", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("退出后 me 应返回 401, 实际: %d", status)
	}
}

func TestRegisterValidation(t *testing.T) {
	c := newTestEnv(t)

	cases := []struct {
		name   string
		body   map[string]any
		expect string
	}{
		{"密码过短", map[string]any{"username": "小明", "email": "a@b.com", "password": "short"}, "invalid_password"},
		{"邮箱非法", map[string]any{"username": "小明", "email": "not-an-email", "password": "password123"}, "invalid_email"},
		{"用户名过短", map[string]any{"username": "A", "email": "a@b.com", "password": "password123"}, "invalid_username"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			status, body := c.do(http.MethodPost, "/api/auth/register", testCase.body)
			if status != http.StatusBadRequest {
				t.Fatalf("应返回 400, 实际: %d", status)
			}
			errBody, _ := body["error"].(map[string]any)
			if errBody["code"] != testCase.expect {
				t.Fatalf("错误码应为 %s, 实际: %v", testCase.expect, errBody["code"])
			}
		})
	}

	// 非法 JSON → 400
	req, err := http.NewRequest(http.MethodPost, c.server.URL+"/api/auth/register", strings.NewReader("{bad json"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("非法 JSON 应返回 400, 实际: %d", res.StatusCode)
	}
}
