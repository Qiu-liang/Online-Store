package api

import (
	"path"
	"strings"
	"unicode/utf8"

	"online-store/server/internal/models"
)

// 允许的商品类型与图片扩展名。
var (
	productTypes        = []string{"蔬菜", "水果", "花卉"}
	productImageExtList = []string{".jpg", ".jpeg", ".png", ".webp", ".avif", ".svg"}
)

// 金额上限：100000000 分 = ¥1,000,000。
const maxPriceCents = 100_000_000

// adminProductRequest 是后台商品创建/编辑请求。
// 指针字段用于区分“未提供”与“零值”：PATCH 只应用出现的字段，避免历史数据阻断上下架。
type adminProductRequest struct {
	Title       *string  `json:"title"`
	Category    *string  `json:"category"`
	PriceCents  *int64   `json:"priceCents"`
	Unit        *string  `json:"unit"`
	Image       *string  `json:"image"`
	Description *string  `json:"description"`
	Detail      *string  `json:"detail"`
	Origin      *string  `json:"origin"`
	Storage     *string  `json:"storage"`
	Nutrients   *string  `json:"nutrients"`
	Care        *string  `json:"care"`
	Color       *string  `json:"color"`
	Rating      *float64 `json:"rating"`
	Type        *string  `json:"type"`
	Active      *bool    `json:"active"`
	SortOrder   *int64   `json:"sortOrder"`
}

