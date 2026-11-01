package api

import (
	"net/http"
	"strings"
	"testing"
)

// loginAdmin 用指定邮箱密码登录管理员并保持会话。
func loginAdmin(t *testing.T, c *testClient, email, password string) {
	t.Helper()
	status, body := c.do(http.MethodPost, "/api/admin/auth/login", map[string]any{
		"email": email, "password": password,
	})
	if status != http.StatusOK {
		t.Fatalf("管理员登录应返回 200, 实际: %d %v", status, body)
	}
}

// adminFixtures 创建一名管理员与一名普通用户，返回两个独立会话客户端。
func adminFixtures(t *testing.T) (admin *testClient, user *testClient) {
	t.Helper()
	c := newTestEnv(t)
	c.createAdmin("admin@test.local", "admin-pass-123")
	c.createUser("ming@example.com", "user-pass-123")

	admin = c.newClient()
	user = c.newClient()
	loginAdmin(t, admin, "admin@test.local", "admin-pass-123")

	status, _ := user.do(http.MethodPost, "/api/auth/login", map[string]any{
		"email": "ming@example.com", "password": "user-pass-123",
	})
	if status != http.StatusOK {
		t.Fatalf("用户登录应返回 200, 实际: %d", status)
	}
	return admin, user
}

// TestAdminAuthGuard 验证后台认证与权限边界。
func TestAdminAuthGuard(t *testing.T) {
	c := newTestEnv(t)
	c.createUser("ming@example.com", "user-pass-123")

	// 未登录访问后台 → 401
	if status, _ := c.do(http.MethodGet, "/api/admin/stats", nil); status != http.StatusUnauthorized {
		t.Fatalf("未登录访问后台应返回 401, 实际: %d", status)
	}

	// 普通用户登录后访问后台 → 403 not_admin
	status, body := c.do(http.MethodPost, "/api/auth/login", map[string]any{
		"email": "ming@example.com", "password": "user-pass-123",
	})
	if status != http.StatusOK {
		t.Fatalf("用户登录失败: %d", status)
	}
	status, body = c.do(http.MethodGet, "/api/admin/stats", nil)
	if status != http.StatusForbidden {
		t.Fatalf("普通用户访问后台应返回 403, 实际: %d", status)
	}
	if errBody, _ := body["error"].(map[string]any); errBody["code"] != "not_admin" {
		t.Fatalf("错误码应为 not_admin, 实际: %v", errBody["code"])
	}

	// 普通用户调用管理员登录 → 403，且不覆盖已有客户会话
	status, _ = c.do(http.MethodPost, "/api/admin/auth/login", map[string]any{
		"email": "ming@example.com", "password": "user-pass-123",
	})
	if status != http.StatusForbidden {
		t.Fatalf("普通用户调用管理员登录应返回 403, 实际: %d", status)
	}
	if status, _ = c.do(http.MethodGet, "/api/auth/me", nil); status != http.StatusOK {
		t.Fatalf("原有客户会话应保持有效, 实际: %d", status)
	}

	// 管理员登录 → 200 + role=admin；me 可用；响应带 no-store
	admin, _ := adminFixtures(t)
	status, body = admin.do(http.MethodGet, "/api/admin/auth/me", nil)
	if status != http.StatusOK {
		t.Fatalf("管理员 me 应返回 200, 实际: %d", status)
	}
	me, _ := body["user"].(map[string]any)
	if me["role"] != "admin" {
		t.Fatalf("me 应返回 role=admin, 实际: %v", me)
	}
	_, header, _ := admin.get("/api/admin/stats")
	if header.Get("Cache-Control") != "no-store" {
		t.Fatalf("后台响应应包含 Cache-Control: no-store, 实际: %q", header.Get("Cache-Control"))
	}
}

