package api

import (
	"errors"
	"net/http"

	"online-store/server/internal/models"
	"online-store/server/internal/store"
)

type productResponse struct {
	Product *models.Product `json:"product"`
}

type adminProductListResponse struct {
	Products []models.Product `json:"products"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}

// handleAdminListProducts 后台商品列表：query/category/active + 分页。
func (s *Server) handleAdminListProducts(w http.ResponseWriter, r *http.Request) {
	page := parsePagination(r)
	filter := store.ProductFilter{
		Query:    r.URL.Query().Get("query"),
		Category: r.URL.Query().Get("category"),
		Active:   r.URL.Query().Get("active"),
		Limit:    page.PageSize,
		Offset:   page.offset(),
	}

	products, total, err := s.products.AdminList(r.Context(), filter)
	if err != nil {
		s.logger.Error("后台查询商品失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, adminProductListResponse{
		Products: products, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}

// handleAdminCreateProduct 新增商品。
func (s *Server) handleAdminCreateProduct(w http.ResponseWriter, r *http.Request) {
	var req adminProductRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeFieldError(w, err)
		return
	}

	var product models.Product
	if err := req.applyToCreate(&product); err != nil {
		writeFieldError(w, err)
		return
	}
	if product.Category == "" {
		writeError(w, http.StatusBadRequest, "invalid_category", "请选择商品分类")
		return
	}
	exists, err := s.products.CategoryExists(r.Context(), product.Category)
	if err != nil {
		s.logger.Error("查询分类失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	if !exists {
		writeError(w, http.StatusBadRequest, "invalid_category", "分类不存在")
		return
	}

	if err := s.products.Create(r.Context(), &product); err != nil {
		s.logger.Error("新增商品失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	created, err := s.products.AdminGetByID(r.Context(), product.ID)
	if err != nil {
		s.logger.Error("查询新商品失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	s.logAdminAction(r, "product.create", created.ID)
	writeJSON(w, http.StatusCreated, productResponse{Product: created})
}

// handleAdminGetProduct 后台商品详情（含下架商品）。
func (s *Server) handleAdminGetProduct(w http.ResponseWriter, r *http.Request) {
	product, err := s.products.AdminGetByID(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "product_not_found", "商品不存在")
		return
	}
	if err != nil {
		s.logger.Error("后台查询商品失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, productResponse{Product: product})
}

// handleAdminUpdateProduct 编辑商品：只更新请求中出现的字段。
func (s *Server) handleAdminUpdateProduct(w http.ResponseWriter, r *http.Request) {
	var req adminProductRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeFieldError(w, err)
		return
	}

	product, err := s.products.AdminGetByID(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "product_not_found", "商品不存在")
		return
	}
	if err != nil {
		s.logger.Error("后台查询商品失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}

	if err := req.applyToPatch(product); err != nil {
		writeFieldError(w, err)
		return
	}
	if product.Category == "" {
		writeError(w, http.StatusBadRequest, "invalid_category", "请选择商品分类")
		return
	}
	exists, err := s.products.CategoryExists(r.Context(), product.Category)
	if err != nil {
		s.logger.Error("查询分类失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	if !exists {
		writeError(w, http.StatusBadRequest, "invalid_category", "分类不存在")
		return
	}

	updated, err := s.products.Update(r.Context(), product)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "product_not_found", "商品不存在")
		return
	}
	if err != nil {
		s.logger.Error("更新商品失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	s.logAdminAction(r, "product.update", updated.ID)
	writeJSON(w, http.StatusOK, productResponse{Product: updated})
}

// handleAdminDeleteProduct 删除商品：cart_items 级联删除，order_items 快照保留。
func (s *Server) handleAdminDeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.products.Delete(r.Context(), id); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "product_not_found", "商品不存在")
		return
	} else if err != nil {
		s.logger.Error("删除商品失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	s.logAdminAction(r, "product.delete", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