// applyToCreate 校验创建请求并生成完整商品；必填字段缺失时返回字段错误。
func (req *adminProductRequest) applyToCreate(product *models.Product) error {
	if req.Title == nil {
		return errInvalidField("title", "商品标题为必填项")
	}
	if req.Category == nil {
		return errInvalidField("category", "商品分类为必填项")
	}
	if req.PriceCents == nil {
		return errInvalidField("priceCents", "商品价格为必填项")
	}
	if req.Unit == nil {
		return errInvalidField("unit", "商品单位为必填项")
	}
	if req.Image == nil {
		return errInvalidField("image", "商品图片为必填项")
	}
	if req.Type == nil {
		return errInvalidField("type", "商品类型为必填项")
	}

	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	detail := ""
	if req.Detail != nil {
		detail = *req.Detail
	}
	rating := 0.0
	if req.Rating != nil {
		rating = *req.Rating
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	var sortOrder int64 // 0 表示追加到末尾（由 store 层处理）
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	*product = models.Product{
		Title:       *req.Title,
		Category:    *req.Category,
		PriceCents:  *req.PriceCents,
		Unit:        *req.Unit,
		Image:       *req.Image,
		Description: description,
		Detail:      detail,
		Origin:      optionalString(req.Origin),
		Storage:     optionalString(req.Storage),
		Nutrients:   optionalString(req.Nutrients),
		Care:        optionalString(req.Care),
		Color:       optionalString(req.Color),
		Rating:      rating,
		Type:        *req.Type,
		Active:      active,
		SortOrder:   sortOrder,
	}
	return validateProductFields(product, productFieldProvided{
		title: true, price: true, unit: true, image: true, type_: true,
		rating: req.Rating != nil, sort: req.SortOrder != nil,
	})
}

// applyToPatch 把请求中出现的字段覆盖到现有商品上，再只针对出现的字段做校验，
// 避免历史数据中的空字段阻断上下架等小改动。
func (req *adminProductRequest) applyToPatch(product *models.Product) error {
	if req.Title != nil {
		product.Title = *req.Title
	}
	if req.Category != nil {
		product.Category = *req.Category
	}
	if req.PriceCents != nil {
		product.PriceCents = *req.PriceCents
	}
	if req.Unit != nil {
		product.Unit = *req.Unit
	}
	if req.Image != nil {
		product.Image = *req.Image
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Detail != nil {
		product.Detail = *req.Detail
	}
	if req.Origin != nil {
		product.Origin = *req.Origin
	}
	if req.Storage != nil {
		product.Storage = *req.Storage
	}
	if req.Nutrients != nil {
		product.Nutrients = *req.Nutrients
	}
	if req.Care != nil {
		product.Care = *req.Care
	}
	if req.Color != nil {
		product.Color = *req.Color
	}
	if req.Rating != nil {
		product.Rating = *req.Rating
	}
	if req.Type != nil {
		product.Type = *req.Type
	}
	if req.Active != nil {
		product.Active = *req.Active
	}
	if req.SortOrder != nil {
		product.SortOrder = *req.SortOrder
	}
	return validateProductFields(product, productFieldProvided{
		title:  req.Title != nil,
		unit:   req.Unit != nil,
		image:  req.Image != nil,
		type_:  req.Type != nil,
		rating: req.Rating != nil,
		sort:   req.SortOrder != nil,
	})
}

// productFieldProvided 标记哪些字段需要校验（创建时全部必填，PATCH 时仅出现的字段）。
type productFieldProvided struct {
	title  bool
	price  bool
	unit   bool
	image  bool
	type_  bool
	rating bool
	sort   bool
}

// validateProductFields 按需校验商品字段。
func validateProductFields(product *models.Product, provided productFieldProvided) error {
	product.Title = strings.TrimSpace(product.Title)
	if provided.title {
		if length := utf8.RuneCountInString(product.Title); length < 1 || length > 80 {
			return errInvalidField("title", "商品标题需要 1-80 个字符")
		}
	}
	if provided.unit {
		if length := utf8.RuneCountInString(strings.TrimSpace(product.Unit)); length < 1 || length > 20 {
			return errInvalidField("unit", "商品单位需要 1-20 个字符")
		}
	}
	if provided.type_ && !isValidProductType(product.Type) {
		return errInvalidField("type", "商品类型只支持：蔬菜、水果、花卉")
	}
	if provided.price && (product.PriceCents < 0 || product.PriceCents > maxPriceCents) {
		return errInvalidField("priceCents", "商品价格需要在 ¥0 - ¥1,000,000 之间")
	}
	if provided.image {
		product.Image = strings.TrimSpace(product.Image)
		if !isValidProductImage(product.Image) {
			return errInvalidField("image", "图片必须为站内相对路径，扩展名限 jpg/jpeg/png/webp/avif/svg")
		}
	}
	if length := utf8.RuneCountInString(product.Description); length > 200 {
		return errInvalidField("description", "商品简介最多 200 个字符")
	}
	if length := utf8.RuneCountInString(product.Detail); length > 2000 {
		return errInvalidField("detail", "商品详情最多 2000 个字符")
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{"origin", product.Origin},
		{"storage", product.Storage},
		{"nutrients", product.Nutrients},
		{"care", product.Care},
		{"color", product.Color},
	} {
		if length := utf8.RuneCountInString(field.value); length > 200 {
			return errInvalidField(field.name, "最多 200 个字符")
		}
	}
	if provided.rating && (product.Rating < 0 || product.Rating > 5) {
		return errInvalidField("rating", "评分需要在 0-5 之间")
	}
	if provided.sort && product.SortOrder < 0 {
		return errInvalidField("sortOrder", "排序值不能为负数")
	}
	return nil
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func isValidProductType(productType string) bool {
	for _, allowed := range productTypes {
		if productType == allowed {
			return true
		}
	}
	return false
}

// isValidProductImage 校验图片路径：相对路径、无外链/目录穿越/引号与控制字符，扩展名白名单。
func isValidProductImage(image string) bool {
	if length := utf8.RuneCountInString(image); length < 1 || length > 300 {
		return false
	}
	if strings.HasPrefix(image, "/") || strings.Contains(image, "://") || strings.Contains(image, "..") {
		return false
	}
	for _, r := range image {
		// 禁止空白、控制字符与引号，避免 URL 解析与属性逃逸问题。
		if r <= ' ' || r == 0x7f || r == '"' || r == '\'' || r == '\\' {
			return false
		}
	}
	clean := path.Clean(image)
	if clean == "." || clean != image {
		return false
	}
	ext := strings.ToLower(path.Ext(image))
	for _, allowed := range productImageExtList {
		if ext == allowed {
			return true
		}
	}
	return false
}