// TestAdminProductLifecycle 覆盖商品创建/编辑/上下架/删除与商城联动。
func TestAdminProductLifecycle(t *testing.T) {
	admin, user := adminFixtures(t)

	// 创建商品 → 201，ID 以 prd- 开头
	status, body := admin.do(http.MethodPost, "/api/admin/products", map[string]any{
		"title": "有机生菜", "category": "vegetable", "priceCents": 1280, "unit": "500g",
		"image": "images/Products/tomato.jpg", "description": "清晨采摘",
		"type": "蔬菜", "rating": 4.8,
	})
	if status != http.StatusCreated {
		t.Fatalf("创建商品应返回 201, 实际: %d %v", status, body)
	}
	product, _ := body["product"].(map[string]any)
	id, _ := product["id"].(string)
	if !strings.HasPrefix(id, "prd-") {
		t.Fatalf("商品ID应以 prd- 开头, 实际: %q", id)
	}

	// 商城列表可见
	status, body = user.do(http.MethodGet, "/api/products", nil)
	products, _ := body["products"].([]any)
	if status != http.StatusOK || len(products) != 11 {
		t.Fatalf("商城应看到 11 个商品, 实际: %d", len(products))
	}

	// PATCH 编辑价格/标题/评分
	status, body = admin.do(http.MethodPatch, "/api/admin/products/"+id, map[string]any{
		"priceCents": 1580, "title": "有机生菜王", "rating": 4.9,
	})
	if status != http.StatusOK {
		t.Fatalf("编辑商品应返回 200, 实际: %d %v", status, body)
	}
	product, _ = body["product"].(map[string]any)
	if product["priceCents"].(float64) != 1580 || product["title"] != "有机生菜王" {
		t.Fatalf("编辑未生效: %v", product)
	}

	// 下架后商城不可见
	status, _ = admin.do(http.MethodPatch, "/api/admin/products/"+id, map[string]any{"active": false})
	if status != http.StatusOK {
		t.Fatalf("下架应返回 200, 实际: %d", status)
	}
	status, body = user.do(http.MethodGet, "/api/products", nil)
	products, _ = body["products"].([]any)
	if status != http.StatusOK || len(products) != 10 {
		t.Fatalf("下架后商城应看到 10 个商品, 实际: %d", len(products))
	}
	if status, _ = user.do(http.MethodGet, "/api/products/"+id, nil); status != http.StatusNotFound {
		t.Fatalf("下架商品详情应返回 404, 实际: %d", status)
	}

	// 下架商品不可加购
	status, _ = user.do(http.MethodPost, "/api/cart/items", map[string]any{"productId": id, "quantity": 1})
	if status != http.StatusNotFound {
		t.Fatalf("下架商品加购应返回 404, 实际: %d", status)
	}

	// 下架后重新上架
	if status, _ = admin.do(http.MethodPatch, "/api/admin/products/"+id, map[string]any{"active": true}); status != http.StatusOK {
		t.Fatalf("上架应返回 200, 实际: %d", status)
	}
	if status, _ = user.do(http.MethodGet, "/api/products/"+id, nil); status != http.StatusOK {
		t.Fatalf("上架后商品详情应可见, 实际: %d", status)
	}

	// 列表搜索 / 分类 / 状态筛选
	status, body = admin.do(http.MethodGet, "/api/admin/products?query=%E7%94%9F%E8%8F%9C", nil)
	products, _ = body["products"].([]any)
	if status != http.StatusOK || len(products) != 1 {
		t.Fatalf("搜索应命中 1 个商品, 实际: %d", len(products))
	}
	status, body = admin.do(http.MethodGet, "/api/admin/products?category=fruit", nil)
	_, total := body["products"].([]any)
	if status != http.StatusOK || body["total"].(float64) != 3 {
		t.Fatalf("分类筛选 total 应为 3, 实际: %v", body["total"])
	}
	_ = total
	status, body = admin.do(http.MethodGet, "/api/admin/products?active=true&pageSize=5", nil)
	if status != http.StatusOK || body["total"].(float64) != 11 || body["pageSize"].(float64) != 5 {
		t.Fatalf("状态筛选与分页异常: %v", body)
	}

	// 删除商品 → 购物车条目级联删除，历史订单明细保留
	// 先加购并下单
	if status, _ = user.do(http.MethodPost, "/api/cart/items", map[string]any{"productId": id, "quantity": 2}); status != http.StatusOK {
		t.Fatalf("加购失败: %d", status)
	}
	status, body = user.do(http.MethodPost, "/api/orders", map[string]any{
		"shippingName": "小明", "shippingPhone": "13800138000", "shippingAddress": "上海市静安区南京西路 100 号",
	})
	if status != http.StatusCreated {
		t.Fatalf("下单应成功, 实际: %d %v", status, body)
	}
	if status, _ = user.do(http.MethodPost, "/api/cart/items", map[string]any{"productId": id, "quantity": 1}); status != http.StatusOK {
		t.Fatalf("再次加购失败: %d", status)
	}
	if status, _ = admin.do(http.MethodDelete, "/api/admin/products/"+id, nil); status != http.StatusOK {
		t.Fatalf("删除商品应返回 200, 实际: %d", status)
	}
	if status, _ = admin.do(http.MethodDelete, "/api/admin/products/"+id, nil); status != http.StatusNotFound {
		t.Fatalf("重复删除应返回 404, 实际: %d", status)
	}
	// 购物车条目级联删除
	status, body = user.do(http.MethodGet, "/api/cart", nil)
	cart, _ := body["cart"].(map[string]any)
	if status != http.StatusOK || len(cart["items"].([]any)) != 0 {
		t.Fatalf("删除商品后购物车应为空, 实际: %v", cart)
	}
	// 历史订单明细保留
	status, body = user.do(http.MethodGet, "/api/orders", nil)
	orders, _ := body["orders"].([]any)
	if status != http.StatusOK || len(orders) != 1 {
		t.Fatalf("历史订单应保留, 实际: %d", len(orders))
	}
	order, _ := orders[0].(map[string]any)
	items, _ := order["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("订单快照明细应保留, 实际: %d", len(items))
	}
}

