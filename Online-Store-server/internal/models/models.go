// Package models 定义 API 与存储层共用的数据结构。
package models

import "strconv"

// 用户角色。
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// IsValidRole 校验角色取值。
func IsValidRole(role string) bool {
	return role == RoleUser || role == RoleAdmin
}

// 订单状态：后台可在五种状态间自由切换（不强制状态机，便于人工纠错）。
const (
	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusShipped   = "shipped"
	OrderStatusDone      = "done"
	OrderStatusCancelled = "cancelled"
)

// OrderStatuses 是全部合法订单状态。
var OrderStatuses = []string{
	OrderStatusPending,
	OrderStatusPaid,
	OrderStatusShipped,
	OrderStatusDone,
	OrderStatusCancelled,
}

// IsValidOrderStatus 校验订单状态取值。
func IsValidOrderStatus(status string) bool {
	for _, allowed := range OrderStatuses {
		if status == allowed {
			return true
		}
	}
	return false
}

// User 是脱敏后的用户信息（不含密码哈希）。
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

// Category 是商品分类。
type Category struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	SortOrder int64  `json:"-"`
}

// Product 是商品完整信息。PriceText 由 PriceCents + Unit 生成，用于直接展示。
// Active 为 false 表示已下架，商城侧不可见。
type Product struct {
	ID          string  `json:"id"`
	Category    string  `json:"category"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	PriceCents  int64   `json:"priceCents"`
	PriceText   string  `json:"priceText"`
	Unit        string  `json:"unit"`
	Image       string  `json:"image"`
	Detail      string  `json:"detail"`
	Origin      string  `json:"origin"`
	Storage     string  `json:"storage"`
	Nutrients   string  `json:"nutrients"`
	Care        string  `json:"care"`
	Color       string  `json:"color"`
	Rating      float64 `json:"rating"`
	Type        string  `json:"type"`
	Active      bool    `json:"active"`
	SortOrder   int64   `json:"sortOrder"`
}

// CartItem 是购物车中的一项，价格信息来自商品表当前数据。
type CartItem struct {
	ProductID      string `json:"productId"`
	Title          string `json:"title"`
	PriceCents     int64  `json:"priceCents"`
	PriceText      string `json:"priceText"`
	Unit           string `json:"unit"`
	Image          string `json:"image"`
	Quantity       int64  `json:"quantity"`
	LineTotalCents int64  `json:"lineTotalCents"`
}

// Cart 是服务端购物车聚合结果。
type Cart struct {
	Items         []CartItem `json:"items"`
	SubtotalCents int64      `json:"subtotalCents"`
	TotalCents    int64      `json:"totalCents"`
}

// MergeItem 是游客购物车合并请求中的一项。
type MergeItem struct {
	ProductID string `json:"productId"`
	Quantity  int64  `json:"quantity"`
}

// Shipping 是下单时的收货信息。
type Shipping struct {
	Name    string `json:"shippingName"`
	Phone   string `json:"shippingPhone"`
	Address string `json:"shippingAddress"`
}

// OrderItem 是订单中的商品快照（下单时的标题与单价）。
type OrderItem struct {
	ProductID      string `json:"productId"`
	Title          string `json:"title"`
	UnitPriceCents int64  `json:"unitPriceCents"`
	UnitPriceText  string `json:"unitPriceText"`
	Unit           string `json:"unit"`
	Image          string `json:"image"`
	Quantity       int64  `json:"quantity"`
}

// Order 是一笔订单。
type Order struct {
	ID              string      `json:"id"`
	UserID          int64       `json:"-"`
	TotalCents      int64       `json:"totalCents"`
	Status          string      `json:"status"`
	ShippingName    string      `json:"shippingName"`
	ShippingPhone   string      `json:"shippingPhone"`
	ShippingAddress string      `json:"shippingAddress"`
	CreatedAt       string      `json:"createdAt"`
	Items           []OrderItem `json:"items"`
}

// AdminUser 是后台用户列表条目：用户信息 + 订单聚合统计。
type AdminUser struct {
	User
	OrderCount      int64  `json:"orderCount"`
	TotalSpentCents int64  `json:"totalSpentCents"`
	LastOrderAt     string `json:"lastOrderAt"`
}

// AdminOrder 是后台视角的订单：带买家信息与商品件数。
type AdminOrder struct {
	ID              string      `json:"id"`
	UserID          int64       `json:"userId"`
	Username        string      `json:"username"`
	UserEmail       string      `json:"userEmail"`
	TotalCents      int64       `json:"totalCents"`
	Status          string      `json:"status"`
	ShippingName    string      `json:"shippingName"`
	ShippingPhone   string      `json:"shippingPhone"`
	ShippingAddress string      `json:"shippingAddress"`
	CreatedAt       string      `json:"createdAt"`
	ItemCount       int64       `json:"itemCount"`
	Items           []OrderItem `json:"items,omitempty"`
}

// AdminStats 是后台概览统计。销售额与用户消费均不含已取消订单。
type AdminStats struct {
	UserCount          int64 `json:"userCount"`
	AdminCount         int64 `json:"adminCount"`
	ProductCount       int64 `json:"productCount"`
	ActiveProductCount int64 `json:"activeProductCount"`
	OrderCount         int64 `json:"orderCount"`
	PendingOrderCount  int64 `json:"pendingOrderCount"`
	TodayOrderCount    int64 `json:"todayOrderCount"`
	RevenueCents       int64 `json:"revenueCents"`
}

// FormatPrice 把整数分格式化为展示文本，例如 2680 + "500g" -> "¥26.8/500g"。
func FormatPrice(cents int64, unit string) string {
	yuan := strconv.FormatFloat(float64(cents)/100, 'f', -1, 64)
	if unit == "" {
		return "¥" + yuan
	}
	return "¥" + yuan + "/" + unit
}
