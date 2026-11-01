// Package models 定义 API 与存储层共用的数据结构。
package models

import "strconv"

// User 是脱敏后的用户信息（不含密码哈希）。
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
}

// Category 是商品分类。
type Category struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	SortOrder int64  `json:"-"`
}

// Product 是商品完整信息。PriceText 由 PriceCents + Unit 生成，用于直接展示。
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

// FormatPrice 把整数分格式化为展示文本，例如 2680 + "500g" -> "¥26.8/500g"。
func FormatPrice(cents int64, unit string) string {
	yuan := strconv.FormatFloat(float64(cents)/100, 'f', -1, 64)
	if unit == "" {
		return "¥" + yuan
	}
	return "¥" + yuan + "/" + unit
}