// TestAdminProductValidation 覆盖商品字段校验。
func TestAdminProductValidation(t *testing.T) {
	admin, _ := adminFixtures(t)

	cases := []struct {
		name   string
		body   map[string]any
		expect string
	}{
		{"负价格", map[string]any{"title": "x", "category": "vegetable", "priceCents": -1, "unit": "500g", "image": "images/Products/tomato.jpg", "type": "蔬菜"}, "invalid_priceCents"},
		{"非法分类", map[string]any{"title": "x", "category": "nope", "priceCents": 100, "unit": "500g", "image": "images/Products/tomato.jpg", "type": "蔬菜"}, "invalid_category"},
		{"非法图片", map[string]any{"title": "x", "category": "vegetable", "priceCents": 100, "unit": "500g", "image": "https://evil.com/a.jpg", "type": "蔬菜"}, "invalid_image"},
		{"目录穿越图片", map[string]any{"title": "x", "category": "vegetable", "priceCents": 100, "unit": "500g", "image": "images/../secret.jpg", "type": "蔬菜"}, "invalid_image"},
		{"非法类型", map[string]any{"title": "x", "category": "vegetable", "priceCents": 100, "unit": "500g", "image": "images/Products/tomato.jpg", "type": "电子数码"}, "invalid_type"},
		{"评分越界", map[string]any{"title": "x", "category": "vegetable", "priceCents": 100, "unit": "500g", "image": "images/Products/tomato.jpg", "type": "蔬菜", "rating": 5.5}, "invalid_rating"},
		{"未知字段", map[string]any{"title": "x", "category": "vegetable", "priceCents": 100, "unit": "500g", "image": "images/Products/tomato.jpg", "type": "蔬菜", "hack": 1}, "invalid_request"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			status, body := admin.do(http.MethodPost, "/api/admin/products", testCase.body)
			if status != http.StatusBadRequest {
				t.Fatalf("应返回 400, 实际: %d %v", status, body)
			}
			errBody, _ := body["error"].(map[string]any)
			if errBody["code"] != testCase.expect {
				t.Fatalf("错误码应为 %s, 实际: %v", testCase.expect, errBody["code"])
			}
		})
	}
}

// TestAdminUsers 覆盖用户列表、角色调整、重置密码与删除。
func TestAdminUsers(t *testing.T) {
	admin, user := adminFixtures(t)

	// 让用户产生订单统计
	if status, _ := user.do(http.MethodPost, "/api/cart/items", map[string]any{"productId": "veg-1", "quantity": 2}); status != http.StatusOK {
		t.Fatalf("加购失败: %d", status)
	}
	status, body := user.do(http.MethodPost, "/api/orders", map[string]any{
		"shippingName": "小明", "shippingPhone": "13800138000", "shippingAddress": "上海市静安区南京西路 100 号",
	})
	if status != http.StatusCreated {
		t.Fatalf("下单失败: %d %v", status, body)
	}

	// 用户列表搜索与统计
	status, body = admin.do(http.MethodGet, "/api/admin/users?query=ming", nil)
	if status != http.StatusOK {
		t.Fatalf("用户列表失败: %d", status)
	}
	users, _ := body["users"].([]any)
	if len(users) != 1 {
		t.Fatalf("搜索应命中 1 个用户, 实际: %d", len(users))
	}
	entry, _ := users[0].(map[string]any)
	if entry["orderCount"].(float64) != 1 {
		t.Fatalf("订单数应为 1, 实际: %v", entry["orderCount"])
	}
	if entry["totalSpentCents"].(float64) != 5360 {
		t.Fatalf("累计消费应为 5360, 实际: %v", entry["totalSpentCents"])
	}

	// 角色筛选
	status, body = admin.do(http.MethodGet, "/api/admin/users?role=admin", nil)
	users, _ = body["users"].([]any)
	if status != http.StatusOK || len(users) != 1 {
		t.Fatalf("管理员筛选应命中 1 人, 实际: %d", len(users))
	}

	// 唯一管理员降级自己 → 409 last_admin（最后管理员保护优先于自助操作保护）
	status, body = admin.do(http.MethodPatch, "/api/admin/users/1", map[string]any{"role": "user"})
	if status != http.StatusConflict {
		t.Fatalf("唯一管理员降级应返回 409, 实际: %d %v", status, body)
	}
	if errBody, _ := body["error"].(map[string]any); errBody["code"] != "last_admin" {
		t.Fatalf("错误码应为 last_admin, 实际: %v", errBody["code"])
	}

	// 提升普通用户为管理员 → 200
	status, _ = admin.do(http.MethodPatch, "/api/admin/users/2", map[string]any{"role": "admin"})
	if status != http.StatusOK {
		t.Fatalf("提升角色应返回 200, 实际: %d", status)
	}

	// 有两名管理员时改自己 → 400 self_operation_forbidden
	status, body = admin.do(http.MethodPatch, "/api/admin/users/1", map[string]any{"role": "user"})
	if status != http.StatusBadRequest {
		t.Fatalf("改自己角色应返回 400, 实际: %d %v", status, body)
	}
	if errBody, _ := body["error"].(map[string]any); errBody["code"] != "self_operation_forbidden" {
		t.Fatalf("错误码应为 self_operation_forbidden, 实际: %v", errBody["code"])
	}

	// 角色变更实时生效：普通用户的旧会话立即获得后台权限
	if status, _ = user.do(http.MethodGet, "/api/admin/stats", nil); status != http.StatusOK {
		t.Fatalf("提升后旧会话应可访问后台, 实际: %d", status)
	}

	// 降级另一名管理员 → 200（仍剩一名管理员）
	status, _ = admin.do(http.MethodPatch, "/api/admin/users/2", map[string]any{"role": "user"})
	if status != http.StatusOK {
		t.Fatalf("降级另一名管理员应返回 200, 实际: %d", status)
	}
	// 被降级的管理员会话无需重新登录即失效 → 403
	if status, _ = user.do(http.MethodGet, "/api/admin/stats", nil); status != http.StatusForbidden {
		t.Fatalf("降级后的会话应返回 403, 实际: %d", status)
	}

	// 重置密码：旧密码失败、新密码成功、旧会话失效
	if status, _ = admin.do(http.MethodPost, "/api/admin/users/2/password", map[string]any{"password": "new-pass-456"}); status != http.StatusOK {
		t.Fatalf("重置密码应返回 200, 实际: %d", status)
	}
	if status, _ = user.do(http.MethodGet, "/api/auth/me", nil); status != http.StatusUnauthorized {
		t.Fatalf("重置后旧会话应失效, 实际: %d", status)
	}
	if status, _ = user.do(http.MethodPost, "/api/auth/login", map[string]any{
		"email": "ming@example.com", "password": "user-pass-123",
	}); status != http.StatusUnauthorized {
		t.Fatalf("旧密码应失效, 实际: %d", status)
	}
	if status, _ = user.do(http.MethodPost, "/api/auth/login", map[string]any{
		"email": "ming@example.com", "password": "new-pass-456",
	}); status != http.StatusOK {
		t.Fatalf("新密码应可登录, 实际: %d", status)
	}

	// 删除唯一管理员（自己）→ 409 last_admin
	status, _ = admin.do(http.MethodDelete, "/api/admin/users/1", nil)
	if status != http.StatusConflict {
		t.Fatalf("删除唯一管理员应返回 409, 实际: %d", status)
	}

	// 有两名管理员时删除自己 → 400
	if status, _ = admin.do(http.MethodPatch, "/api/admin/users/2", map[string]any{"role": "admin"}); status != http.StatusOK {
		t.Fatalf("再次提升应返回 200, 实际: %d", status)
	}
	status, _ = admin.do(http.MethodDelete, "/api/admin/users/1", nil)
	if status != http.StatusBadRequest {
		t.Fatalf("删除自己应返回 400, 实际: %d", status)
	}

	// 删除另一名管理员：会话失效、登录失败、订单级联消失
	if status, _ = admin.do(http.MethodDelete, "/api/admin/users/2", nil); status != http.StatusOK {
		t.Fatalf("删除用户应返回 200, 实际: %d", status)
	}
	if status, _ = user.do(http.MethodGet, "/api/auth/me", nil); status != http.StatusUnauthorized {
		t.Fatalf("被删除用户的会话应失效, 实际: %d", status)
	}
	if status, _ = user.do(http.MethodPost, "/api/auth/login", map[string]any{
		"email": "ming@example.com", "password": "new-pass-456",
	}); status != http.StatusUnauthorized {
		t.Fatalf("被删除用户登录应 401, 实际: %d", status)
	}
	status, body = admin.do(http.MethodGet, "/api/admin/orders", nil)
	if status != http.StatusOK || body["total"].(float64) != 0 {
		t.Fatalf("被删除用户的订单应级联删除, 实际: %v", body["total"])
	}
}

// TestAdminOrders 覆盖订单列表、筛选、状态修改与删除。
func TestAdminOrders(t *testing.T) {
	admin, user := adminFixtures(t)

	// 用户下单两笔
	for _, quantity := range []int64{2, 1} {
		if status, _ := user.do(http.MethodPost, "/api/cart/items", map[string]any{"productId": "veg-1", "quantity": quantity}); status != http.StatusOK {
			t.Fatalf("加购失败: %d", status)
		}
		status, body := user.do(http.MethodPost, "/api/orders", map[string]any{
			"shippingName": "小明", "shippingPhone": "13800138000", "shippingAddress": "上海市静安区南京西路 100 号",
		})
		if status != http.StatusCreated {
			t.Fatalf("下单失败: %d %v", status, body)
		}
	}

	// 后台列表可见买家信息
	status, body := admin.do(http.MethodGet, "/api/admin/orders", nil)
	if status != http.StatusOK {
		t.Fatalf("订单列表失败: %d", status)
	}
	orders, _ := body["orders"].([]any)
	if len(orders) != 2 {
		t.Fatalf("应有 2 笔订单, 实际: %d", len(orders))
	}
	first, _ := orders[0].(map[string]any)
	if first["username"] != "用户" || first["userEmail"] != "ming@example.com" {
		t.Fatalf("买家信息缺失: %v", first)
	}
	if first["itemCount"].(float64) != 1 {
		t.Fatalf("商品件数应为 1, 实际: %v", first["itemCount"])
	}
	orderID, _ := first["id"].(string)

	// 详情含明细
	status, body = admin.do(http.MethodGet, "/api/admin/orders/"+orderID, nil)
	if status != http.StatusOK {
		t.Fatalf("订单详情失败: %d", status)
	}
	detail, _ := body["order"].(map[string]any)
	items, _ := detail["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("详情应含明细, 实际: %v", detail)
	}

	// 状态筛选
	status, body = admin.do(http.MethodGet, "/api/admin/orders?status=pending", nil)
	if status != http.StatusOK || body["total"].(float64) != 2 {
		t.Fatalf("状态筛选应命中 2 笔, 实际: %v", body["total"])
	}

	// userId 筛选与非法 userId
	status, body = admin.do(http.MethodGet, "/api/admin/orders?userId=2", nil)
	if status != http.StatusOK || body["total"].(float64) != 2 {
		t.Fatalf("userId 筛选应命中 2 笔, 实际: %v", body["total"])
	}
	if status, _ = admin.do(http.MethodGet, "/api/admin/orders?userId=abc", nil); status != http.StatusBadRequest {
		t.Fatalf("非法 userId 应返回 400, 实际: %d", status)
	}

	// 修改状态：合法成功，非法 400
	status, body = admin.do(http.MethodPatch, "/api/admin/orders/"+orderID, map[string]any{"status": "shipped"})
	if status != http.StatusOK {
		t.Fatalf("修改状态应返回 200, 实际: %d %v", status, body)
	}
	if status, _ = admin.do(http.MethodPatch, "/api/admin/orders/"+orderID, map[string]any{"status": "flying"}); status != http.StatusBadRequest {
		t.Fatalf("非法状态应返回 400, 实际: %d", status)
	}

	// 客户侧看到新状态
	status, body = user.do(http.MethodGet, "/api/orders", nil)
	orders, _ = body["orders"].([]any)
	statuses := map[string]bool{}
	for _, raw := range orders {
		entry, _ := raw.(map[string]any)
		statuses[entry["status"].(string)] = true
	}
	if !statuses["shipped"] {
		t.Fatalf("客户侧应看到 shipped 状态: %v", statuses)
	}

	// 删除订单 → 客户列表减少
	if status, _ = admin.do(http.MethodDelete, "/api/admin/orders/"+orderID, nil); status != http.StatusOK {
		t.Fatalf("删除订单应返回 200, 实际: %d", status)
	}
	status, body = user.do(http.MethodGet, "/api/orders", nil)
	orders, _ = body["orders"].([]any)
	if status != http.StatusOK || len(orders) != 1 {
		t.Fatalf("删除后客户应剩 1 笔订单, 实际: %d", len(orders))
	}
	if status, _ = admin.do(http.MethodDelete, "/api/admin/orders/"+orderID, nil); status != http.StatusNotFound {
		t.Fatalf("重复删除应返回 404, 实际: %d", status)
	}
}

// TestAdminStats 覆盖概览统计。
func TestAdminStats(t *testing.T) {
	admin, user := adminFixtures(t)

	// 用户下一笔订单
	if status, _ := user.do(http.MethodPost, "/api/cart/items", map[string]any{"productId": "veg-1", "quantity": 2}); status != http.StatusOK {
		t.Fatalf("加购失败: %d", status)
	}
	if status, _ := user.do(http.MethodPost, "/api/orders", map[string]any{
		"shippingName": "小明", "shippingPhone": "13800138000", "shippingAddress": "上海市静安区南京西路 100 号",
	}); status != http.StatusCreated {
		t.Fatalf("下单失败")
	}

	status, body := admin.do(http.MethodGet, "/api/admin/stats", nil)
	if status != http.StatusOK {
		t.Fatalf("概览失败: %d", status)
	}
	stats, _ := body["stats"].(map[string]any)
	if stats["userCount"].(float64) != 2 {
		t.Fatalf("用户数应为 2, 实际: %v", stats["userCount"])
	}
	if stats["adminCount"].(float64) != 1 {
		t.Fatalf("管理员数应为 1, 实际: %v", stats["adminCount"])
	}
	if stats["productCount"].(float64) != 10 || stats["activeProductCount"].(float64) != 10 {
		t.Fatalf("商品统计异常: %v", stats)
	}
	if stats["orderCount"].(float64) != 1 || stats["pendingOrderCount"].(float64) != 1 || stats["todayOrderCount"].(float64) != 1 {
		t.Fatalf("订单统计异常: %v", stats)
	}
	if stats["revenueCents"].(float64) != 5360 {
		t.Fatalf("销售额应为 5360, 实际: %v", stats["revenueCents"])
	}
	recent, _ := body["recentOrders"].([]any)
	if len(recent) != 1 {
		t.Fatalf("最近订单应为 1, 实际: %d", len(recent))
	}
}

// TestAdminCartEmptyAfterInactive 下架全部商品后下单 → 400 cart_empty。
func TestAdminCartEmptyAfterInactive(t *testing.T) {
	admin, user := adminFixtures(t)

	if status, _ := user.do(http.MethodPost, "/api/cart/items", map[string]any{"productId": "veg-1", "quantity": 1}); status != http.StatusOK {
		t.Fatalf("加购失败: %d", status)
	}
	// 下架 veg-1：购物车隐藏该商品，下单变成空购物车
	status, body := admin.do(http.MethodPatch, "/api/admin/products/veg-1", map[string]any{"active": false})
	if status != http.StatusOK {
		t.Fatalf("下架失败: %d %v", status, body)
	}
	status, body = user.do(http.MethodGet, "/api/cart", nil)
	cart, _ := body["cart"].(map[string]any)
	if status != http.StatusOK || len(cart["items"].([]any)) != 0 {
		t.Fatalf("下架商品应从购物车隐藏, 实际: %v", cart)
	}
	status, body = user.do(http.MethodPost, "/api/orders", map[string]any{
		"shippingName": "小明", "shippingPhone": "13800138000", "shippingAddress": "上海市静安区南京西路 100 号",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("全部下架后下单应返回 400, 实际: %d %v", status, body)
	}
	if errBody, _ := body["error"].(map[string]any); errBody["code"] != "cart_empty" {
		t.Fatalf("错误码应为 cart_empty, 实际: %v", errBody["code"])
	}
}

// TestAdminRedirect 验证 /admin 302 跳转（不跟随重定向）。
func TestAdminRedirect(t *testing.T) {
	c := newTestEnv(t)

	req, err := http.NewRequest(http.MethodGet, c.server.URL+"/admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	res, err := noRedirect.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusFound {
		t.Fatalf("/admin 应返回 302, 实际: %d", res.StatusCode)
	}
	if location := res.Header.Get("Location"); location != "/admin.html" {
		t.Fatalf("跳转目标应为 /admin.html, 实际: %q", location)
	}
}
